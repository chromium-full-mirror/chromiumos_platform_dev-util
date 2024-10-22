// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package payload

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
)

// HTTP backed payload.
type HTTP struct {
	// URL of zstd-compressed images accessible from dut-agent.
	Kernel   string `json:"kernel"`
	Rootfs   string `json:"rootfs"`
	Stateful string `json:"stateful"`
}

var _ Payload = &HTTP{}

// InitClient implements Payload.
func (h *HTTP) InitClient(ctx context.Context) error {
	return nil
}

// CloseClient implements Payload.
func (h *HTTP) CloseClient() error {
	return nil
}

func (h *HTTP) imageURL(image artifact.ImageID) (string, error) {
	switch image {
	case artifact.Kernel:
		return h.Kernel, nil
	case artifact.Rootfs:
		return h.Rootfs, nil
	case artifact.Stateful:
		return h.Stateful, nil
	}
	return "", fmt.Errorf("invalid image ID %q", image)
}

// OpenImage implements Payload.
func (h *HTTP) OpenImage(ctx context.Context, image artifact.ImageID) (r io.ReadCloser, size int64, err error) {
	url, err := h.imageURL(image)
	if err != nil {
		return nil, 0, err
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("GET error on %q: %w", url, err)
	}
	if resp.ContentLength == -1 {
		return nil, 0, fmt.Errorf("Content-Length unavailable for %q", url)
	}
	return resp.Body, resp.ContentLength, err
}

// Compression implements Payload.
func (h *HTTP) Compression() artifact.Compression {
	return artifact.Zstd
}
