// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"chromiumos/test/util/portdiscovery"
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
	"runtime"
	"strings"
	"syscall"
	"time"

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
		h.getCacheGSHandler(w, strings.TrimPrefix(r.URL.EscapedPath(), downloadPrefix))
	default:
		http.Error(w, "Only GETs are supported.", http.StatusNotFound)
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
	switch r.Method {
	case http.MethodGet:
		bucketParam, ok := r.URL.Query()[gsBucketParam]
		if !ok || len(bucketParam) != 1 {
			http.Error(w, "URL must have a bucket query parameter", http.StatusUnprocessableEntity)
			return
		}
		gsPath := path.Join(bucketParam[0], strings.TrimPrefix(r.URL.Path, staticPrefix))
		h.getCacheGSHandler(w, gsPath)
	default:
		http.Error(w, "Only GETs are supported.", http.StatusNotFound)
	}
}

// getCacheGSHandler handles GET requests to GS cache
func (h *HTTPHandlers) getCacheGSHandler(w http.ResponseWriter, gsPath string) {
	log.Printf("got GET request for GS file: %v", gsPath)
	if gsPath == "" {
		http.Error(w, "URL must have a path to download", http.StatusUnprocessableEntity)
		return
	}
	localPath, err := h.cache.Get(gsPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to cache %s, %v", gsPath, err), http.StatusBadRequest)
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

// cacheLocalHandler handles the cache for local files
func (h *HTTPHandlers) cacheLocalHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getCacheLocalHandler(w, r)
	default:
		http.Error(w, "Only GETs are supported.", http.StatusNotFound)
	}
}

// getCacgeLocalHandler handles GET requests to local files
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
	extractCmd.Stdout = w
	if err := extractCmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("error when extracting %v from %v", fileToExtract, localTarFilePath), http.StatusInternalServerError)
	}
}
