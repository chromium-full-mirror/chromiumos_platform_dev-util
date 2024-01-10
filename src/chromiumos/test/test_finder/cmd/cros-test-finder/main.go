// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"os"

	"chromiumos/test/test_finder"
)

func main() {
	os.Exit(test_finder.TestFinderInternal(context.Background()))
}
