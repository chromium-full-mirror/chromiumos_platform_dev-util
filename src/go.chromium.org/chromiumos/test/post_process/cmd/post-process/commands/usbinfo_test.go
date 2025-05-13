// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"io"
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"google.golang.org/protobuf/types/known/anypb"
)

const UsbInfoFile = "test_data/usb_info.json"

func TestUsbInfo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	test := "firmware.ECPDCCD.normal"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`usbInfos works`, t, func() {
		usbFiles := map[string]string{
			test: UsbInfoFile,
		}
		got, _ := usbInfos(ctx, emptyLogger, usbFiles)
		want, _ := anypb.New(&artifact.DutInfo_UsbInfo{
			PowerDeliveryPortServo: "0",
			PowerDeliveryPortCount: 2,
		})
		So(got, ShouldResemble, want)
	})

	Convey(`Return nil if no USB info file is found`, t, func() {
		usbFiles := map[string]string{
			test: "",
		}
		got, _ := usbInfos(ctx, emptyLogger, usbFiles)
		want := (*anypb.Any)(nil)
		So(got, ShouldResemble, want)
	})
}
