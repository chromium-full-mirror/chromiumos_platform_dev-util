// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package payload

import (
	"context"
	"fmt"
	"io"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	gsmisc "chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/gsmisc"
	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// CloudStorage backed payload.
type CloudStorage struct {
	Token    *oauth2.Token
	Artifact *artifact.GSArtifact

	client *storage.Client
}

var _ Payload = &CloudStorage{}

// NewClient returns a new client with the token on c.
func (c *CloudStorage) NewClient(ctx context.Context) (*storage.Client, error) {
	client, err := storage.NewClient(ctx,
		option.WithTokenSource(oauth2.StaticTokenSource(c.Token)),
	)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient failed: %w", err)
	}
	return client, nil
}

// InitClient implements Payload.
func (c *CloudStorage) InitClient(ctx context.Context) error {
	client, err := c.NewClient(ctx)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

// CloseClient implements Payload.
func (c *CloudStorage) CloseClient() error {
	return c.client.Close()
}

// OpenImage implements Payload.
func (c *CloudStorage) OpenImage(ctx context.Context, image artifact.ImageID) (io.ReadCloser, int64, error) {
	obj := gsmisc.ObjectHandle(c.client, c.Artifact.Bucket, c.Artifact.Dir, c.Artifact.Images.Name(image))

	rd, err := obj.NewReader(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("obj.NewReader for %s failed: %w", gsmisc.URI(obj), err)
	}
	return rd, rd.Attrs.Size, nil
}

// Compression implements Payload.
func (c *CloudStorage) Compression() artifact.Compression {
	return c.Artifact.Compression
}
