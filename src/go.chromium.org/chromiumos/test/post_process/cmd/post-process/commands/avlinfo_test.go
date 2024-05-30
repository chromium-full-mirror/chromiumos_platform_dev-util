// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"io"
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	. "go.chromium.org/luci/common/testing/assertions"
)

const AvlInfoFile = "test_data/avl_info.json"

func TestGetAvlInfo(t *testing.T) {
	t.Parallel()

	test := "storage.LowPowerStateResidence"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`GetAvlInfo works`, t, func() {
		want := &api.GetAvlInfoResponse{
			AvlInfos: map[string]*api.AvlInfo{
				test: {
					AvlPartModel:     "0x0000f5 MMC32G",
					AvlPartFirmware:  "0xa200000000000000",
					AvlComponentType: "storage",
				},
			},
		}
		req := &api.GetAvlInfoRequest{
			AvlFiles: map[string]string{
				test: AvlInfoFile,
			},
		}

		got, err := GetAvlInfo(emptyLogger, req)

		So(err, ShouldBeNil)
		So(got, ShouldResembleProto, want)
	})

	Convey(`Return empty if no avl info file is found`, t, func() {
		want := &api.GetAvlInfoResponse{
			AvlInfos: map[string]*api.AvlInfo{
				test: {},
			},
		}
		req := &api.GetAvlInfoRequest{
			AvlFiles: map[string]string{
				test: "",
			},
		}

		got, err := GetAvlInfo(emptyLogger, req)

		So(err, ShouldBeNil)
		So(got, ShouldResembleProto, want)
	})
}
