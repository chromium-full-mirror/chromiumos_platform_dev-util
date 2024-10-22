// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internal

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/downscope"
	"google.golang.org/api/option"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/gsmisc"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/payload"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/ssh"
)

// buildConditionExpression builds the CEL (https://github.com/google/cel-spec)
// expression for restricting Google Cloud Storage access to resources within
// the bucket and with the objectPrefix.
func buildConditionExpression(bucket, objectPrefix string) string {
	name := "projects/_/buckets/" + bucket + "/objects/" + objectPrefix
	if strings.ContainsAny(name, `'"\`) {
		panic("Bad name: " + name)
	}
	return fmt.Sprintf("resource.name.startsWith('%s')", name)
}

// createGSPayload creates a payload to flash GS artifacts.
func createGSPayload(ctx context.Context, token oauth2.TokenSource, sshClient *ssh.Client, sshTarget string, opts *Options) (*payload.OneOf, error) {
	storageClient, err := storage.NewClient(ctx,
		option.WithTokenSource(token),
	)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient failed: %w", err)
	}

	var board string
	if opts.GS != "" || opts.Board != "" {
		board = opts.Board
	} else {
		var err error
		board, err = DetectBoard(sshClient)
		if err != nil {
			return nil, fmt.Errorf("cannot detect board of %s: %w. %s", sshTarget, err,
				"specify --board or --gs to bypass auto board detection",
			)
		}
		log.Println("DUT board:", board)
	}

	art, err := getFlashTarget(ctx, storageClient, board, opts)
	if err != nil {
		return nil, err
	}
	log.Printf("flashing directory: gs://%s", path.Join(art.Bucket, art.Dir))
	log.Print("flashing images: ", strings.Join(art.Names(), " "))

	payload, err := downscopeGSPayload(ctx, token, art)
	if err != nil {
		return nil, fmt.Errorf("downscopeGSPayload(): %w", err)
	}

	return payload, nil
}

// downscopeGSPayload creates a payload.OneOf for the dut-agent to perform a flash.
//
// The token is downscoped to only allow access to the Google Cloud Storage directory.
func downscopeGSPayload(ctx context.Context, token oauth2.TokenSource, art *artifact.GSArtifact) (*payload.OneOf, error) {
	// Downscope with Credential Access Boundaries
	// https://cloud.google.com/iam/docs/downscoping-short-lived-credentials
	down, err := downscope.NewTokenSource(ctx, downscope.DownscopingConfig{
		RootSource: token,
		Rules: []downscope.AccessBoundaryRule{
			{
				AvailableResource: "//storage.googleapis.com/projects/_/buckets/" + art.Bucket,
				AvailablePermissions: []string{
					"inRole:roles/storage.objectViewer",
				},
				Condition: &downscope.AvailabilityCondition{
					Title:       "bound-to-directory",
					Description: "Limit access to the intended directory.",
					Expression:  buildConditionExpression(art.Bucket, art.Dir),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("downscope.NewTokenSource failed: %w", err)
	}

	tok, err := down.Token()
	if err != nil {
		return nil, fmt.Errorf("down.Token() failed: %w", err)
	}
	tok.RefreshToken = ""

	gsPayload := &payload.CloudStorage{
		Token:    tok,
		Artifact: art,
	}

	// Check that the downscoped token works.
	if err := checkGSPayload(ctx, gsPayload); err != nil {
		return nil, fmt.Errorf("checkGSPayload: %w", err)
	}

	return &payload.OneOf{CloudStorage: gsPayload}, nil
}

func checkGSPayload(ctx context.Context, p *payload.CloudStorage) error {
	c, err := p.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return gsmisc.CheckArtifact(ctx, c, p.Artifact)
}
