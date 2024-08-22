// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/chromiumos/config/go/test/api"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
)

func TestCompanions(t *testing.T) {
	androidDUT := &labapi.Dut{
		Id: &labapi.Dut_Id{Value: "android1"},
		DutType: &labapi.Dut_Android_{
			Android: &labapi.Dut_Android{
				AssociatedHostname: &labapi.IpEndpoint{Address: "127.0.0.1", Port: 2224},
				DutModel:           &labapi.DutModel{BuildTarget: "XYZ", ModelName: "pixel6"},
				SerialNumber:       "ABC123",
			},
		},
	}
	chromeOSDUT := &labapi.Dut{
		Id: &labapi.Dut_Id{Value: "chromeos1"},
		DutType: &labapi.Dut_Chromeos{
			Chromeos: &labapi.Dut_ChromeOS{
				Ssh: &labapi.IpEndpoint{Address: "127.0.0.1", Port: 2223},
			},
		},
	}
	devices := []*api.CrosTestRequest_Device{
		{
			Dut: androidDUT,
		},
		{
			Dut: chromeOSDUT,
		},
	}

	expectedCompanions := []*device.DutInfo{
		{
			Addr:  "127.0.0.1:2223",
			Role:  "cd1",
			DUT:   chromeOSDUT,
			Phase: "PHASE_UNSPECIFIED",
			Ip:    "127.0.0.1",
		},
	}
	expectedAndroids := []*device.AndroidInfo{
		{
			AssoicateAddr: "127.0.0.1:2224",
			Serial:        "ABC123",
			ModelName:     "pixel6",
			DUT:           androidDUT,
		},
	}

	companions, androids, err := Companions(devices)
	if err != nil {
		t.Error("Failed to call Companions: ", err)
	}

	if diff := cmp.Diff(companions, expectedCompanions,
		cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("Got unexpected compainons (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(androids, expectedAndroids,
		cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("Got unexpected androids (-got +want):\n%s", diff)
	}

}
