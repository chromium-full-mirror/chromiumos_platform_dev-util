// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package v2 holds various provisioning services.
package v2

// Use  ~/chromiumos/src/platform/dev/fast_build.sh -G to generate mocks.
//go:generate mockgen -source common-utils/service_adapter_interface.go -destination mock-common-utils/serviceadaptermock.go -copyright_file copyright2022.txt
