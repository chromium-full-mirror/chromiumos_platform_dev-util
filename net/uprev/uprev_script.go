// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	cqCommitText = "\n" +
		"Cros-Add-Test-Suites: cellular_ota,cellular_ota_flaky\n" +
		"Cros-Add-TS-Boards-BuildTarget: trogdor,brya,octopus,nissa\n" +
		"Cros-Add-TS-Pool: cellular\n"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
	// Constants in the empty CL logic
	emptyCLRepoName   = "modemmanager-next"
	emptyCLBranchName = "uprev_empty_cl"
	emptyCLFileName   = "src/meson.build"
	emptyCLMsg        = "\n# EMPTY"
	emptyCLCommitMsg  = "EMPTY CL FOR UPREV PURPOSES: **DO NOT MERGE**\n" +
		"\nBUG=None\nFIXED=None\n\nTEST=NONE\n" + cqCommitText
)

func logPanic(s string) {
	fmt.Print(s)
	log.Panicln(s)
}

func runCmd(cmd *exec.Cmd, abortOnFail bool) (string, error) {
	fmt.Printf("Executing: %q\n", cmd.String())
	log.Printf("Executing: %q\n", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("err: %q\n", err)
		if abortOnFail {
			logPanic(err.Error())
		}
		log.Printf("Ignoring non-fatal error")
	}
	log.Printf("Execution result: %s\n", string(out))
	return string(out), err
}

// mergeUntilConflict merges the last non conflicting merge from the upstream branch
func mergeUntilConflict(upstreamBranch string) (string, error) {
	cmd := exec.Command("git", "log", "HEAD.."+upstreamBranch, "--abbrev", "--oneline", "--format=%h")
	out, _ := runCmd(cmd, true)
	if out == "" {
		return "", nil
	}

	commits := strings.Split(out, "\n")
	// The first element of this array is the most recent commit, and we're going backwards in time until we find one that merges cleanly
	for i, commitSHA := range commits[:len(commits)-1] {
		cmd := exec.Command("git", "merge", commitSHA)
		_, e := runCmd(cmd, false)
		if e != nil {
			cmd := exec.Command("git", "merge", "--abort")
			if _, err := runCmd(cmd, false); err != nil {
				return "", fmt.Errorf("Could not abort merge: %w", e)
			}
		} else {
			if i != 0 {
				fmt.Println(colorRed, "Could not merge cros/upstream fully. Merged only until commit: ", commitSHA, colorReset)
				fmt.Println(colorRed, "Warning: CQ does not support merge commits stacked on top of other commits. Your next commit needs to be a merge of the commit after", commitSHA, "without any commits in between.", colorReset)
			} else {
				fmt.Println(colorGreen, "Merged ", upstreamBranch, colorReset)
				genCommitMsg("cros/main", upstreamBranch, "", "None", upstreamBranch)
			}
			return fmt.Sprintf(commitSHA), nil
		}
	}
	fmt.Println(colorRed, "Could not merge any commit in cros/upstream", colorReset)
	return "", nil
}

// uprevRepo creates a new branch and attempts to git merge
func uprevRepo(rootDir string, repoName string, upstreamBranch string, lastTagFlag bool, commitSHA string, forceFlag bool) {
	uprevDate := time.Now().Format("01-02-2006")
	branchName := "merge-upstream-" + uprevDate
	fmt.Println(rootDir, repoName, upstreamBranch, uprevDate)
	if err := os.Chdir(filepath.Join(rootDir, repoName)); err != nil {
		logPanic(err.Error())
	}
	newDir, _ := os.Getwd()
	fmt.Printf(colorGreen+"Working on : %s\n"+colorReset, newDir)
	var cmd *exec.Cmd

	cmd = exec.Command("git", "status", "--porcelain")
	gitStatus, _ := runCmd(cmd, true)
	if gitStatus != "" {
		if forceFlag {
			cmd = exec.Command("git", "stash", "--include-untracked")
			runCmd(cmd, false)
			cmd = exec.Command("git", "reset", "--hard", "cros/main")
			runCmd(cmd, false)
		} else {
			logPanic("Please ensure that there are no uncommitted changes. Or run with --force=true")
		}
	}

	// Repo start a clean branch with HEAD cros/main and merge cros/upstream
	cmd = exec.Command("repo", "sync", "-d", ".")
	runCmd(cmd, true)
	if forceFlag {
		cmd = exec.Command("git", "branch", "-D", branchName)
	}
	runCmd(cmd, false)
	cmd = exec.Command("repo", "start", branchName)
	runCmd(cmd, true)
	cmd = exec.Command("git", "fetch", "cros", upstreamBranch)
	runCmd(cmd, true)
	targetSHA := ""
	if lastTagFlag {
		cmd = exec.Command("git", "describe", "--abbrev=0", commitSHA)
		gitOut, _ := runCmd(cmd, true)
		// git may throw warnings, but still have the last tag as the last but one line.
		gitOutArr := strings.Split(gitOut, "\n")
		targetSHA = gitOutArr[len(gitOutArr)-2]
	} else {
		targetSHA = commitSHA
	}
	_, err := mergeUntilConflict(targetSHA)
	if err != nil {
		logPanic(fmt.Errorf("Could not merge: %w", err).Error())
	}
}

