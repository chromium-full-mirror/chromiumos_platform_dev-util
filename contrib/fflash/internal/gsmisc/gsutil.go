// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gsmisc

import (
	"context"
	"fmt"
	"path"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"cloud.google.com/go/storage"
)

// URI returns the gs:// URI for the object handle.
func URI(obj *storage.ObjectHandle) string {
	return fmt.Sprintf("gs://%s/%s", obj.BucketName(), obj.ObjectName())
}

// ObjectHandle returns the storage.ObjectHandle for the artifact bucket, dir and name.
func ObjectHandle(client *storage.Client, bucket, dir, name string) *storage.ObjectHandle {
	return client.Bucket(bucket).Object(path.Join(dir, name))
}

// CheckArtifact checks that all image artifacts are accessible.
func CheckArtifact(ctx context.Context, client *storage.Client, artifact *artifact.GSArtifact) error {
	for _, file := range artifact.Images.Names() {
		obj := ObjectHandle(client, artifact.Bucket, artifact.Dir, file)

		if _, err := obj.Attrs(ctx); err != nil {
			return fmt.Errorf("%s: %w", URI(obj), err)
		}
	}
	return nil
}
