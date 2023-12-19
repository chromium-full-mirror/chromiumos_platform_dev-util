// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package driver

import (
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestNewCrosierDriver ensures the crosier driver is built correctly
func TestNewCrosierDriver(t *testing.T) {
	l := new(log.Logger)

	crosierDriver := NewCrosierDriver(l)

	if crosierDriver.logger != l {
		t.Errorf("Got unexpected logger (%v), want (%v)", crosierDriver.logger, l)
	}
}

// TestGtestName ensures the proper name is associated with the driver
func TestCrosierName(t *testing.T) {
	const expectedName string = "crosier"

	l := new(log.Logger)

	crosierDriver := NewCrosierDriver(l)
	driverName := crosierDriver.Name()
	if diff := cmp.Diff(driverName, expectedName); diff != "" {
		t.Errorf("Got unexpected argument from CrosierDriver.Name (-got +want):\n%s\n%v\n--\n%v\n", diff, driverName, expectedName)
	}
}
