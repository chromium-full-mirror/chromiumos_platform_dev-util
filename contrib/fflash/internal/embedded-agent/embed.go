// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:build with_embedded_agent

package embeddedagent

import (
	_ "embed"
	"fmt"
)

// SelfCheck returns an error if the embeddedagent was built incorrectly
func SelfCheck() error {
	return nil
}

//go:embed dut-agent-amd64.zst
var x8664Zst []byte

//go:embed dut-agent-arm64.zst
var aarch64Zst []byte

// ExecutableForArch returns the executable binary of dut-agent for the arch.
func ExecutableForArch(arch string) ([]byte, error) {
	switch arch {
	case ArchAarch64:
		return aarch64Zst, nil
	case ArchX8664:
		return x8664Zst, nil
	default:
		return nil, fmt.Errorf("unknown arch %q", arch)
	}
}
