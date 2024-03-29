// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package common provide command utilities and variables for all components in
// cros-test to use.
package common

import (
	"bufio"
	"fmt"
	"io"
	"log"

	"go.chromium.org/chromiumos/config/go/test/api"

	"go.chromium.org/chromiumos/test/execution/cmd/cros-test/internal/device"
	"go.chromium.org/chromiumos/test/execution/errors"
)

// TestScanner makes a scanner to read from test streams.
func TestScanner(stream io.Reader, logger *log.Logger, harness string) {
	const maxCapacity = 4096 * 1024
	scanner := bufio.NewScanner(stream)
	// Expand the buffer size to avoid deadlocks on heavy logs
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		logger.Printf("[%v] %v", harness, scanner.Text())
	}
	if scanner.Err() != nil {
		logger.Println("Failed to read pipe: ", scanner.Err())
	}
}

// ExtraArgs upacks the extra arguements from test reqeusts.
func ExtraArgs(req *api.CrosTestRequest) (args []*api.Arg) {
	for _, t := range req.GetTestSuites() {
		for _, a := range t.GetExecutionMetadata().GetArgs() {
			args = append(args, a)
		}
	}
	return args
}

// UnpackMetadata unpacks the Any metadata field into AutotestExecutionMetadata
func UnpackMetadata(req *api.CrosTestRequest) (args []*api.Arg, resultsRootDir string, err error) {
	if req.Metadata == nil {
		return nil, "", nil
	}
	if autotestMetaData, resultsRootDir, err := unpackAutotestMetadata(req); err == nil {
		return autotestMetaData, resultsRootDir, nil
	}
	if tastMetaData, resultsRootDir, err := unpackTastMetadata(req); err == nil {
		return tastMetaData, resultsRootDir, nil
	}
	return unpackUnifiedMetadata(req)
}

// unpackAutotestMetadata unpacks the Any metadata field into AutotestExecutionMetadata
func unpackAutotestMetadata(req *api.CrosTestRequest) (args []*api.Arg, resultRootDir string, err error) {
	if req.Metadata == nil {
		return nil, "", nil
	}
	var m api.AutotestExecutionMetadata
	if err := req.Metadata.UnmarshalTo(&m); err != nil {
		return nil, "", fmt.Errorf("improperly formatted input proto metadata, %s", err)
	}
	for _, a := range m.GetArgs() {
		args = append(args, &api.Arg{Flag: a.GetFlag(), Value: a.GetValue()})
	}
	return args, m.GetResultsSubDir(), nil
}

// unpackTastMetadata unpacks the Any metadata field into TastExecutionMetadata
func unpackTastMetadata(req *api.CrosTestRequest) (args []*api.Arg, resultRootDir string, err error) {
	if req.Metadata == nil {
		return nil, "", nil
	}
	var m api.TastExecutionMetadata
	if err := req.Metadata.UnmarshalTo(&m); err != nil {
		return nil, "", fmt.Errorf("improperly formatted input proto metadata, %s", err)
	}
	for _, a := range m.GetArgs() {
		args = append(args, &api.Arg{Flag: a.GetFlag(), Value: a.GetValue()})
	}
	return args, "", nil
}

// unpackUnifiedMetadata unpacks the Any metadata field into TastExecutionMetadata
func unpackUnifiedMetadata(req *api.CrosTestRequest) (args []*api.Arg, resultRootDir string, err error) {
	if req.Metadata == nil {
		return nil, "", nil
	}
	var m api.ExecutionMetadata
	if err := req.Metadata.UnmarshalTo(&m); err != nil {
		return nil, "", fmt.Errorf("improperly formatted input proto metadata, %s", err)
	}
	for _, a := range m.GetArgs() {
		args = append(args, &api.Arg{Flag: a.GetFlag(), Value: a.GetValue()})
	}
	return args, "", nil
}

// Companions separates devices to ChromeOS devices and Android devices.
func Companions(devices []*api.CrosTestRequest_Device) (chromeOSCompanions []*device.DutInfo,
	AndriodCompanions []*device.AndroidInfo, err error) {

	for i, c := range devices {
		if c.Dut.GetChromeos() != nil {
			info, err := device.FillDUTInfo(c, fmt.Sprintf("cd%d", i+1))
			if err != nil {
				return nil, nil, errors.NewStatusError(errors.InvalidArgument,
					fmt.Errorf("cannot get address from companion device: %v", c))
			}
			chromeOSCompanions = append(chromeOSCompanions, info)
			continue
		}
		if c.Dut.GetAndroid() != nil {
			androidInfo := device.FillAndroidInfo(c)
			AndriodCompanions = append(AndriodCompanions, androidInfo)
		}
	}
	return chromeOSCompanions, AndriodCompanions, nil
}
