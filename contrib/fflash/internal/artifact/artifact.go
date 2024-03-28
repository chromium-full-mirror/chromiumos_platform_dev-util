// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package artifact

type Artifact struct {
	// Cloud storage bucket path.
	Bucket string
	Dir    string

	Images
}

// Images describes the image names in the artifact directory.
type Images struct {
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

// GzipImages to flash.
//
// See https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:chromite/lib/constants.py;l=697-704;drc=e94de9b6a35c961b92a9aa57fc3f7923525b9925
var GzipImages = Images{
	Kernel:   "full_dev_part_KERN.bin.gz",
	Rootfs:   "full_dev_part_ROOT.bin.gz",
	Stateful: "stateful.tgz",
}
