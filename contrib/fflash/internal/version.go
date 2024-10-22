// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/exp/slices"
	"google.golang.org/api/iterator"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/gsmisc"
)

// version like R{r}-{x}.{y}.{z} e.g. R109-15236.80.0
//
// In fflash, we refer to those as:
//
//	R109-15236.80.0
//	 --- ----- -- -
//	 |   |     |  |
//	 |   |     |  Patch number
//	 |   |     Branch number
//	 |   Build number
//	 Milestone number
//
// The full string is called the version.
type version struct {
	r int // Milestone number
	x int // Build number
	y int // Branch number
	z int // Patch number
}

var versionRegexp = regexp.MustCompile(`^R(\d+)-(\d+).(\d+)\.(\d+)$`)

func parseVersion(v string) (version, error) {
	m := versionRegexp.FindStringSubmatch(v)
	if m == nil {
		return version{}, fmt.Errorf("cannot parse %q as version", v)
	}
	var nums [4]int
	for i := range nums {
		var err error
		nums[i], err = strconv.Atoi(m[i+1])
		if err != nil {
			return version{}, fmt.Errorf("cannot parse %q as version: %v", v, err)
		}
	}
	return version{nums[0], nums[1], nums[2], nums[3]}, nil
}

func (v version) branched() bool {
	return v.y != 0
}

func (v version) less(w version, latestMilestone int) bool {
	if v.r != w.r {
		return v.r < w.r
	}

	// For the same release R###, prefer branched versions.
	// For example R109-15236.80.0 should be preferred to R109-15237.0.0.
	// See also b/259389997.
	// The exception to this rule is if the milestone is the latest
	// milestone which is on ToT and has not been branched to a release branch.
	// In that case we should just pick the one with highest build number.
	// See also b/271417619
	if latestMilestone != v.r && v.branched() != w.branched() {
		return w.branched()
	}

	if v.x != w.x {
		return v.x < w.x
	}
	if v.y != w.y {
		return v.y < w.y
	}
	return v.z < w.z
}

func (v version) String() string {
	return fmt.Sprintf("R%d-%d.%d.%d", v.r, v.x, v.y, v.z)
}

func queryPrefix(ctx context.Context, c *storage.Client, q *storage.Query) ([]*storage.ObjectAttrs, error) {
	var result []*storage.ObjectAttrs

	if err := q.SetAttrSelection([]string{"Name"}); err != nil {
		panic("SetAttrSelection failed")
	}
	objects := c.Bucket("chromeos-image-archive").Objects(ctx, q)
	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error while executing query: %w", err)
		}
		result = append(result, attrs)
	}
	return result, nil
}

// GetLatestVersionWithPrefix finds the latest build with the given prefix on gs://chromeos-image-archive.
func GetLatestVersionWithPrefix(ctx context.Context, c *storage.Client, board, prefix string) (string, error) {
	fullPrefix := fmt.Sprintf("%s-release/%s", board, prefix)

	q := &storage.Query{
		Delimiter: "/",
		Prefix:    fullPrefix,
	}
	objects, err := queryPrefix(ctx, c, q)
	if err != nil {
		return "", fmt.Errorf("cannot get versions available for gs://chromeos-image-archive/%s*", fullPrefix)
	}

	var versions []version
	for _, attrs := range objects {
		v, err := parseVersion(path.Base(attrs.Prefix))
		if err != nil {
			log.Printf("Cannot parse %q, ignoring: %v", attrs.Prefix, err)
			continue
		}
		versions = append(versions, v)
	}

	latestMilestone, err := latestMilestone(ctx, c, board)
	if err != nil {
		return "", err
	}

	slices.SortFunc(versions, func(a, b version) bool { return b.less(a, latestMilestone) })
	for _, v := range versions {
		dir := fmt.Sprintf("%s-release/%s", board, v)
		if _, err := resolveArtifactImages(ctx, c, "chromeos-image-archive", dir); err != nil {
			log.Printf("ignoring version %q: %v", v, err)
			continue
		}
		return v.String(), nil
	}
	return "", fmt.Errorf("no versions found for gs://chromeos-image-archive/%s*", fullPrefix)
}

