// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	common_utils "chromiumos/test/provision/v2/common-utils"
	"chromiumos/test/provision/v2/cros-fw-provision/cli"
	firmwareservice "chromiumos/test/provision/v2/cros-fw-provision/service"
	state_machine "chromiumos/test/provision/v2/cros-fw-provision/state-machine"
	mock_common_utils "chromiumos/test/provision/v2/mock-common-utils"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	conf "go.chromium.org/chromiumos/config/go"
	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/api"
)

type curlMatcher struct {
	target      string
	fileMatcher gomock.Matcher
	log         *log.Logger
}

func newCurlMatcher(target string, fileMatcher gomock.Matcher, log *log.Logger) curlMatcher {
	return curlMatcher{
		target:      target,
		fileMatcher: fileMatcher,
		log:         log,
	}
}
func (c curlMatcher) Matches(x interface{}) bool {
	args, ok := x.([]string)
	if ok {
		if len(args) != 4 {
			return false
		}
		if args[0] != "-S" || args[1] != "-o" || args[2] != c.target {
			return false
		}
		idx := strings.Index(args[3], "?file=")
		if idx < 0 {
			return false
		}
		return c.fileMatcher.Matches(args[3][idx+6:])
	}
	return false
}

func (c curlMatcher) String() string {
	return fmt.Sprintf("-S -o %s url(%s)", c.target, c.fileMatcher)
}

