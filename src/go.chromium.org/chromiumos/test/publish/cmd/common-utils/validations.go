// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Validations functions
package common_utils

import (
	"fmt"
	"regexp"

	_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
)

var sourcesPathRE = regexp.MustCompile(`gs://[a-z0-9-_.]+/.+/metadata/sources.jsonpb`)

// ValidateTKOPublishRequest validates tko publish request
func ValidateTKOPublishRequest(req *api.PublishRequest, metadata *api.PublishTkoMetadata) error {
	if err := ValidateGenericPublishRequest(req); err != nil {
		return fmt.Errorf("error in publish request: %s", err)
	} else if err := ValidateTKORequestMetadata(metadata); err != nil {
		return fmt.Errorf("error in tko publish request metadata: %s", err)
	}
	return nil
}

// ValidateCpconPublishRequest validates cpcon publish request
func ValidateCpconPublishRequest(req *api.PublishRequest) error {
	if err := ValidateGenericPublishRequest(req); err != nil {
		return fmt.Errorf("error in publish request: %s", err)
	}
	return nil
}

// ValidateGenericPublishRequest validates generic publish request
func ValidateGenericPublishRequest(req *api.PublishRequest) error {
	if req.GetArtifactDirPath().GetPath() == "" {
		return fmt.Errorf("local artifact dir path is empty")
	} else if req.GetArtifactDirPath().GetHostType() != _go.StoragePath_LOCAL {
		return fmt.Errorf("artifact dir path must be of type local")
	}
	return nil
}

// ValidateTKORequestMetadata validates tko request metadata
func ValidateTKORequestMetadata(metadata *api.PublishTkoMetadata) error {
	if metadata.GetJobName() == "" {
		return fmt.Errorf("JobName is required in metadata for tko publish")
	}

	return nil
}
