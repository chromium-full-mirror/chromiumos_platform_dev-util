// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// To run:
// `go run loganalyzer.go --bad <directory containing bad reports> \
// --good <directory containing good reports> [--bad-bar <integer 0 to 100>] \
// [--good-bar <integer 0 to 100>]
//
// An example:
// `go run loganalyzer.go --bad ./bad --good ./good`
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// normalizationRegxp represents the information needed to replace an
// environment specific regex pattern from a log line e.g. timestamps.
type normalizationRegxp struct {
	// The description of the normalization logic, must be in format <desc>
	// e.g. <mac_addr>. Avoid numbers or any other names that could be matched
	// by the subsequent regex patterns.
	description string

	// The regexp for the normalization, use non capturing groups for
	// faster performance. This is because the regex engine wouldn't have
	// the overhead of allocating any memory to store the matched groups.
	regexp *regexp.Regexp
}

var (
	infectedFlag    = flag.String("bad", "", "Directory containing bad files")
	nonInfectedFlag = flag.String("good", "", "Directory containing good files")
	sigFlag         = flag.Int("bad-bar", 80, "Minimum percentage of bad reports having a line to be considered worthy")
	nonSigFlag      = flag.Int("good-bar", 20, "Maximum percentage of good reports having a matchingline to be considered worthy")
	debug           = flag.Bool("debug", false, "Output debug information")
	// Normalization Regexes. This is order sensitive, use the more specific regex
	// e.g. IPV6 address on top and more generic ones e.g. decimals at the bottom.
	// If not observed, the regex pattern will invalidate the subsequent ones.
	// Make sure the description does not cause matches with subsequent regexes,
	// e.g. <ipv6_addr> will cause a match with number regex.
	normalizationRegexps = []normalizationRegxp{
		{
			description: "<timestamp_letter>",
			regexp:      regexp.MustCompile(`(?:(Sat|Sun|Mon|Tue|Wed|Thu|Fri|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|UTC|PDT|PST)[,\s]+)`),
		},
		{
			// IPV6 address
			description: "<ipv_addr>",
			regexp:      regexp.MustCompile(`(?:([A-Fa-f0-9]{1,4}::?){1,7}[A-Fa-f0-9]{1,4})`),
		},
		{
			description: "<mac_addr>",
			regexp:      regexp.MustCompile(`(?:([A-Fa-f0-9]{2}[:-]){5}[A-Fa-f0-9]{2})`),
		},
		{
			description: "<http_addr>",
			regexp:      regexp.MustCompile(`(?:https?\:\S*)`),
		},
		{
			description: "<local_dir>",
			regexp:      regexp.MustCompile(`(?:\/[user|usr|tmp|dev|home|run|devices|lib|root]+\/\S*)`),
		},
		{
			description: "<drone_name>",
			regexp:      regexp.MustCompile(`(?:chromeos\d+-row\d+-\S*|chrome-bot@\S*)`),
		},
		{
			// Group any generic key, value pairs, e.g. SSID=<val>.
			description: "<key_val_pair>",
			regexp:      regexp.MustCompile(`(?:SSID=\S*)`),
		},
		{
			// hexdumps mainly in net.log files for connectivity tests.
			description: "<hex_dump>",
			regexp:      regexp.MustCompile(`(?:hexdump\(len=.*)`),
		},
		// TODO(b/288896588): Evaluate whether this generic pattern requires
		// cleaned up.
		{
			description: "<key>",
			regexp:      regexp.MustCompile(`(?:\w{32,})`),
		},
		{
			// Hexadecimal values can cause matches with actual words like
			// "beef" so we look at certain prefixes (e.g. 0x) or
			// postfixes (e.g. _) to limit the mismatches on shorter words.
			description: "<hex_val>",
			regexp:      regexp.MustCompile(`(?:0x[A-Fa-f0-9]+|[A-Fa-f0-9x]{8,}|[A-Fa-f0-9]{4,}[\-\_\'\]+|\:[A-Fa-f0-9]{4,})`),
		},
		{
			description: "<number>",
			regexp:      regexp.MustCompile(`(?:(-)?[0-9]+)`),
		},
	}
)

type fileInfo struct {
	// Map for mapping uint64 hash of a normalized line to the number
	// of files containing this unique line.
	hash map[uint64]int
	// Map for mapping uint64 hashes to the actual un-normalized line.
	revHash map[uint64]string
}

// normalize removes the environment specific patterns from the line, e.g. IDs.
func normalize(line string) string {
	normalized := line
	for _, re := range normalizationRegexps {
		normalized = re.regexp.ReplaceAllString(normalized, re.description)
	}
	return normalized
}

// hash provides a 64-bit hash of the line string.
func hash(line string) uint64 {
	f := fnv.New64()
	f.Write([]byte(line))
	return f.Sum64()
}

