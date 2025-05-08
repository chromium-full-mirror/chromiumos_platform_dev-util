// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestReadProtoJsonFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey(`ReadProtoJsonFile works`, t, func() {
		want := &api.AvlInfo{
			AvlPartModel:     "0x0000f5 MMC32G",
			AvlPartFirmware:  "0xa200000000000000",
			AvlComponentType: "storage",
		}
		avlInfoFile := "test_data/avl_info.json"

		got := &api.AvlInfo{}
		err := ReadProtoJSONFile(ctx, avlInfoFile, got)

		So(err, ShouldBeNil)
		So(got, ShouldResembleProto, want)
	})
}

func TestReadJSONFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey(`ReadJSONFile works`, t, func() {
		avlInfoFile := "test_data/avl_info.json"
		want := map[string]interface{}{
			"avl_part_model":     "0x0000f5 MMC32G",
			"avl_part_firmware":  "0xa200000000000000",
			"avl_component_type": "storage",
		}

		bytes, err := ReadJSONFile(ctx, avlInfoFile)
		var got map[string]interface{}
		err = json.Unmarshal(bytes, &got)
		So(err, ShouldBeNil)
		So(got, ShouldResemble, want)
	})
}