// squash squashes all commits between the HEAD commit and base commit into the HEAD commit. The HEAD commit must be a merge.
func squash(baseSHA string) {
	if getWD() != "modemmanager-next" {
		logPanic("squash-merges are supported for MM only. Please run uprev_script inside ~/chromiumos/src/third_party/modemmanager-next/")
	}

	bkupDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	var cmd *exec.Cmd
	// store the merge conflict resolution
	cmd = exec.Command("cp", "-R", "../modemmanager-next", bkupDir)
	runCmd(cmd, true)
	cmd = exec.Command("rm", "-rf", bkupDir+"/modemmanager-next/.git")
	runCmd(cmd, true)

	// get the SHA of the parent commit from cros/upstream, and attempt to merge it again.
	cmd = exec.Command("git", "log", "--pretty=%p", "-n", "1", "HEAD")
	out, _ := runCmd(cmd, true)

	parentCommits := strings.Split(out, " ")
	if len(parentCommits) != 2 {
		logPanic("Cannot squash-merge since top commit is not a 2-way merge")
	}
	mergeSHA := strings.TrimSuffix(parentCommits[1], "\n")

	cmd = exec.Command("git", "reset", "--hard", baseSHA)
	runCmd(cmd, true)
	cmd = exec.Command("git", "merge", mergeSHA)
	// there will be merge conflicts, but we already have the resolution
	runCmd(cmd, false)
	fmt.Println("Please run the following commnand and press Enter")
	// TODO(pholla): Figure out why cmd = exec.Command("cp", "-R", logsDir+"/modemmanager-next/*", "./") fails
	fmt.Println("cp", "-r", "-f", bkupDir+"/modemmanager-next/*", "~/chromiumos/src/third_party/modemmanager-next/")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	cmd = exec.Command("git", "add", "-u")
	runCmd(cmd, true)
	cmd = exec.Command("git", "commit", "--no-edit")
	runCmd(cmd, true)
	genCommitMsg(baseSHA, mergeSHA, "", "None", mergeSHA)
}

// returns current working directory
func getWD() string {
	d, err := os.Getwd()
	if err != nil {
		logPanic(err.Error())
	}
	return filepath.Base(d)
}

func genCommitMsg(baseSHA string, mergeSHA string, cqDepend string, bugFlag string, lastTag string) {
	var cmd *exec.Cmd

	cmd = exec.Command("git", "log", "cros/main.."+mergeSHA, "--abbrev", "--oneline", "--no-merges", "--format=\"%C(auto) %h %s (%an)\"")
	out, _ := runCmd(cmd, false)
	commitMsg := "Merge cros/upstream to cros/main - " + lastTag + "\n\nPart of an uprev that contains the following commits:\n\n" + out + "\n\nBUG=" + bugFlag + "\nFIXED=" + bugFlag + "\n\nTEST=None"
	if cqDepend != "" {
		commitMsg = commitMsg + "\n\nCq-Depend: " + cqDepend
	}
	if getWD() == "modemmanager-next" {
		commitMsg += cqCommitText
	}
	if err := os.WriteFile("/tmp/commit-msg.log", []byte(commitMsg), 0644); err != nil {
		logPanic("Cannot write commits.log")
	}
	cmd = exec.Command("git", "commit", "--amend", "-F", "/tmp/commit-msg.log")
	runCmd(cmd, true)
}