// getLatestVersionForLATEST finds the latest version for LATEST file for board on gs://chromeos-image-archive.
// If isPrefix is true, looks for the LATEST files having the {latest} as their prefix.
func getLatestVersionForLATEST(ctx context.Context, c *storage.Client, board string, latest string, isPrefix bool) (string, error) {
	name := fmt.Sprintf("%s-release/%s", board, latest)
	var objects []*storage.ObjectAttrs
	if isPrefix {
		var err error
		q := &storage.Query{Prefix: name}
		objects, err = queryPrefix(ctx, c, q)
		if err != nil {
			return "", fmt.Errorf("cannot get LATEST file from gs://chromeos-image-archive/%s*", name)
		}
	} else {
		objects = []*storage.ObjectAttrs{
			{
				Name: name,
			},
		}
	}

	var latestVersion version
	for _, attrs := range objects {
		latestFileObj := c.Bucket("chromeos-image-archive").Object(attrs.Name)
		r, err := latestFileObj.NewReader(ctx)
		if err != nil {
			return "", fmt.Errorf("cannot open LATEST file %s: %w", gsmisc.URI(latestFileObj), err)
		}
		rawVersion, err := io.ReadAll(r)
		if err != nil {
			return "", fmt.Errorf("cannot read from LATEST file %s: %w", gsmisc.URI(latestFileObj), err)
		}
		version, err := parseVersion(string(rawVersion))
		if err != nil {
			return "", fmt.Errorf("cannot parse LATEST file %s: %w", gsmisc.URI(latestFileObj), err)
		}
		// using the latest file to parse version so don't need to worry about milestone when comparing
		// versions
		if latestVersion.less(version, 0) {
			latestVersion = version
		}
	}

	if latestVersion == (version{}) {
		return "", fmt.Errorf("no LATEST file found for gs://chromeos-image-archive/%s*", name)
	}
	return latestVersion.String(), nil
}

func latestMilestone(ctx context.Context, c *storage.Client, board string) (int, error) {
	latestVersionStr, err := GetLatestVersion(ctx, c, board)
	if err != nil {
		return 0, fmt.Errorf("Cannot determine latest milestone %w", err)
	}
	latestVersion, err := parseVersion(latestVersionStr)
	if err != nil {
		return 0, fmt.Errorf("Cannot determine latest milestone %w", err)
	}
	return latestVersion.r, nil
}

// GetLatestVersionForMilestone finds the latest version for board and milestone on gs://chromeos-image-archive.
func GetLatestVersionForMilestone(ctx context.Context, c *storage.Client, board string, milestone int) (string, error) {
	version, err := getLatestVersionForLATEST(ctx, c, board, fmt.Sprintf("LATEST-release-R%d-", milestone), true)
	if err != nil {
		log.Printf("%v, maybe the milestone is not branched yet. Retrying with prefix matching", err)
		version, err = GetLatestVersionWithPrefix(ctx, c, board, fmt.Sprintf("R%d-", milestone))
	}
	return version, err
}

// GetLatestVersion finds the latest version for board on gs://chromeos-image-archive.
func GetLatestVersion(ctx context.Context, c *storage.Client, board string) (string, error) {
	return getLatestVersionForLATEST(ctx, c, board, "LATEST-main", false)
}

type snapshotID struct {
	milestone   int
	build       int
	snapshot    int
	buildbucket *big.Int
}

var snapshotIDRegexp = regexp.MustCompile(`^R(\d+)-(\d+).0.0-(\d+)-(\d+)$`)

func parseSnapshotID(s string) (snapshotID, error) {
	m := snapshotIDRegexp.FindStringSubmatch(s)
	if m == nil {
		return snapshotID{}, errors.New("bad format")
	}
	var id snapshotID
	var err error
	if id.milestone, err = strconv.Atoi(m[1]); err != nil {
		return snapshotID{}, fmt.Errorf("bad milestone %q", m[1])
	}
	if id.build, err = strconv.Atoi(m[2]); err != nil {
		return snapshotID{}, fmt.Errorf("bad build %q", m[2])
	}
	if id.snapshot, err = strconv.Atoi(m[3]); err != nil {
		return snapshotID{}, fmt.Errorf("bad snapshot %q", m[3])
	}
	if val, ok := new(big.Int).SetString(m[4], 10); !ok {
		return snapshotID{}, fmt.Errorf("bad buildbucket %q", m[4])
	} else {
		id.buildbucket = val
	}
	return id, nil
}

func (sid snapshotID) less(b snapshotID) bool {
	if sid.milestone != b.milestone {
		return sid.milestone < b.milestone
	}
	if sid.build != b.build {
		return sid.build < b.build
	}
	if sid.snapshot != b.snapshot {
		return sid.snapshot < b.snapshot
	}
	return sid.buildbucket.Cmp(b.buildbucket) == -1
}

func (sid snapshotID) String() string {
	return fmt.Sprintf("R%d-%d.0.0-%d-%s", sid.milestone, sid.build, sid.snapshot, sid.buildbucket)
}

