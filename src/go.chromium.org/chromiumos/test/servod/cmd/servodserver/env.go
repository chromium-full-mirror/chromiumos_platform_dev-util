// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servodserver

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultCloudBotSSHDir = "/usr/local/etc/cloudbots/.ssh/"
)

var defaultSSHConfigPathOnCloudBot = filepath.Join(defaultCloudBotSSHDir, "config")

// getSwarmingBotID retrieves the swarming bot ID.
func getSwarmingBotID() string {
	return os.Getenv("SWARMING_BOT_ID")
}

// isCloudBot returns whether the process is running on cloud bot VM.
func isCloudBot() bool {
	if swarmingBotID := getSwarmingBotID(); strings.HasPrefix(swarmingBotID, "cloudbots-") {
		return true
	}
	return false
}