func compile(compileBoardFlag string) {
	repoNames := []string{"libqrtr-glib", "libmbim", "libqmi", "modemmanager-next"}
	for _, repoName := range repoNames {
		cBoards := strings.Split(compileBoardFlag, ",")
		for _, c := range cBoards {
			runCmd(exec.Command("cros_sdk", "cros_workon", "--board="+c, "start", repoName), true)
			runCmd(exec.Command("cros_sdk", "emerge-"+c, repoName), true)
			defer runCmd(exec.Command("cros_sdk", "cros_workon", "--board="+c, "stop", repoName), false)
		}
	}
}

// returns gerrit cl
func postMerge(rootDir string, repoName string, upstreamBranch string, cqDepend string, uploadFlag bool, cqP1Flag bool, prettyMsg bool, bugFlag string) string {
	uprevDate := time.Now().Format("01-02-2006")
	branchName := "merge-upstream-" + uprevDate
	fmt.Println(rootDir, repoName, upstreamBranch, uprevDate)
	if err := os.Chdir(filepath.Join(rootDir, repoName)); err != nil {
		logPanic(err.Error())
	}
	newDir, _ := os.Getwd()
	fmt.Printf(colorGreen+"Working on : %s\n"+colorReset, newDir)

	var cmd *exec.Cmd
	cmd = exec.Command("git", "log", "cros/main..HEAD", "--abbrev", "--oneline", "--format=%h")
	out, _ := runCmd(cmd, true)
	if out == "" {
		return ""
	}

	cmd = exec.Command("git", "describe", "--abbrev=0", "cros/"+upstreamBranch)
	gitOut, _ := runCmd(cmd, true)
	// git may throw warnings, but still have the last tag as the last but one line.
	gitOutArr := strings.Split(gitOut, "\n")
	lastTag := gitOutArr[len(gitOutArr)-2]
	fmt.Printf(colorGreen+"Last tag : %s\n"+colorReset, lastTag)

	if prettyMsg {
		genCommitMsg("cros/main", "HEAD", cqDepend, bugFlag, lastTag)
	}

	if !uploadFlag {
		return ""
	}
	//TODO: check if -o uploadvalidator~skip is needed whenever AUTHORS change. (Florence_ is a banned word)
	cmd = exec.Command("repo", "upload", "--cbr", ".", "--no-verify", "-o", "topic="+branchName, "-y")
	cmd.Stdin = strings.NewReader("yes")
	out, _ = runCmd(cmd, true)
	re := regexp.MustCompile(`\+/(.*) Merge`)
	res := re.FindStringSubmatch(out)
	fmt.Printf("%sUploaded crrev.com/c/%s\n%s", colorGreen, res[1], colorReset)

	if cqP1Flag {
		runCmd(exec.Command("gerrit", "label-v", res[1], "1"), false)
		runCmd(exec.Command("gerrit", "label-cq", res[1], "1"), false)
	}

	return "chromium:" + res[1] + " "
}

