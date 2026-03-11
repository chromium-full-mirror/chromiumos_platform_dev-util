// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"go.chromium.org/chromiumos/test/util/portdiscovery"

	"golang.org/x/sys/unix"
)

const (
	gsBucketParam       = "gs_bucket"
	sourceURLKey        = "source_url"
	filePathKey         = "file"
	downloadPrefix      = "/download/"
	downloadLocalPrefix = "/download-local/"
	extractPrefix       = "/extract/"
	staticPrefix        = "/static/"
	isStagedPrefix      = "/is_staged/"
	stagePrefix         = "/stage/"
	checkHealthPrefix   = "/check_health/"
)

// Maximum number of allowed concurrent cacheGSHandler executions
var (
	maxConcurrency          = runtime.NumCPU() * 4
	gsHandlerConcurrencySem = make(chan struct{}, maxConcurrency) // Semaphore to control concurrency
)

// HTTPHandlers contains the cache server api endpoint logic
type HTTPHandlers struct {
	cache *Cache
}

// InstantiateHandlers creates the caching layer, sets up the HTTP handlers,
// and sets it so the cache gets destroyed on sigterm (i.e.: deletes files)
func InstantiateHandlers(ctx context.Context, port int, cacheLocation string, timeout time.Duration) error {
	cache, err := MakeCache(cacheLocation, timeout)
	if err != nil {
		return fmt.Errorf("could not create cache, %w", err)
	}
	defer cache.Close()

	// Clean up on SIGINT and SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, unix.SIGTERM)
	go func() {
		<-c
		cache.Close()
		os.Exit(1)
	}()

	// TODO(jaquesc): Add SSL (currently unnecessary for localhost)
	h := HTTPHandlers{
		cache: cache,
	}

	http.HandleFunc(downloadPrefix, h.cacheGSHandler)
	http.HandleFunc(downloadLocalPrefix, h.cacheLocalHandler)
	http.HandleFunc(extractPrefix, h.extractHandler)
	http.HandleFunc(staticPrefix, h.staticHandler)
	http.HandleFunc(isStagedPrefix, h.isStagedHandler)
	http.HandleFunc(stagePrefix, h.stageHandler)
	http.HandleFunc(checkHealthPrefix, h.checkHealthHandler)

	config := &net.ListenConfig{Control: reusePort}
	l, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	// Write port number to ~/.cftmeta for go/cft-port-discovery
	err = portdiscovery.WriteServiceMetadata("cache-server", l.Addr().String(), log.Default())
	if err != nil {
		log.Println("Warning: error when writing to metadata file: ", err)
	}

	if err := http.Serve(l, nil); err != nil {
		return err
	}

	return nil
}

func reusePort(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(descriptor uintptr) {
		syscall.SetsockoptInt(int(descriptor), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
}

// cacheGSHandler handles the cache for GS
func (h *HTTPHandlers) cacheGSHandler(w http.ResponseWriter, r *http.Request) {

	gsHandlerConcurrencySem <- struct{}{}
	defer func() { <-gsHandlerConcurrencySem }()
	switch r.Method {
	case http.MethodGet:
		h.getCacheGSHandler(r.Context(), w, strings.TrimPrefix(r.URL.EscapedPath(), downloadPrefix))
	case http.MethodHead:
		h.headCacheGSHandler(r.Context(), w, strings.TrimPrefix(r.URL.EscapedPath(), downloadPrefix))
	default:
		http.Error(w, "Only GETs and HEADs are supported.", http.StatusMethodNotAllowed)
	}
}

// checkHealthHandler is a stub endpoint that returns nothing
func (h *HTTPHandlers) checkHealthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("received %v request", checkHealthPrefix)
	return
}

// isStagedHandler is a stub endpoint that returns "True"
func (h *HTTPHandlers) isStagedHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("received %v request", isStagedPrefix)
	io.WriteString(w, "True")
	return
}

// stageHandler is a stub endpoint that returns nothing
func (h *HTTPHandlers) stageHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("received %v request", stagePrefix)
	return
}

// staticHandler handles GET requests to GS cache
func (h *HTTPHandlers) staticHandler(w http.ResponseWriter, r *http.Request) {
	bucketParam, ok := r.URL.Query()[gsBucketParam]
	if !ok || len(bucketParam) != 1 {
		http.Error(w, "URL must have a bucket query parameter", http.StatusUnprocessableEntity)
		return
	}
	gsPath := path.Join(bucketParam[0], strings.TrimPrefix(r.URL.Path, staticPrefix))
	switch r.Method {
	case http.MethodGet:
		h.getCacheGSHandler(r.Context(), w, gsPath)
	case http.MethodHead:
		h.headCacheGSHandler(r.Context(), w, gsPath)
	default:
		http.Error(w, "Only GETs and HEADs are supported.", http.StatusMethodNotAllowed)
	}
}

// rewriteAndroidURL rewrites a URL for an android artifact to a GCS path.
// Returns the new path and true if it was rewritten.
func rewriteAndroidURL(gsPath string) (string, bool) {
	// android-build/builds/14099659/brya-trunk_staging-userdebug/attempts/latest/artifacts/partition-common-cgpt.sh
	// ->
	// android-build/builds/*-linux-brya-trunk_staging-userdebug/14099659/*/1/partition-common-cgpt.sh
	r := regexp.MustCompile(`^android-build/builds/(\d+)/(.+)/attempts/latest/artifacts/(.+)`)
	matches := r.FindStringSubmatch(gsPath)
	if len(matches) == 4 {
		buildID := matches[1]
		target := matches[2]
		artifact := matches[3]
		newPath := fmt.Sprintf("android-build/builds/*-linux-%s/%s/*/1/%s", target, buildID, artifact)
		return newPath, true
	}
	return gsPath, false
}

