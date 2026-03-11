// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const cacheSize = 128

type Cache struct {
	location    string
	pathToLocal *LRU
	gcsClient   *storage.Client
	timeout     time.Duration
}

func MakeCache(location string, timeout time.Duration) (*Cache, error) {
	log.Printf("creating cache on location %s", location)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate GCS client, %w", err)
	}

	lru, _ := MakeLRU(cacheSize, deleteFileCallback)

	return &Cache{
		location:    location,
		pathToLocal: lru,
		gcsClient:   client,
		timeout:     timeout,
	}, nil
}

func MakeCacheFromexisting(location string) *Cache {
	// TODO(jaquesc): option to reuse local cache on server restart
	return nil
}

// Get retrieves the local path to the gsPath. If none exist, download.
func (c *Cache) Get(gsPath string) (string, error) {
	fetcher := func(destPath string) error {
		return c.fetchFromGS(gsPath, destPath)
	}
	return c.GetWithFetcher(gsPath, fetcher)
}

// GetWithFetcher retrieves the local path to the key. If none exist, it uses the
// provided fetcher function to download the file.
func (c *Cache) GetWithFetcher(key string, fetcher func(destPath string) error) (string, error) {
	log.Printf("attempting to fetch %s from cache", key)
	if c.pathToLocal.Exists(key) {
		localPath, err := c.pathToLocal.Get(key)
		if err != nil {
			return "", err
		}
		log.Printf("%s already cached at location %s", key, localPath)
		return localPath, nil
	}
	localPath := path.Join(c.location, uuid.New().String())
	log.Printf("%s not cached, retrieving at %s", key, localPath)
	if err := fetcher(localPath); err != nil {
		log.Printf("error when fetching artifact: %v", err)
		// Clean up empty file if fetcher created it.
		os.Remove(localPath)
		return "", err
	}
	log.Printf("successfully downloaded %s", key)

	// storing in local cache
	c.pathToLocal.Add(key, localPath)
	return localPath, nil
}

// fetchFromGS downloads a file from gsPath onto the local URI localPath
func (c *Cache) fetchFromGS(gsPath, localPath string) error {
	bucket, object, err := c.parseGSURL(gsPath)
	if err != nil {
		return fmt.Errorf("failed to parse gs url, %w", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resolvedObject := object
	if strings.Contains(object, "*") {
		log.Printf("Resolving wildcard in object path: %s", object)
		resolved, err := c.resolveGSWildcard(ctx, bucket, object)
		if err != nil {
			return fmt.Errorf("failed to resolve wildcard path %s: %w", gsPath, err)
		}
		resolvedObject = resolved
	}

	log.Printf("Fetching object %s from bucket %s", resolvedObject, bucket)
	rc, err := c.gcsClient.Bucket(bucket).Object(resolvedObject).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("could not get a reader for GCS object %q in bucket %q, %w", resolvedObject, bucket, err)
	}
	defer rc.Close()

	wf, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("could not create local file %s, %w", localPath, err)
	}

	defer wf.Close()

	//Using half of the default buffer size for memory optimization
	buf := make([]byte, 16*1024)

	if _, err := io.CopyBuffer(wf, rc, buf); err != nil {
		return fmt.Errorf("could not download gcs file, %w", err)
	}

	return nil
}

// resolveGSWildcard finds a GCS object given a pattern with one or two wildcards.
// If multiple objects match, the most recently updated one is returned.
func (c *Cache) resolveGSWildcard(ctx context.Context, bucket, objectPattern string) (string, error) {
	log.Printf("Searching for object with glob %q in bucket %q", objectPattern, bucket)

	query := &storage.Query{MatchGlob: objectPattern}
	it := c.gcsClient.Bucket(bucket).Objects(ctx, query)

	var newestObject *storage.ObjectAttrs
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error iterating objects in bucket %q with glob %q: %w", bucket, objectPattern, err)
		}

		if newestObject == nil || attrs.Updated.After(newestObject.Updated) {
			newestObject = attrs
		}
	}

	if newestObject == nil {
		return "", fmt.Errorf("could not find object matching pattern %s in bucket %s", objectPattern, bucket)
	}

	log.Printf("Found object: %s (updated at %s)", newestObject.Name, newestObject.Updated)
	return newestObject.Name, nil
}

// parseGSURL retrieves the bucket and object from a GS URL.
// URL expectation is of the form: "bucket/object"
// This method does not exists in the GS client, so creating bespoke.
func (c *Cache) parseGSURL(gsUrl string) (string, string, error) {
	if strings.HasPrefix(gsUrl, "gs://") {
		return "", "", fmt.Errorf("gs url must not have \"gs://\" prefix")
	}
	r := strings.SplitN(gsUrl, "/", 2)
	if len(r) != 2 {
		return "", "", fmt.Errorf("gs url must contain both a bucket and object")
	}
	return r[0], r[1], nil
}

// Close cleans up the cache (deletes files).
func (c *Cache) Close() {
	log.Println("cleaning up cache.")
	defer c.gcsClient.Close()
	c.pathToLocal.Delete()
}

// deleteFileCallback acts as the callback for what the LRU needs to do on item
// deletion.
func deleteFileCallback(key, value string) {
	log.Printf("deleting file %s", value)
	if err := os.Remove(value); err != nil {
		log.Fatalf("Could not delete %s because %v", value, err)
	}
}