func TestDetailedRequestSSHStates(t *testing.T) {
	fakeGSPath := "gs://test-archive.tar.gz"

	apImageWithinArchive := "image.bin"
	ecImageWithinArchive := "ec.bin"
	npcxImageWithinArchive := "npcx_monitor.bin"

	makeRequest := func(main_rw, main_ro, ec_ro bool) *api.ProvisionFirmwareRequest {
		fakePayload := &build_api.FirmwarePayload{FirmwareImage: &build_api.FirmwarePayload_FirmwareImagePath{FirmwareImagePath: &conf.StoragePath{HostType: conf.StoragePath_GS, Path: fakeGSPath}}}
		FirmwareConfig := build_api.FirmwareConfig{}
		if main_rw {
			FirmwareConfig.MainRwPayload = fakePayload
		}
		if main_ro {
			FirmwareConfig.MainRoPayload = fakePayload
		}
		if ec_ro {
			FirmwareConfig.EcRoPayload = fakePayload
		}
		req := &api.ProvisionFirmwareRequest{
			Board:           "test_board",
			Model:           "test_model",
			DetailedRequest: &FirmwareConfig,
		}

		return req
	}

	checkStateName := func(st common_utils.ServiceState, expectedStateName string) {
		if st == nil {
			if len(expectedStateName) > 0 {
				t.Fatalf("expected state %v. got: nil state", expectedStateName)
			}
			return
		}
		stateName := st.Name()
		if stateName != expectedStateName {
			t.Fatalf("expected state %v. got: %v", expectedStateName, stateName)
		}
	}

	type TestCase struct {
		// inputs
		main_rw, main_ro, ec_ro bool
		// expected outputs
		updateRw, updateRo     bool
		expectConstructorError bool
	}

	testCases := []TestCase{
		{ /*in*/ false, false, false /*out*/, false, false /*err*/, true},
		{ /*in*/ true, false, false /*out*/, true, false /*err*/, false},
		{ /*in*/ false, true, false /*out*/, false, true /*err*/, false},
		{ /*in*/ false, false, true /*out*/, false, true /*err*/, false},
		{ /*in*/ false, true, true /*out*/, false, true /*err*/, false},
		{ /*in*/ true, true, true /*out*/, true, true /*err*/, false},
		{ /*in*/ true, true, false /*out*/, true, true /*err*/, false},
	}

	// Set up the mock.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sam := mock_common_utils.NewMockServiceAdapterInterface(ctrl)
	log, _ := cli.SetUpLog(cli.DefaultLogDirectory)

	for _, testCase := range testCases {
		// Create FirmwareService.
		ctx := context.Background()
		req := makeRequest(testCase.main_rw, testCase.main_ro, testCase.ec_ro)
		log.Printf("  Test Case: %#v\n  Request: {%s}", testCase, req.String())
		fakeCacheServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.String(), "/stage?") {
				fmt.Fprintf(w, "ok\r\n")
				return
			}
			if r.URL.String() == "/extract/test-archive.tar.gz?file=config.yaml" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			t.Fatalf("Unexpected cache server call to %q", r.URL.String())
		}))
		defer fakeCacheServer.Close()
		cacheServer, err := url.Parse(fakeCacheServer.URL)
		if err != nil {
			t.Fatal("Failed to parse url: ", err)
		}
		fws, err := firmwareservice.NewFirmwareService(
			ctx,
			sam,
			nil,
			*cacheServer,
			req,
		)
		// Check if init error is expected/got.
		if err != nil {
			if testCase.expectConstructorError {
				continue
			}
			t.Fatalf("failed to create FirmwareService with test case %#v: %v", testCase, err)
		}
		if err == nil && testCase.expectConstructorError {
			t.Fatalf("expected constructor error for test case %#v. got: %v", testCase, err)
		}
		// Check expected states.
		if testCase.updateRo != fws.UpdateRo() {
			t.Fatalf("test case %#v expects updateRo to be %v. got: %v.", testCase, testCase.updateRo, fws.UpdateRo())
		}
		if testCase.updateRw != fws.UpdateRw() {
			t.Fatalf("test case %#v expects updateRw to be %v. got: %v.", testCase, testCase.updateRw, fws.UpdateRw())
		}

		// Start with the first state of the service.
		st := state_machine.NewFirmwarePrepareState(fws)
		// Confirm state name is Prepare.
		checkStateName(st, state_machine.PrepareStateName)

		// Set mock expectations.
		gomock.InOrder(
			sam.EXPECT().RunCmd(gomock.Any(), "mktemp", gomock.Any()).Return("/tmp/someTempDir\n", nil),
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Not(apImageWithinArchive), log)).AnyTimes().Return("not found\n", fmt.Errorf("execution fail: 1")),
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq(apImageWithinArchive), log)).MaxTimes(2).Return("success\n", nil),
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/ec.bin", gomock.Not(ecImageWithinArchive), log)).AnyTimes().Return("not found\n", fmt.Errorf("execution fail: 1")),
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/ec.bin", gomock.Eq(ecImageWithinArchive), log)).MaxTimes(1).Return("success\n", nil),
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/npcx_monitor.bin", gomock.Eq(npcxImageWithinArchive), log)).MaxTimes(1).Return("success\n", nil),
		)

		// Execute the state and proceed.
		_, _, err = st.Execute(ctx, log)
		if err != nil {
			t.Fatal(err)
		}
		st = st.Next()

		if testCase.updateRo {
			// Confirm state name is RO.
			checkStateName(st, state_machine.UpdateRoStateName)

			// Set mock expectations.
			expectedFutilityImageArgs := []string{}
			if testCase.main_ro {
				expectedFutilityImageArgs = append(expectedFutilityImageArgs, "--image=/tmp/someTempDir/bios.bin")
			}
			if testCase.ec_ro {
				expectedFutilityImageArgs = append(expectedFutilityImageArgs, "--ec_image=/tmp/someTempDir/ec.bin")
			}
			expectedFutilityArgs := append([]string{"update", "--mode=recovery"}, expectedFutilityImageArgs...)
			expectedFutilityArgs = append(expectedFutilityArgs, "--wp=0")
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "futility", expectedFutilityArgs).Return("", nil),
			)

			// Execute the state and proceed.
			_, _, err := st.Execute(ctx, log)
			if err != nil {
				t.Fatal(err)
			}
			st = st.Next()
		}

		if testCase.updateRw {
			// Confirm state name is RW.
			checkStateName(st, state_machine.UpdateRwStateName)

			// Set mock expectations.
			expectedFutilityArgs := []string{"update", "--mode=recovery", "--image=/tmp/someTempDir/bios.bin", "--wp=1"}
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "futility", expectedFutilityArgs).Return("", nil),
			)

			// Execute the state and proceed.
			_, _, err := st.Execute(ctx, log)
			if err != nil {
				t.Fatal(err)
			}
			st = st.Next()
		}

		// Confirm state name is postinstall.
		checkStateName(st, state_machine.PostInstallStateName)
		// Set mock expectations.
		gomock.InOrder(
			sam.EXPECT().DeleteDirectory(gomock.Any(), "/tmp/someTempDir").Return(nil),
			sam.EXPECT().Restart(gomock.Any()).Return(nil),
			sam.EXPECT().RunCmd(gomock.Any(), "true", nil).Return("", nil),
		)
		// Execute the state and proceed.
		_, _, err = st.Execute(ctx, log)
		if err != nil {
			t.Fatal(err)
		}
		st = st.Next()

		// Confirm no states left.
		checkStateName(st, "")
	}
}
