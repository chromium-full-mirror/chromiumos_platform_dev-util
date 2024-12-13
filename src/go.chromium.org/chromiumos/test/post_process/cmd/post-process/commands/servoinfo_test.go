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
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"google.golang.org/protobuf/types/known/anypb"
)

const TastServoInfoFile = "test_data/servo_info_tast.json"
const TautoServoInfoFile = "test_data/servo_info_tauto.json"

func TestServoInfos(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`ServoInfo works for tast test`, t, func() {
		wantServoInfo, _ := anypb.New(&artifact.BuildMetadata_ServoInfo{
			ServodVersion: "v1.0.2388-4e980b5d 2024-10-31 14:47:15",
			ServoType:     "servo_v4p1_with_ccd_cr50",
		})

		gotServoInfo, _ := ReadServoInfo(ctx, emptyLogger, TastServoInfoFile)

		So(gotServoInfo, ShouldResemble, wantServoInfo)
	})

	Convey(`ServoInfo works for tauto test`, t, func() {
		wantServoInfoTauto, _ := anypb.New(&artifact.BuildMetadata_ServoInfo{
			ServodVersion: "v1.0.2388-4e980b5d 2024-10-31 14:47:15",
			ServoType:     "servo_v4p1_with_ccd_cr50",
			ServoVersions: "fizz-labstation-release/R131-16063.25.0,servo_v4p1_v2.0.24152-0b36eb51a,0.5.261/cr50_v4.11_mp.76-bc730d0f41",
		})

		gotServoInfoTauto, _ := ReadServoInfo(ctx, emptyLogger, TautoServoInfoFile)

		So(gotServoInfoTauto, ShouldResemble, wantServoInfoTauto)
	})

	Convey(`Return empty if no servo info file is found`, t, func() {
		wantEmptyServoInfo, _ := anypb.New(&artifact.BuildMetadata_ServoInfo{})
		gotEmptyServoInfo, _ := ReadServoInfo(ctx, emptyLogger, "")

		So(gotEmptyServoInfo, ShouldResemble, wantEmptyServoInfo)
	})
}

func TestFindServoPath(t *testing.T) {
	t.Parallel()

	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`Returns empty if no tast servo path`, t, func() {
		gotServoPath := TastServoFilePath(emptyLogger, "n/a")

		So(gotServoPath, ShouldBeEmpty)
	})
}
