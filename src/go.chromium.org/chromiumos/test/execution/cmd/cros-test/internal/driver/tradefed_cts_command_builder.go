// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"log"
	"strings"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	ctsTestRunner        = "google/cts/google-cts-launcher-for-aosp"
	ctsVmConfigName      = "cts-virtual-device"
	ctsDeviceConfigName  = "cts-automated"
	ctsPackageName       = "android-cts.zip"
	ctsInvocationTimeout = "10h"
	ctsMaxRuntime        = "3h"
	ctsTestTag           = "al-cts-test"
	ctsRetryFailedRuns   = true
	ctsResultsUpload     = true
)

var ctsParams = []string{
	"--skip-device-info", "--no-use-device-build-info", "--property-check:no-throw-error",
	"--log-level", "VERBOSE", "--max-log-size", "62914560", "--max-tmp-logcat-file", "62914560",
	"--logcat-on-failure", "--screenshot-on-failure",
	"--result-reporter:disable-result-posting", "--use-log-saver",
}

var ctsSkipDownloadArtifacts = []string{"TESTS_ZIP", "DEVICE_IMAGE", "BASEBAND"}

func extractBuildInfo(test *api.TestCase) (branch string, target string, build string) {
	for _, t := range test.Tags {
		if strings.HasPrefix(t.Value, "branch:") {
			branch = strings.TrimPrefix(t.Value, "branch:")
		}
		// TODO: select the correct target based on device architecture.
		if strings.HasPrefix(t.Value, "target:") && strings.Contains(t.Value, "x86") {
			target = strings.TrimPrefix(t.Value, "target:")
		}
		if strings.HasPrefix(t.Value, "build_id:") {
			build = strings.TrimPrefix(t.Value, "build_id:")
		}
	}
	return
}

func BuildCtsTestCommand(logger *log.Logger, tests []*api.TestCaseMetadata, serials []string) []string {
	cmd := []string{ctsTestRunner}

	// Only physical devices are currently supported. For VMs, ctsVmConfigName
	// config should be used.
	cmd = append(cmd, "--config-name", ctsDeviceConfigName)

	cmd = append(cmd, "--cts-package-name", ctsPackageName, "--cts-use-partial-download",
		"--cts-version", "2", "--gdevice-flash:disable", "--invocation-timeout", ctsInvocationTimeout,
		"--log-level", "VERBOSE", "--max-run-time", ctsMaxRuntime, "--test-tag", ctsTestTag,
		"--invocation-data", "android-build-environment=prod", "--no-run-as-root",
		"--reporter-template", "template/reporters/subprocess-reporter", "--no-use-event-streaming",
		"--google-device-setup:set-global-setting", "verifier_verify_adb_installs=0")

	// Adding cts-params specific to the CTS test config.
	for _, param := range ctsParams {
		cmd = append(cmd, "--cts-params", param)
	}

	// Adding a list of artifacts to skip downloading by the CTS runner.
	for _, artifact := range ctsSkipDownloadArtifacts {
		cmd = append(cmd, "--skip-download", artifact)
	}

	if ctsRetryFailedRuns {
		cmd = append(cmd, "--max-testcase-run-count", "3", "--retry-isolation-grade", "FULLY_ISOLATED",
			"--retry-strategy", "RETRY_ANY_FAILURE", "--module-preparation-retry")
	} else {
		cmd = append(cmd, "--max-testcase-run-count", "1", "--retry-strategy", "NO_RETRY")
	}

	if ctsResultsUpload {
		cmd = append(cmd, "--cts-params", "--result-reporter:no-disable", "--ants-result-reporter:create-local-invocation")
	} else {
		cmd = append(cmd, "--cts-params", "--result-reporter:disable")
	}

	var buildInfoReported = false

	for _, t := range tests {
		testName := strings.TrimPrefix(t.TestCase.Name, "cts.")
		branch, target, build := extractBuildInfo(t.TestCase)
		logger.Println("Running provided test: ", testName,
			" for branch/target/build: ", branch, "/", target, "/", build)
		if !buildInfoReported {
			if len(branch) > 0 {
				cmd = append(cmd, "--cts-branch", branch, "--branch", branch)
			}
			if len(target) > 0 {
				cmd = append(cmd, "--cts-build-flavor", target, "--build-flavor", target)
			}
			if len(build) > 0 {
				cmd = append(cmd, "--cts-build-id", build, "--build-id", build)
			}
			buildInfoReported = true
		}

		cmd = append(cmd, "--cts-params", "--compatibility:include-filter", "--cts-params", testName)
	}

	for _, s := range serials {
		cmd = append(cmd, "-s", s)
	}

	logger.Println("Running TF command: ", cmd)
	return cmd
}
