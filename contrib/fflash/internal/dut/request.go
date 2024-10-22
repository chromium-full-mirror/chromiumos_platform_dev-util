// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"time"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/payload"
)

// Request contains everything needed to perform a flash.
type Request struct {
	// Base time when the flash started, for logging.
	ElapsedTimeWhenSent time.Duration

	Payload *payload.OneOf

	FlashOptions
}

// FlashOptions for Request.
// Unlike Request.Bucket, Request.Directory, these are determined solely by
// parsing the command line without further processing.
type FlashOptions struct {
	DisableRootfsVerification bool // whether to disable rootfs verification
	ClobberStateful           bool // whether to clobber the stateful partition
	ClearTpmOwner             bool // whether to clean tpm owner on reboot
}

type Result struct {
	RetryDisableRootfsVerification bool
	RetryClearTpmOwner             bool
}
