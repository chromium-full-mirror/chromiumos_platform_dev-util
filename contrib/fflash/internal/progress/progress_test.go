// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package progress

import (
	qt "github.com/frankban/quicktest"
	"math"
	"testing"
)

func TestFormatUnit(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{987654, "988K"},
		{999499, "999K"},
		{math.Nextafter(999500, 0), "999K"},
		{999500, "1.00M"},
		{999999, "1.00M"},
		{1000000, "1.00M"},
		{1000001, "1.00M"},
		{1234567, "1.23M"},
	}

	c := qt.New(t)
	for _, test := range tests {
		c.Check(formatUnit(test.input), qt.Equals, test.expected)
	}
}