// Given an artifact directory, figure out the image names.
func resolveArtifactImages(ctx context.Context, c *storage.Client, bucket, dir string) (*artifact.GSArtifact, error) {
	art := &artifact.GSArtifact{
		Bucket: bucket,
		Dir:    dir,
		Images: artifact.ZstdImages,
	}
	err1 := gsmisc.CheckArtifact(ctx, c, art)
	if err1 == nil {
		return art, nil
	}

	art = &artifact.GSArtifact{
		Bucket: bucket,
		Dir:    dir,
		Images: artifact.GzipImages,
	}
	err2 := gsmisc.CheckArtifact(ctx, c, art)
	if err2 == nil {
		return art, nil
	}

	return nil, errors.Join(err1, err2)
}

// Given a version hint, try to find a snapshot build.
// For snapshot/postsubmit images, see also:
// https://docs.google.com/presentation/d/1IynhrQMDtH3dAjXJD3Aheb4ZTYveVwBAEPgf6TgKzUs/edit?resourcekey=0-epY9CwLiU9zLv4E3dDs0FA#slide=id.g150f95639fd_0_70
func findSnapshotAroundVersion(ctx context.Context, c *storage.Client, snapshotOrPostsubmit, board, versionOrPrefix string) (*artifact.GSArtifact, error) {
	var tryVersions []string
	if v, err := parseVersion(versionOrPrefix); err != nil {
		log.Printf("Looking for %s builds with prefix %s", snapshotOrPostsubmit, versionOrPrefix)
		tryVersions = []string{versionOrPrefix}
	} else {
		log.Printf("Looking for %s builds around %s", snapshotOrPostsubmit, v)
		tryVersions = []string{
			version{v.r, v.x, 0, 0}.String(),
			// Also try one release build before.
			version{v.r, v.x - 1, 0, 0}.String(),
		}
	}

	buildDirectory := fmt.Sprintf("%s-%s", board, snapshotOrPostsubmit)

	for _, v := range tryVersions {
		q := &storage.Query{
			Delimiter: "/",
			Prefix:    fmt.Sprintf("%s/%s", buildDirectory, v),
		}
		objects, err := queryPrefix(ctx, c, q)
		if err != nil {
			return nil, fmt.Errorf("cannot get snapshots available for %s: %w", v, err)
		}
		var snapshots []snapshotID
		for _, obj := range objects {
			basename := path.Base(obj.Prefix)
			pid, err := parseSnapshotID(basename)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as snapshot ID: %w", basename, err)
			}
			snapshots = append(snapshots, pid)
		}
		slices.SortFunc(snapshots, func(a, b snapshotID) bool { return b.less(a) })
		for _, pid := range snapshots {
			// Verify that the flash payload actually exist.
			dir := fmt.Sprintf("%s/%s", buildDirectory, pid)
			art, err := resolveArtifactImages(ctx, c, "chromeos-image-archive", dir)
			if err != nil {
				log.Printf("%s not valid: %s", pid, err)
				continue
			}
			return art, nil
		}
	}
	return nil, fmt.Errorf("no valid %s build found", snapshotOrPostsubmit)
}

func getFlashTarget(ctx context.Context, c *storage.Client, board string, opts *Options) (*artifact.GSArtifact, error) {
	if opts.GS != "" {
		url, err := url.Parse(opts.GS)
		if err != nil {
			return nil, err
		}
		return resolveArtifactImages(ctx, c, url.Host, strings.TrimPrefix(url.RequestURI(), "/"))
	}

	var gsVersion string
	var err error
	if opts.VersionString != "" {
		if opts.Postsubmit || opts.Snapshot {
			// Don't resolve the version and let findSnapshotAroundVersion() handle it.
			gsVersion = opts.VersionString
		} else {
			// Resolve for release build
			gsVersion, err = GetLatestVersionWithPrefix(ctx, c, board, opts.VersionString)
		}
	} else if opts.MilestoneNum != 0 {
		gsVersion, err = GetLatestVersionForMilestone(ctx, c, board, opts.MilestoneNum)
	} else {
		gsVersion, err = GetLatestVersion(ctx, c, board)
	}
	if err != nil {
		return nil, err
	}

	if opts.Snapshot {
		return findSnapshotAroundVersion(ctx, c, "snapshot", board, gsVersion)
	} else if opts.Postsubmit {
		return findSnapshotAroundVersion(ctx, c, "postsubmit", board, gsVersion)
	}

	return resolveArtifactImages(ctx, c, "chromeos-image-archive", fmt.Sprintf("%s-release/%s", board, gsVersion))
}