func uploadEmptyCL() {
	fmt.Printf("%sUploading an empty CL before the merge...\n%s", colorGreen, colorReset)
	checkoutDir := lookupCheckout()
	rootDir := filepath.Join(checkoutDir, "src/third_party")
	if err := os.Chdir(filepath.Join(rootDir, emptyCLRepoName)); err != nil {
		logPanic(err.Error())
	}
	newDir, _ := os.Getwd()
	fmt.Printf("%sWorking on: %s\n%s", colorGreen, newDir, colorReset)

	runCmd(exec.Command("repo", "sync", "-d", "."), true)
	runCmd(exec.Command("git", "branch", "-D", emptyCLBranchName), false)
	runCmd(exec.Command("repo", "start", emptyCLBranchName, "."), true)
	defer runCmd(exec.Command("repo", "abandon", emptyCLBranchName, "."), false)

	f, err := os.OpenFile(emptyCLFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(emptyCLMsg); err != nil {
		panic(err)
	}

	runCmd(exec.Command("git", "add", "."), true)
	runCmd(exec.Command("git", "commit", "-m", emptyCLCommitMsg), true)

	cmd := exec.Command("repo", "upload", "--cbr", ".", "--no-verify", "-o", "topic="+emptyCLBranchName, "-y")
	cmd.Stdin = strings.NewReader("yes")
	out, _ := runCmd(cmd, true)
	re := regexp.MustCompile(`\+/(.*) EMPTY`)
	res := re.FindStringSubmatch(out)

	runCmd(exec.Command("gerrit", "label-v", res[1], "1"), false)
	runCmd(exec.Command("gerrit", "label-cq", res[1], "1"), false)
	fmt.Printf("%sUploaded an empty CL: crrev.com/c/%s\n%s", colorGreen, res[1], colorReset)
}

// Returns the absolute path of the chromeos source checkout.
func lookupCheckout() string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		logPanic("Failed to retrieve working directory: " + err.Error())
	}

	currentDirectory := workingDirectory
	for {
		repoDirectory := filepath.Join(currentDirectory, ".repo")
		if _, err := os.Stat(repoDirectory); err == nil {
			return currentDirectory
		}

		currentDirectory = filepath.Dir(currentDirectory)
		if currentDirectory == "/" {
			logPanic("Please run within a chromeos source checkout.")
		}
	}
}

func printUsageAndExit(exitcode int) {
	fmt.Printf("Usage: %s [ACTION] ...\n", filepath.Base(os.Args[0]))
	fmt.Printf("  ACTION is one of 'empty-cl', 'merge', 'post-merge' or 'help'\n")
	fmt.Printf("  Run 'ACTION -help' for details\n")
	fmt.Printf("\n")
	os.Exit(exitcode)
}

func main() {
	file, err := ioutil.TempFile("/tmp", "uprev")
	if err != nil {
		logPanic(err.Error())
	}
	log.SetOutput(file)
	defer fmt.Println("\nUprev logs in ", file.Name())

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "error: action not specified\n\n")
		printUsageAndExit(1)
	}

	if _, err := os.Stat("/etc/cros_chroot_version"); !errors.Is(err, os.ErrNotExist) {
		logPanic("Please run out of the chroot.")
	}

	switch os.Args[1] {
	case "empty-cl":
		uploadEmptyCL()

	case "merge":
		mergeSubCmd()

	case "compile":
		compileSubCmd()

	case "post-merge":
		postMergeSubCmd()

	case "help":
		printUsageAndExit(0)

	default:
		fmt.Fprintf(os.Stderr, "error: invalid action: %s\n\n", os.Args[1])
		printUsageAndExit(2)
	}
}

func mergeSubCmd() {
	mergeFlagSet := flag.NewFlagSet("merge", flag.ExitOnError)
	mergeUntilConflictFlag := mergeFlagSet.Bool("merge-until-conflict", false, "Merge the last non conflicting commit between current branch and cros/upstream")
	forceFlag := mergeFlagSet.Bool("force", false, "force a reset to cros/main. You will lose all changes on the current branch. Prefer stashing your changes instead")
	createAndMergeFlag := mergeFlagSet.Bool("create-branch-and-merge", false, "Creates a new branch and performs a git merge of cros/upstream* until a conflict occurs")
	squashMergeFlag := mergeFlagSet.String("squash-merge", "", "squashes all commits between provided SHA and HEAD. Use to overcome CQ uncertainity with stacked merge commits")
	libmbimCommitFlag := mergeFlagSet.String("libmbim-commit", "cros/upstream", "libmbim SHA that needs to be merged")
	libqmiCommitFlag := mergeFlagSet.String("libqmi-commit", "cros/upstream", "libqmi SHA that needs to be merged")
	libqrtrCommitFlag := mergeFlagSet.String("libqrtr-commit", "cros/upstream/main", "libqrtr-glib SHA that needs to be merged")
	mmCommitFlag := mergeFlagSet.String("mm-commit", "cros/upstream", "modemmanager SHA that needs to be merged")
	lastTagFlag := mergeFlagSet.Bool("last-tag", false, "merges until the last tag in upstream")

	mergeFlagSet.Parse(os.Args[2:])
	flagCount := 0
	if *mergeUntilConflictFlag {
		flagCount++
	}
	if *squashMergeFlag != "" {
		flagCount++
	}
	if *createAndMergeFlag {
		flagCount++
	}
	if flagCount != 1 {
		pre := "Only one"
		if flagCount == 0 {
			pre = "One"
		}
		logPanic(pre + " of create-branch-and-merge, squash-merge, merge-until-conflict should be provided")
	}
	if *lastTagFlag && !*createAndMergeFlag {
		logPanic("--last-tag needs to be used with create-and-merge")
	}

	if *mergeUntilConflictFlag {
		commitSHA, err := mergeUntilConflict("cros/upstream")
		if err != nil {
			logPanic(err.Error())
			return
		}
		fmt.Println(commitSHA, "is the last commit that can be merged without conflicts")
		return
	}

	if *squashMergeFlag != "" {
		squash(*squashMergeFlag)
		return
	}

	checkoutDir := lookupCheckout()
	rootDir := filepath.Join(checkoutDir, "src/third_party")

	if *createAndMergeFlag {
		uprevRepo(rootDir, "libqrtr-glib", "upstream/main", *lastTagFlag, *libqrtrCommitFlag, *forceFlag)
		uprevRepo(rootDir, "libqmi", "upstream", *lastTagFlag, *libqmiCommitFlag, *forceFlag)
		uprevRepo(rootDir, "libmbim", "upstream", *lastTagFlag, *libmbimCommitFlag, *forceFlag)
		uprevRepo(rootDir, "modemmanager-next", "upstream", *lastTagFlag, *mmCommitFlag, *forceFlag)
		return
	}
}

