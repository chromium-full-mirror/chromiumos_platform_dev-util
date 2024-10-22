// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package artifact

import "log"

type GSArtifact struct {
	// Cloud storage bucket path.
	Bucket string
	Dir    string

	Images
}

// Compression type.
type Compression int

// List of compression types.
const (
	Gzip Compression = iota
	Zstd
)

// Images describes the image names in the artifact directory.
type Images struct {
	Compression Compression

	Kernel   string
	Rootfs   string
	Stateful string
}

// Names of images as a slice.
func (i *Images) Names() []string {
	return []string{
		i.Kernel,
		i.Rootfs,
		i.Stateful,
	}
}

// Name of the image ID.
func (i *Images) Name(id ImageID) string {
	switch id {
	case Kernel:
		return i.Kernel
	case Rootfs:
		return i.Rootfs
	case Stateful:
		return i.Stateful
	default:
		log.Panicf("unknown ImageID %q", id)
	}
	panic("unreachable")
}

// GzipImages to flash.
//
// See https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:chromite/lib/constants.py;l=697-704;drc=e94de9b6a35c961b92a9aa57fc3f7923525b9925
var GzipImages = Images{
	Compression: Gzip,
	Kernel:      "full_dev_part_KERN.bin.gz",
	Rootfs:      "full_dev_part_ROOT.bin.gz",
	Stateful:    "stateful.tgz",
}

// ZstdImages to flash.
//
// See https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:chromite/lib/constants.py;l=706-711;drc=e94de9b6a35c961b92a9aa57fc3f7923525b9925
var ZstdImages = Images{
	Compression: Zstd,
	Kernel:      "full_KERN.bin.zst",
	Rootfs:      "full_ROOT.bin.zst",
	Stateful:    "stateful.zst",
}

// ImageID is used to
type ImageID string

const (
	Kernel   ImageID = "kernel"
	Rootfs   ImageID = "rootfs"
	Stateful ImageID = "stateful"
)
