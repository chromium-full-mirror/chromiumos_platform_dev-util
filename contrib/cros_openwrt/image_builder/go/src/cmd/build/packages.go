// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package build

import (
	"fmt"

	"github.com/spf13/cobra"
)

func packagesCmd(builderOptions *CrosOpenWrtImageBuilderOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "packages",
		Short: "Compiles custom OpenWrt packages.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ib, err := NewCrosOpenWrtImageBuilder(builderOptions)
			if err != nil {
				return fmt.Errorf("failed to initialize CrosOpenWrtImageBuilder: %w", err)
			}
			if err := ib.CompileCustomPackages(cmd.Context()); err != nil {
				return fmt.Errorf("failed to compile custom OpenWrt packages: %w", err)
			}
			return nil
		},
	}
}