func compileSubCmd() {
	compileFlagSet := flag.NewFlagSet("compile", flag.ExitOnError)
	compileBoardFlag := compileFlagSet.String("board", "trogdor,dedede", "boards to be used in the compile process (default:trogdor,dedede)")
	skipSetupBoardFlag := compileFlagSet.Bool("skip-setup-board", false, "Skip executing setup_board. You will have to ensure that it's already been run before.")

	compileFlagSet.Parse(os.Args[2:])

	if !*skipSetupBoardFlag {
		cBoards := strings.Split(*compileBoardFlag, ",")
		for _, c := range cBoards {
			cmd := exec.Command("cros_sdk", "setup_board", "--board="+c)
			runCmd(cmd, true)
		}
	}

	compile(*compileBoardFlag)
}

func postMergeSubCmd() {
	postmergeFlagSet := flag.NewFlagSet("post-merge", flag.ExitOnError)
	prettyMsgFlag := postmergeFlagSet.Bool("pretty-msg", true, "prettify the commit message of the head commit.")
	uploadFlag := postmergeFlagSet.Bool("upload", false, "upload to gerrit")
	cqP1Flag := postmergeFlagSet.Bool("cq", false, "V+1, CQ+1 on gerrit")
	bugFlag := postmergeFlagSet.String("bug", "None", "Bug number in commit msg")
	postmergeFlagSet.Parse(os.Args[2:])
	if !*uploadFlag && !*prettyMsgFlag {
		logPanic("Atleast one of --upload, --cq, --pretty-msg needs to be set")
	}
	if *cqP1Flag && !*uploadFlag {
		logPanic("--cq needs to be used with --upload")
	}

	checkoutDir := lookupCheckout()
	rootDir := filepath.Join(checkoutDir, "src/third_party")
	cqDepend := ""

	libqrtrCL := postMerge(rootDir, "libqrtr-glib", "upstream/main", cqDepend, *uploadFlag, *cqP1Flag, *prettyMsgFlag, *bugFlag)
	libqmiCL := postMerge(rootDir, "libqmi", "upstream", cqDepend, *uploadFlag, *cqP1Flag, *prettyMsgFlag, *bugFlag)
	libmbimCL := postMerge(rootDir, "libmbim", "upstream", cqDepend, *uploadFlag, *cqP1Flag, *prettyMsgFlag, *bugFlag)

	cqDepend = libqrtrCL + libqmiCL + libmbimCL
	cqDepend = strings.ReplaceAll(strings.TrimSpace(cqDepend), " ", ",")
	postMerge(rootDir, "modemmanager-next", "upstream", cqDepend, *uploadFlag, *cqP1Flag, *prettyMsgFlag, *bugFlag)
}
