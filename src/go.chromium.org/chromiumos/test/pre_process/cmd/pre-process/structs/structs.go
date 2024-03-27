// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package structs

// SignalFormat is the common format all policies/interface must return for the main filter service to use.
type SignalFormat struct {
	Runs           int
	Failruns       int
	Passrate       float64
	Signal         bool
	PassrateRecent float64
	RunsRecent     int
}
