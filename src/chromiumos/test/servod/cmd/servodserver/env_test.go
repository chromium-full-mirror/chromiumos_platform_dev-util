// Copyright 2024 The Chromium Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servodserver

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSwarmingBotID(t *testing.T) {
	t.Parallel()
	Convey("getSwarmingBotID", t, func() {
		Convey("Returns bot ID when env variable is set", func() {
			os.Setenv("SWARMING_BOT_ID", "cloudbots-12345")
			So(getSwarmingBotID(), ShouldEqual, "cloudbots-12345")
		})
		Convey("Returns empty string when bot ID env variable is not set", func() {
			os.Unsetenv("SWARMING_BOT_ID")
			So(getSwarmingBotID(), ShouldBeEmpty)
		})
	})
}

func TestIsCloudBot(t *testing.T) {
	t.Parallel()
	Convey("isCloudBot", t, func() {
		Convey("Returns true when bot ID starts with cloudbots", func() {
			os.Setenv("SWARMING_BOT_ID", "cloudbots-12345")
			So(isCloudBot(), ShouldBeTrue)
		})
		Convey("Returns false when bot ID does not start with cloudbots", func() {
			os.Setenv("SWARMING_BOT_ID", "chrome-bot-12345")
			So(isCloudBot(), ShouldBeFalse)
		})
	})
}
