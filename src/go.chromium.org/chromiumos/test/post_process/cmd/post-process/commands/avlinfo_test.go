// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"io"
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	. "go.chromium.org/luci/common/testing/assertions"
)

const AvlInfoFile = "test_data/avl_info.json"

func TestAvlInfos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	test := "storage.LowPowerStateResidence"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`AVLInfos works`, t, func() {
		want := map[string]*api.AvlInfo{
			test: {
				AvlPartModel:     "0x0000f5 MMC32G",
				AvlPartFirmware:  "0xa200000000000000",
				AvlComponentType: "storage",
			},
		}
		avlFiles := map[string]string{
			test: AvlInfoFile,
		}

		got := avlInfos(ctx, emptyLogger, avlFiles)

		So(len(got), ShouldEqual, 1)
		So(got[test], ShouldResembleProto, want[test])
	})

	Convey(`Return empty if no avl info file is found`, t, func() {
		want := map[string]*api.AvlInfo{
			test: {},
		}
		avlFiles := map[string]string{
			test: "",
		}

		got := avlInfos(ctx, emptyLogger, avlFiles)

		So(len(got), ShouldEqual, 1)
		So(got[test], ShouldResembleProto, want[test])
	})
}