// readfile reads either .zip or .txt files and returns the
// []byte content.
func readFile(fpath string) ([]byte, error) {
	if strings.HasSuffix(fpath, ".txt") {
		return ioutil.ReadFile(fpath)
	} else if strings.HasSuffix(fpath, ".zip") {
		zipReader, err := zip.OpenReader(fpath)
		if err != nil {
			log.Fatal(err)
		}
		defer zipReader.Close()
		file := zipReader.File[0]
		zippedFile, err := file.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer zippedFile.Close()
		var buffer bytes.Buffer
		_, err = io.Copy(&buffer, zippedFile)
		if err != nil {
			log.Fatal(err)
		}
		return buffer.Bytes(), nil
	}
	return nil, fmt.Errorf("Unsupported file: %v", fpath)
}

// parseReport parses the system_logs.txt and fills the fileset.
func parseReport(fpath string, fileset map[string]*fileInfo) {
	content, err := readFile(fpath)
	if err != nil {
		log.Fatalf("Unable to read %v: %v", fpath, err)
	}

	re := regexp.MustCompile(`^(Profile\[.*\] )?(.*)=<multiline>$`)
	lines := strings.Split(string(content), "\n")

	for lc := 0; lc < len(lines); lc++ {
		line := lines[lc]
		if !re.MatchString(line) {
			continue
		}

		matches := re.FindStringSubmatch(line)
		fname := string(matches[2])
		fname = strings.ReplaceAll(fname, " ", "_")
		if _, ok := fileset[fname]; !ok {
			fileset[fname] = &fileInfo{make(map[uint64]int), make(map[uint64]string)}
		}
		file := fileset[fname]

		// map to ensure we don't count the same line again per file.
		umap := make(map[uint64]bool)

		for lc++; lc < len(lines); lc++ {
			line := lines[lc]
			if line == "---------- START ----------" {
				continue
			}
			if line == "---------- END ----------" {
				break
			}
			nline := normalize(line)
			h := hash(nline)
			if !umap[h] {
				file.hash[h]++
				file.revHash[h] = line
				umap[h] = true
			}
		}
	}
}

func reportFiles(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("Unable to read directory %v: %v", dir, err)
	}
	var logFiles []string
	for _, file := range files {
		fp := filepath.Join(dir, file.Name())
		logFiles = append(logFiles, fp)
	}
	return logFiles
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsDir()
}

func createLogs(infSet, nonInfSet map[string]*fileInfo, infSetCount, nonInfSetCount int) {
	outputDir := "extracted_logs"
	if isDir(outputDir) {
		log.Fatalf("%v already exists", outputDir)
	}
	err := os.Mkdir(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating directory %v: %v", outputDir, err)
	}
	for fname, finfo := range infSet {
		nonInfectedFInfo := nonInfSet[fname]
		var nonInfectedFinfoHashMap map[uint64]int
		if nonInfectedFInfo == nil {
			nonInfectedFinfoHashMap = make(map[uint64]int)
		} else {
			nonInfectedFinfoHashMap = nonInfectedFInfo.hash
		}

		var infectedLogs []string
		for hash, count := range finfo.hash {
			nonInfCount := nonInfectedFinfoHashMap[hash]
			if *debug {
				// filename: normalized-line: bad-count: good-count
				fmt.Printf("%s: %s: %d: %d\n", fname, normalize(finfo.revHash[hash]), count, nonInfCount)
			}
			if (count*100 > infSetCount*(*sigFlag)) && (nonInfCount*100 < nonInfSetCount*(*nonSigFlag)) {
				infectedLogs = append(infectedLogs, finfo.revHash[hash])
			}
		}

		if len(infectedLogs) == 0 {
			continue
		}

		logFile := filepath.Join(outputDir, fname)
		file, err := os.Create(logFile)
		if err != nil {
			log.Fatalf("Error creating file %v: %v", logFile, err)
		}
		defer file.Close()
		for _, str := range infectedLogs {
			_, err := file.WriteString(str + "\n")
			if err != nil {
				log.Fatalf("Error writing to file %v: %v", logFile, err)
			}
		}
	}
}

func main() {
	flag.Parse()
	infectedFiles := reportFiles(*infectedFlag)
	nonInfectedFiles := reportFiles(*nonInfectedFlag)
	infectedFileset, nonInfectedFileset := make(map[string]*fileInfo), make(map[string]*fileInfo)
	for _, file := range infectedFiles {
		parseReport(file, infectedFileset)
	}
	for _, file := range nonInfectedFiles {
		parseReport(file, nonInfectedFileset)
	}
	createLogs(infectedFileset, nonInfectedFileset, len(infectedFiles), len(nonInfectedFiles))
}