// resolveGSPath fetches a GCS path, downloading it if necessary, and returns the local path.
func (h *HTTPHandlers) resolveGSPath(ctx context.Context, gsPath string) (string, error) {
	// Check if it's an android URL that should be fetched with fetch_artifact
	r := regexp.MustCompile(`^android-build/builds/(\d+)/(.+)/attempts/latest/artifacts/(.+)`)
	matches := r.FindStringSubmatch(gsPath)

	if len(matches) == 4 {
		buildID := matches[1]
		target := matches[2]
		artifact := matches[3]

		fetcher := func(destPath string) error {
			// Stubby is only available on corp.
			// If running off corp we need to use Cloud ADC.
			useApplicationDefaultCredentials := false
			if _, err := exec.LookPath("network-detect"); err == nil {
				if err := exec.Command("network-detect").Run(); err != nil {
					// ExitCode 1 means off corp.
					if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
						useApplicationDefaultCredentials = true
					}
				}
			}

			args := []string{"--bid", buildID, "--target", target, artifact, destPath}
			if useApplicationDefaultCredentials {
				args = append([]string{"--use_adc"}, args...)
			}

			cmd := exec.CommandContext(ctx, "fetch_artifact", args...)
			log.Printf("Fetching android artifact with: %s", cmd)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("fetch_artifact failed: %v\n%s", err, stderr.String())
			}
			return nil
		}
		return h.cache.GetWithFetcher(gsPath, fetcher)

	}
	// Fallback to original GCS logic for other URLs
	if newPath, ok := rewriteAndroidURL(gsPath); ok {
		log.Printf("rewriting android url from %s to %s", gsPath, newPath)
		gsPath = newPath
	}
	return h.cache.Get(gsPath)
}

// getCacheGSHandler handles GET requests to GS cache
func (h *HTTPHandlers) getCacheGSHandler(ctx context.Context, w http.ResponseWriter, gsPath string) {
	log.Printf("got GET request for GS file: %v", gsPath)
	if gsPath == "" {
		http.Error(w, "URL must have a path to download", http.StatusUnprocessableEntity)
		return
	}

	localPath, err := h.resolveGSPath(ctx, gsPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to get artifact %s, %v", gsPath, err), http.StatusBadRequest)
		return
	}

	fr, err := os.OpenFile(localPath, os.O_RDONLY, 0644)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to open local cache %s, %v", localPath, err), http.StatusInternalServerError)
		return
	}
	defer fr.Close()

	log.Printf("sending file, %s", localPath)
	// Internal impl should stream here
	io.Copy(w, fr)
	log.Printf("sent file, %s", localPath)
}

// headCacheGSHandler handles HEAD requests to GS cache
func (h *HTTPHandlers) headCacheGSHandler(ctx context.Context, w http.ResponseWriter, gsPath string) {
	log.Printf("got HEAD request for GS file: %v", gsPath)
	if gsPath == "" {
		http.Error(w, "URL must have a path to download", http.StatusUnprocessableEntity)
		return
	}

	localPath, err := h.resolveGSPath(ctx, gsPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to get artifact %s, %v", gsPath, err), http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(localPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to stat local cache %s, %v", localPath, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.WriteHeader(http.StatusOK)
}

// cacheLocalHandler handles the cache for local files
func (h *HTTPHandlers) cacheLocalHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getCacheLocalHandler(w, r)
	default:
		http.Error(w, "Only GETs are supported.", http.StatusNotFound)
	}
}

// getCacheLocalHandler handles GET requests to local files
func (h *HTTPHandlers) getCacheLocalHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("got GET request for local file")
	localPath := r.URL.Query().Get(sourceURLKey)
	if localPath == "" {
		http.Error(w, fmt.Sprintf("URL must have an option with %s field", sourceURLKey), http.StatusUnprocessableEntity)
	}
	fr, err := os.OpenFile(localPath, os.O_RDONLY, 0644)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to open local file %s, %v", localPath, err), http.StatusBadRequest)
	}
	defer fr.Close()

	log.Printf("sending file, %s", localPath)
	// Internal impl should stream here
	io.Copy(w, fr)
	log.Printf("sent file, %s", localPath)
}

// extractHandler downloads the given tar.xz file specified in the url path, and extracts the file provided in the url
func (h *HTTPHandlers) extractHandler(w http.ResponseWriter, r *http.Request) {
	fileToExtract := r.URL.Query().Get(filePathKey)
	if fileToExtract == "" {
		http.Error(w, fmt.Sprintf("URL must have a %s parameter", filePathKey), http.StatusBadRequest)
		return
	}
	gcsPath := strings.TrimPrefix(r.URL.Path, extractPrefix)
	localTarFilePath, err := h.cache.Get(gcsPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to cache %s, %v", r.URL.Path, err), http.StatusBadRequest)
		return
	}
	log.Printf("extracting %v from %v", fileToExtract, localTarFilePath)
	extractCmd := exec.Command("tar", "-Oxf", localTarFilePath, fileToExtract)
	var stderr strings.Builder
	extractCmd.Stdout = w
	extractCmd.Stderr = &stderr
	if err := extractCmd.Run(); err != nil {
		log.Printf("tar failed, err = %v stderr = %s", err, stderr.String())
		// Real cache servers return a 404 error when the requested file is not present in the archive.
		if strings.Contains(stderr.String(), "Not found in archive") {
			http.Error(w, stderr.String(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("error when extracting %v from %v: %v\n%s", fileToExtract, localTarFilePath, err, stderr.String()), http.StatusInternalServerError)
		}
	}
}
