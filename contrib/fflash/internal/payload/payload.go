// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package payload

import (
	"context"
	"fmt"
	"io"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
)

// OneOf the Payloads for dut-agent to flash.
// One of the payload types must be set.
type OneOf struct {
	CloudStorage *CloudStorage `json:"cloud_storage"`
	HTTP         *HTTP         `json:"http"`
}

// Instance returns the Payload instance.
func (p *OneOf) Instance() (Payload, error) {
	if p.CloudStorage != nil {
		return p.CloudStorage, nil
	}
	if p.HTTP != nil {
		return p.HTTP, nil
	}
	return nil, fmt.Errorf("none of the payloads are set")
}

// CloseFunc closes a reader.
type CloseFunc func() error

// Payload is an abstraction over a set of Artifacts to flash to the device.
type Payload interface {
	// InitClient initializes the client associated with this Provider.
	InitClient(ctx context.Context) error
	// CloseClient closes the client associated with this Provider.
	CloseClient() error

	// OpenImage opens the image and returns the reader and the size of the compressed image.
	OpenImage(ctx context.Context, image artifact.ImageID) (r io.ReadCloser, size int64, err error)

	// Compression returns the compression kind of the images.
	Compression() artifact.Compression
}
