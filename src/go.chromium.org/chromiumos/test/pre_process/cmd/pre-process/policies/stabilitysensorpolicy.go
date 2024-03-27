// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package policies

import (
	"go.chromium.org/chromiumos/test/pre_process/cmd/pre-process/interfaces"
	"go.chromium.org/chromiumos/test/pre_process/cmd/pre-process/structs"
	"log"
)

// StabilityFromStabilitySensor will get Stability info from the StabilitySensor endpoint.
func StabilityFromStabilitySensor() (map[string]structs.SignalFormat, error) {
	data, err := interfaces.CallForStability()
	if err != nil {
		return nil, err
	}

	return formatStabilityData(data)
}

func formatStabilityData(raw []interfaces.SensorFormat) (map[string]structs.SignalFormat, error) {
	log.Println("STABILITY INTERFACE NOT IMPLEMENTED.")
	data := make(map[string]structs.SignalFormat)
	for _, item := range raw {

		status := false
		if item.Test_stats.Stats == "STABLE" {
			status = true
		}
		data[item.Test_id] = structs.SignalFormat{Signal: status}
	}
	return data, nil
}
