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

const StressInfoFile = "test_data/stress_info.json"

func TestStressInfo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	test := "firmware.ECPDCCD.normal"
	emptyLogger := log.New(io.Discard, "", 0)

	Convey(`stressTestInfos works`, t, func() {
		stressTestFiles := map[string]string{
			test: StressInfoFile,
		}
		got := stressTestInfos(ctx, emptyLogger, stressTestFiles)
		wantStressTestInfo, _ := anypb.New(&artifact.StressTestInfo{
			Iterations: 10,
		})
		want := map[string]*anypb.Any{
			test: wantStressTestInfo,
		}
		So(got, ShouldResemble, want)
	})

	Convey(`Return nil if no stress info file is found`, t, func() {
		stressTestFiles := map[string]string{
			test: "",
		}
		got := stressTestInfos(ctx, emptyLogger, stressTestFiles)
		wantStressInfo, _ := anypb.New(&artifact.StressTestInfo{})
		want := map[string]*anypb.Any{
			test: wantStressInfo,
		}
		So(got, ShouldResemble, want)
	})
}
