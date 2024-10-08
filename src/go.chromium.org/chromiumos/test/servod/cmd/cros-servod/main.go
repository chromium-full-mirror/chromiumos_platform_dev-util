// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os"

	"go.chromium.org/chromiumos/test/servod/cmd/cros-servod/cli"
)

func main() {
	os.Exit(cli.MainInternal())
}
