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
	planMetadataFlag     = "plan"
	tfGoogleTestRunner   = "google/cts/google-cts-launcher-for-aosp"
	tfAospTestRunner     = "common-compatibility-config"
	invocationTimeout    = "10h"
	maxRuntime           = "3h"
	retryFailedRuns      = true
	resultsUpload        = true
	defaultGlobalLogPath = "/tmp"
)

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

func getTestRunner() string {
	if isAospTradefed() {
		return tfAospTestRunner
	} else {
		return tfGoogleTestRunner
	}
}

func formatTestName(fqTestName string) string {
	testName := fqTestName

	// First, trim test suite prefix for all known suites.
	for _, suite := range knownSuites {
		testName = strings.TrimPrefix(testName, suite+".")
	}

	// Then, check if we need to split the test class from the module name,
	// i.e. "CtsModule#com.android.test.Class#testCase" is converted to
	// "CtsModule com.android.test.Class#testCase"
	if strings.Contains(testName, "#") {
		return strings.Replace(testName, "#", " ", 1)
	} else {
		return testName
	}
}

func extractMetadataFlag(metadata *api.ExecutionMetadata, flagName string) (string, error) {
	if metadata != nil && len(metadata.Args) > 0 {
		for _, arg := range metadata.Args {
			if arg.Flag == flagName {
				return arg.Value, nil
			}
		}
	}
	return "", fmt.Errorf("flag %s for found in test metadata", flagName)
}

func BuildXtsTestCommand(logger *log.Logger, testType string, tests []*api.TestCaseMetadata,
	serials []string, metadata *api.ExecutionMetadata) []string {

	cmd := []string{getTestRunner()}

	plan, err := extractMetadataFlag(metadata, planMetadataFlag)
	if err != nil {
		// By default, plan is the same as test suite type, i.e. "cts", "dts", etc.
		plan = testType
	}

	// Add test type specific arguments.
	if isAospTradefed() {
		cmd = append(cmd, "--plan", plan, "--log-level-display", "VERBOSE",
			"--test-tag", fmt.Sprintf("cros-%s-test", testType),
			"--use-device-build-info", "--primary-abi-only", "--stage-remote-file",
			"--load-configs-with-include-filters", "--no-skip-staging-artifacts")
	} else {
		cmd = append(cmd, "--config-name", plan,
			"--test-tag", fmt.Sprintf("cros-%s-test", testType),
			"--cts-package-name", fmt.Sprintf("android-%s.zip", testType),
			"--rootdir-var", fmt.Sprintf("%s_ROOT", strings.ToUpper(testType)))
	}

	logPath := os.Getenv("GLOBAL_LOG_PATH")
	if len(logPath) <= 0 {
		logPath = defaultGlobalLogPath
		os.Setenv("GLOBAL_LOG_PATH", logPath)
	}

	cmd = append(cmd, "--invocation-timeout", invocationTimeout, "--log-level", "VERBOSE")

	if !isAospTradefed() {

		cmd = append(cmd, "--cts-use-partial-download", "--cts-version", "2", "--gdevice-flash:disable",
			"--no-bugreport-on-invocation-ended", "--invocation-data", "android-build-environment=prod",
			"--android-build-api-log-saver:no-remove-staged-files", "--no-use-event-streaming",
			"--google-device-setup:set-global-setting", "verifier_verify_adb_installs=0",
			"--reporter-template", "template/reporters/subprocess-reporter",
			"--template:map", "reporters=template/reporters/xml-reporter.xml",
			"--metricsreporter:metrics-folder", logPath, "--max-run-time", maxRuntime)

		// Adding cts-params with prefix for each one.
		ctsParams := []string{
			"--skip-device-info", "--no-use-device-build-info", "--log-level", "VERBOSE",
			"--max-log-size", "62914560", "--max-tmp-logcat-file", "62914560",
			"--logcat-on-failure", "--screenshot-on-failure", "--include-test-log-tags",
			"--result-reporter:disable-result-posting", "--use-log-saver",
		}
		if testType != "gts" && testType != "vts" {
			ctsParams = append(ctsParams, "--property-check:no-throw-error")
		}
		for _, param := range ctsParams {
			cmd = append(cmd, "--cts-params", param)
		}

		// Adding a list of artifacts to skip downloading by the CTS runner.
		for _, artifact := range ctsSkipDownloadArtifacts {
			cmd = append(cmd, "--skip-download", artifact)
		}

		if resultsUpload {
			cmd = append(cmd, "--cts-params", "--result-reporter:no-disable",
				"--ants-result-reporter:create-local-invocation",
				"--ants-result-reporter:save-metric-file")

			// Extract AnTS invocation id and WorkUnit ID from execution metadata.
			if AnTsInvID, err := extractMetadataFlag(metadata, "ants_invocation_id"); err == nil {
				cmd = append(cmd, "--invocation-data", fmt.Sprintf("invocation_id=%s", AnTsInvID))
				logger.Printf("Got invocation id: %s", AnTsInvID)
			}
			if AnTsWorkUnitID, err := extractMetadataFlag(metadata, "ants_work_unit_id"); err == nil {
				cmd = append(cmd, "--invocation-data", fmt.Sprintf("work_unit_id=%s", AnTsWorkUnitID))
				logger.Printf("Got work unit id: %s", AnTsWorkUnitID)
			}
		} else {
			cmd = append(cmd, "--cts-params", "--result-reporter:disable")
		}
	}

	if retryFailedRuns {
		cmd = append(cmd, "--max-testcase-run-count", "3", "--retry-isolation-grade", "FULLY_ISOLATED",
			"--retry-strategy", "RETRY_ANY_FAILURE", "--module-preparation-retry")
	} else {
		cmd = append(cmd, "--max-testcase-run-count", "1", "--retry-strategy", "NO_RETRY")
	}

	var buildInfoReported = false

	for _, t := range tests {
		testName := formatTestName(t.TestCase.Name)
		branch, target, build := extractBuildInfo(t.TestCase)
		if len(branch) > 0 && len(target) > 0 && len(build) > 0 && !buildInfoReported {
			cmd = append(cmd, "--branch", branch, "--build-flavor", target,
				"--build-id", build)
			if !isAospTradefed() {
				cmd = append(cmd, "--cts-branch", branch, "--cts-build-flavor", target,
					"--cts-build-id", build)
			}
			buildInfoReported = true
		}
		// quotedTestName := fmt.Sprintf("\"%s\"", testName)
		logger.Println("Running provided test: ", testName,
			" for branch/target/build: ", branch, "/", target, "/", build)

		if isAospTradefed() {
			cmd = append(cmd, "--include-filter", testName)
		} else {
			cmd = append(cmd, "--cts-params", "--compatibility:include-filter", "--cts-params", testName)
		}
	}

	// Add devices
	for _, s := range serials {
		cmd = append(cmd, "-s", s)
	}

	return cmd
}
