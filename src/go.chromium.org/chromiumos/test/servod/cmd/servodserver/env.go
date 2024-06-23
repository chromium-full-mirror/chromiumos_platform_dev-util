// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servodserver

import (
	"path/filepath"
)

const (
	defaultCloudBotSSHDir = "/usr/local/etc/cloudbots/.ssh/"
)

var defaultSSHConfigPathOnCloudBot = filepath.Join(defaultCloudBotSSHDir, "config")
