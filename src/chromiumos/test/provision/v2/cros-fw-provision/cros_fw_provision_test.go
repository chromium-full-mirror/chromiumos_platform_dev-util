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
	"io"
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	conf "go.chromium.org/chromiumos/config/go"
	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/protobuf/types/known/anypb"
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
		if len(args) != 6 {
			return false
		}
		if args[0] != "-f" || args[1] != "-s" || args[2] != "-S" || args[3] != "-o" || args[4] != c.target {
			return false
		}
		idx := strings.Index(args[5], "?file=")
		if idx < 0 {
			return false
		}
		filename, err := url.QueryUnescape(args[5][idx+6:])
		if err != nil {
			panic(err)
		}
		return c.fileMatcher.Matches(filename)
	}
	return false
}

func (c curlMatcher) String() string {
	return fmt.Sprintf("-f -s -S -o %s url(%s)", c.target, c.fileMatcher)
}

type yamlReader struct {
	*strings.Reader
}

func (yr yamlReader) Close() error {
	return nil
}

func TestDetailedRequestSSHStates(t *testing.T) {
	fakeGSPath := "gs://chromeos-image-archive/board-firmware-branch/R123-12345.0.0/board/firmware_from_source.tar.bz2"

	makeRequest := func(main_rw, main_ro, ec_ro bool) *api.InstallRequest {
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
		any, err := anypb.New(&api.FirmwareProvisionInstallMetadata{
			FirmwareConfig: &FirmwareConfig,
		})
		if err != nil {
			t.Fatal("any.new failed: ", err)
		}

		req := &api.InstallRequest{
			Metadata: any,
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
		configYAML              string
		// These are the first files that will exist in the archive.
		apImageWithinArchive   string
		ecImageWithinArchive   string
		npcxImageWithinArchive string
		// expected outputs
		updateRw, updateRo     bool
		expectConstructorError bool
	}

	testCases := []TestCase{
		{ /*in*/ false, false, false, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, false, false /*err*/, true},
		{ /*in*/ true, false, false, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, true, false /*err*/, false},
		{ /*in*/ false, true, false, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, false, true /*err*/, false},
		{ /*in*/ false, false, true, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, false, true /*err*/, false},
		{ /*in*/ false, true, true, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, false, true /*err*/, false},
		{ /*in*/ true, true, true, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, true, true /*err*/, false},
		{ /*in*/ true, true, false, "", "image.bin", "ec.bin", "npcx_monitor.bin" /*out*/, true, true /*err*/, false},
		{ /*in*/ true, true, true, `{
			"chromeos": {
				"configs": [{
					"firmware": {
						"build-targets": {
							"coreboot": "fromcoreboot",
							"ec": "fromec",
							"zephyr-ec": "fromzephyr"
						},
						"image-name": "fromimagename"
					},
					"name": "test_model"
				}]
			}
		}`, "image-fromcoreboot.bin", "fromzephyr/ec.bin", "fromzephyr/npcx_monitor.bin" /*out*/, true, true /*err*/, false},
	}

	log, _ := cli.SetUpLog(cli.DefaultLogDirectory)

	for _, testCase := range testCases {
		// Set up the mock.
		ctrl := gomock.NewController(t)
		sam := mock_common_utils.NewMockServiceAdapterInterface(ctrl)

		// Create FirmwareService.
		ctx := context.Background()
		req := makeRequest(testCase.main_rw, testCase.main_ro, testCase.ec_ro)
		log.Printf("  Test Case: %#v\n  Request: {%s}", testCase, req.String())
		cacheServer := url.URL{
			Scheme: "http",
			Host:   "1.2.3.4:5678",
		}
		fws, err := firmwareservice.NewFirmwareService(
			ctx,
			sam,
			nil,
			cacheServer,
			"test_board",
			"test_model",
			false,
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
		if testCase.configYAML == "" {
			gomock.InOrder(
				sam.EXPECT().PathExists(gomock.Any(), "/usr/share/chromeos-config/yaml/config.yaml").Return(false, nil),
			)
		} else {
			var reader io.ReadCloser
			reader = &yamlReader{strings.NewReader(testCase.configYAML)}
			gomock.InOrder(
				sam.EXPECT().PathExists(gomock.Any(), "/usr/share/chromeos-config/yaml/config.yaml").Return(true, nil),
				sam.EXPECT().FetchFile(gomock.Any(), "/usr/share/chromeos-config/yaml/config.yaml").Return(reader, nil),
			)
		}
		stagingURL := "http://1.2.3.4:5678/stage?archive_url=gs%3A%2Fchromeos-image-archive%2Fboard-firmware-branch%2FR123-12345.0.0%2Fboard&files=firmware_from_source.tar.bz2"
		mkdirCall := sam.EXPECT().RunCmd(gomock.Any(), "mktemp", gomock.Eq([]string{"-d", "--tmpdir", "cros-fw-provision.XXXXXXXXX.board"})).Return("/tmp/someTempDir\n", nil)

		if testCase.main_ro {
			stagingCall := sam.EXPECT().RunCmd(gomock.Any(), "curl", gomock.Eq([]string{"-f", "-s", "-S", stagingURL})).MinTimes(1).Return("", nil)
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq("image-test_model.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq("image-test_board.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq(testCase.apImageWithinArchive), log)).
				After(stagingCall).After(mkdirCall).Times(1).Return("success\n", nil)
		}
		if testCase.ec_ro {
			stagingCall := sam.EXPECT().RunCmd(gomock.Any(), "curl", gomock.Eq([]string{"-f", "-s", "-S", stagingURL})).MinTimes(1).Return("", nil)
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/ec.bin", gomock.Eq("test_board/ec.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/ec.bin", gomock.Eq("test_model/ec.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/ec.bin", gomock.Eq(testCase.ecImageWithinArchive), log)).
				After(stagingCall).After(mkdirCall).Times(1).Return("success\n", nil)
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/npcx_monitor.bin", gomock.Eq(testCase.npcxImageWithinArchive), log)).
				After(stagingCall).After(mkdirCall).Times(1).Return("success\n", nil)
		}
		if testCase.main_rw {
			stagingCall := sam.EXPECT().RunCmd(gomock.Any(), "curl", gomock.Eq([]string{"-f", "-s", "-S", stagingURL})).MinTimes(1).Return("", nil)
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq("image-test_model.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq("image-test_board.bin"), log)).
				After(stagingCall).After(mkdirCall).MaxTimes(1).Return("not found\n", fmt.Errorf("execution fail: 1"))
			sam.EXPECT().RunCmd(gomock.Any(), "curl", newCurlMatcher("/tmp/someTempDir/bios.bin", gomock.Eq(testCase.apImageWithinArchive), log)).
				After(stagingCall).After(mkdirCall).Times(1).Return("success\n", nil)
		}

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
			expectedFutilityManifestArgs := append([]string{"update", "--manifest"}, expectedFutilityImageArgs...)
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "futility", expectedFutilityManifestArgs).Return(`{
					"default": {
						"host": {
							"versions": { "ro": "roversion", "rw": "roversioninrw" }
						},
						"ec": {
							"versions": { "ro": "ecroversion", "rw": "ecroversioninrw" }
						}
					}
				}`, nil),
			)
			if testCase.main_ro {
				gomock.InOrder(
					sam.EXPECT().RunCmd(gomock.Any(), "cbfstool", []string{"/tmp/someTempDir/bios.bin", "extract", "-r", "FW_MAIN_A", "-n", "ecrw.hash", "-f", "/tmp/someTempDir/bios.bin-ecrw.hash"}).Return("", nil),
					sam.EXPECT().RunCmd(gomock.Any(), "od", []string{"-A", "n", "-t", "x1", "/tmp/someTempDir/bios.bin-ecrw.hash"}).Return(
						" 38 73 52 6f 4f f3 af 90 62 68 7c 30 91 13 09 ee\n 12 a5 ce ec 23 55 19 a8 ba f6 cd be 63 61 ba 92\n", nil),
				)
			}
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
			expectedFutilityManifestArgs := []string{"update", "--manifest", "--image=/tmp/someTempDir/bios.bin"}
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "futility", expectedFutilityManifestArgs).Return(`{
					"default": {
						"host": {
							"versions": { "ro": "wrongro", "rw": "rwversion" }
						}
					}
				}`, nil),
				sam.EXPECT().RunCmd(gomock.Any(), "cbfstool", []string{"/tmp/someTempDir/bios.bin", "extract", "-r", "FW_MAIN_A", "-n", "ecrw.hash", "-f", "/tmp/someTempDir/bios.bin-ecrw.hash"}).Return("", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "od", []string{"-A", "n", "-t", "x1", "/tmp/someTempDir/bios.bin-ecrw.hash"}).Return(
					" 38 73 52 6f 4f f3 af 90 62 68 7c 30 91 13 09 ee\n 12 a5 ce ec 23 55 19 a8 ba f6 cd be 63 61 ba 92\n", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "futility", expectedFutilityArgs).Return("", nil),
			)

			/* OD output:
			    38 73 52 6f 4f f3 af 90 62 68 7c 30 91 13 09 ee
			    12 a5 ce ec 23 55 19 a8 ba f6 cd be 63 61 ba 92

			    ectool output:
			   status:  done
			   type:    SHA-256
			   offset:  0x00040000
			   size:    0x00033b80
			   hash:    3873526f4ff3af9062687c30911309ee12a5ceec235519a8baf6cdbe6361ba92
			*/

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
			sam.EXPECT().RunCmd(gomock.Any(), "true", nil).Return("ssh timeout\n", fmt.Errorf("execution fail: 1")),
			sam.EXPECT().RunCmd(gomock.Any(), "true", nil).Return("", nil),
		)
		if testCase.updateRo && testCase.updateRw {
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "crossystem", []string{"ro_fwid", "fwid"}).Return("roversion rwversion", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"version"}).Return("RO version: ecroversion\nRW version: ecrwversion\n", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"echash"}).Return(
					"status:  done\ntype:    SHA-256\noffset:  0x00040000\nsize:    0x00033b80\nhash:    3873526f4ff3af9062687c30911309ee12a5ceec235519a8baf6cdbe6361ba92\n", nil),
			)
		} else if testCase.updateRo {
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "crossystem", []string{"ro_fwid", "fwid"}).Return("roversion roversioninrw", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"version"}).Return("RO version: ecroversion\nRW version: ecroversioninrw\n", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"echash"}).Return(
					"status:  done\ntype:    SHA-256\noffset:  0x00040000\nsize:    0x00033b80\nhash:    3873526f4ff3af9062687c30911309ee12a5ceec235519a8baf6cdbe6361ba92\n", nil),
			)
		} else if testCase.updateRw {
			gomock.InOrder(
				sam.EXPECT().RunCmd(gomock.Any(), "crossystem", []string{"ro_fwid", "fwid"}).Return("roversion rwversion", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"version"}).Return("RO version: ecroversion\nRW version: ecrwversion\n", nil),
				sam.EXPECT().RunCmd(gomock.Any(), "ectool", []string{"echash"}).Return(
					"status:  done\ntype:    SHA-256\noffset:  0x00040000\nsize:    0x00033b80\nhash:    3873526f4ff3af9062687c30911309ee12a5ceec235519a8baf6cdbe6361ba92\n", nil),
			)
		}
		// Execute the state and proceed.
		_, _, err = st.Execute(ctx, log)
		if err != nil {
			t.Fatal(err)
		}
		st = st.Next()

		// Confirm no states left.
		checkStateName(st, "")

		// Check for missing mock calls.
		ctrl.Finish()
	}
}
