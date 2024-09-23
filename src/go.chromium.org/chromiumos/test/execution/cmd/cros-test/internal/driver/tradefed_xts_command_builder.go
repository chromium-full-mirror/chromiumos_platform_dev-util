// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"fmt"
	"log"
	"os"
	"strings"

	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	tfTestRunner         = "google/cts/google-cts-launcher-for-aosp"
	invocationTimeout    = "10h"
	maxRuntime           = "3h"
	retryFailedRuns      = true
	resultsUpload        = true
	defaultGlobalLogPath = "/tmp"

	ctsVmConfigName     = "cts-virtual-device"
	ctsDeviceConfigName = "cts-automated"
	ctsPackageName      = "android-cts.zip"
	ctsTestTag          = "cros-cts-test"

	dtsDeviceConfigName = "dts"
	dtsPackageName      = "android-dts.zip"
	dtsTestTag          = "cros-dts-test"
)

var ctsParams = []string{
	"--skip-device-info", "--no-use-device-build-info", "--property-check:no-throw-error",
	"--log-level", "VERBOSE", "--max-log-size", "62914560", "--max-tmp-logcat-file", "62914560",
	"--logcat-on-failure", "--screenshot-on-failure", "--include-test-log-tags",
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

func BuildXtsTestCommand(logger *log.Logger, testType string, tests []*api.TestCaseMetadata,
	serials []string, metadata *api.ExecutionMetadata) []string {

	cmd := []string{tfTestRunner}

	// Extract AnTS invocation id and WorkUnit ID from execution metadata.
	var AnTsInvID string
	var AnTsWorkUnitID string
	if metadata != nil && len(metadata.Args) > 0 {
		for _, arg := range metadata.Args {
			if arg.Flag == "ants_invocation_id" {
				AnTsInvID = arg.Value
			} else if arg.Flag == "ants_work_unit_id" {
				AnTsWorkUnitID = arg.Value
			}
		}
		logger.Printf("Got invocation id: %s, work unit id: %s.", AnTsInvID, AnTsWorkUnitID)
	}

	// Only physical devices are currently supported. For VMs, ctsVmConfigName
	// config should be used.
	if testType == "cts" {
		cmd = append(cmd, "--config-name", ctsDeviceConfigName,
			"--cts-package-name", ctsPackageName, "--test-tag", ctsTestTag)
	} else if testType == "dts" {
		cmd = append(cmd, "--config-name", dtsDeviceConfigName,
			"--cts-package-name", dtsPackageName, "--test-tag", dtsTestTag,
			"--rootdir-var", "DTS_ROOT")
	}

	logPath := os.Getenv("GLOBAL_LOG_PATH")
	if len(logPath) <= 0 {
		logPath = defaultGlobalLogPath
		os.Setenv("GLOBAL_LOG_PATH", logPath)
	}

	cmd = append(cmd, "--cts-use-partial-download",
		"--cts-version", "2", "--gdevice-flash:disable", "--invocation-timeout", invocationTimeout,
		"--log-level", "VERBOSE", "--max-run-time", maxRuntime, "--no-bugreport-on-invocation-ended",
		"--invocation-data", "android-build-environment=prod", "--no-run-as-root",
		"--reporter-template", "template/reporters/subprocess-reporter",
		"--android-build-api-log-saver:no-remove-staged-files", "--no-use-event-streaming",
		"--google-device-setup:set-global-setting", "verifier_verify_adb_installs=0",
		"--template:map", "reporters=template/reporters/xml-reporter.xml",
		"--metricsreporter:metrics-folder", logPath)

	// Adding cts-params specific to the CTS test config.
	for _, param := range ctsParams {
		cmd = append(cmd, "--cts-params", param)
	}

	// Adding a list of artifacts to skip downloading by the CTS runner.
	for _, artifact := range ctsSkipDownloadArtifacts {
		cmd = append(cmd, "--skip-download", artifact)
	}

	if retryFailedRuns {
		cmd = append(cmd, "--max-testcase-run-count", "3", "--retry-isolation-grade", "FULLY_ISOLATED",
			"--retry-strategy", "RETRY_ANY_FAILURE", "--module-preparation-retry")
	} else {
		cmd = append(cmd, "--max-testcase-run-count", "1", "--retry-strategy", "NO_RETRY")
	}

	if resultsUpload {
		cmd = append(cmd, "--cts-params", "--result-reporter:no-disable",
			"--ants-result-reporter:create-local-invocation",
			"--ants-result-reporter:save-metric-file")

		// Add invocation id and workunit id.
		if len(AnTsInvID) > 0 {
			cmd = append(cmd, "--invocation-data", fmt.Sprintf("invocation_id=%s", AnTsInvID))
		}
		if len(AnTsWorkUnitID) > 0 {
			cmd = append(cmd, "--invocation-data", fmt.Sprintf("work_unit_id=%s", AnTsWorkUnitID))
		}
	} else {
		cmd = append(cmd, "--cts-params", "--result-reporter:disable")
	}

	var buildInfoReported = false

	for _, t := range tests {
		testName := strings.TrimPrefix(t.TestCase.Name, "cts.")
		testName = strings.TrimPrefix(testName, "dts.")
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

	// Add devices
	for _, s := range serials {
		cmd = append(cmd, "-s", s)
	}

	return cmd
}
