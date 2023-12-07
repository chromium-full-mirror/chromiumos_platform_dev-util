// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package driver

import (
	"bytes"
	"chromiumos/test/execution/errors"
	"fmt"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.chromium.org/chromiumos/config/go/test/api"
)

var (
	buff       bytes.Buffer
	MockLogger = log.New(&buff, "", log.LstdFlags)
)

func TestGenMoblyArgs(t *testing.T) {
	for _, tc := range []struct {
		input    *api.TestCaseMetadata
		expected []string
	}{
		{
			input: &api.TestCaseMetadata{
				TestCase: &api.TestCase{
					Id: &api.TestCase_Id{
						Value: "case_Id1",
					},
					Name: "test_case_name1",
					Tags: []*api.TestCase_Tag{
						{Value: "mobly_mh_target://testing/mobly/platforms/cros/examples:advanced_features_example_test"},
					},
				},
			},
			expected: []string{"-target_name",
				"//testing/mobly/platforms/cros/examples:advanced_features_example_test",
				"-test_flags",
				"--notest_loasd --test_arg=--run_as=chromeos-engprod --test_arg=--dimension_label=chromeos-engprod-demo"},
		},
	} {
		actual, err := genMoblyArgs(tc.input)
		if err != nil {
			t.Errorf("failed to execute genMoblyArgs (%v):%v", tc.input, err)
		}
		if diff := cmp.Diff(actual, tc.expected); diff != "" {
			t.Errorf(diff)
		}
	}
}

func TestGenMoblyArgsNegative(t *testing.T) {
	for _, tc := range []struct {
		input    *api.TestCaseMetadata
		expected error
	}{
		{
			input: &api.TestCaseMetadata{
				TestCase: &api.TestCase{
					Id: &api.TestCase_Id{
						Value: "case_Id1",
					},
					Name: "test_case_name1",
					Tags: []*api.TestCase_Tag{
						{Value: "mobly_mh_target_suite://testing/mobly/platforms/cros/examples:advanced_features_example_test"},
					},
				},
			},
			expected: errors.NewStatusError(errors.InvalidArgument, fmt.Errorf("%v", "could not find mobly_mh_target")),
		},
	} {
		_, err := genMoblyArgs(tc.input)
		if err == nil {
			t.Errorf("genMoblyArgs should throw an error. input: %v expected error: %v", tc.input, tc.expected)
		}
	}
}

func TestGetTestCaseMetadataTags(t *testing.T) {
	for _, tc := range []struct {
		input    *api.TestCaseMetadata
		expected map[string][]string
	}{
		{
			input: &api.TestCaseMetadata{
				TestCase: &api.TestCase{
					Id: &api.TestCase_Id{
						Value: "case_Id1",
					},
					Name: "test_case_name1",
					Tags: []*api.TestCase_Tag{
						{Value: "mobly_mh_target://testing/mobly/platforms/cros/examples:advanced_features_example_test"},
						{Value: "suite:tast-informational"},
						{Value: "suite:tast-bvt"},
					},
				},
			},
			expected: map[string][]string{
				"mobly_mh_target": []string{"//testing/mobly/platforms/cros/examples:advanced_features_example_test"},
				"suite":           []string{"tast-informational", "tast-bvt"},
			},
		},
		{
			input: &api.TestCaseMetadata{
				TestCase: &api.TestCase{
					Id: &api.TestCase_Id{
						Value: "case_Id2",
					},
					Name: "IncorrectTagFormat",
					Tags: []*api.TestCase_Tag{
						{Value: "tast-informational"},
					},
				},
			},
			expected: map[string][]string{},
		},
	} {
		actual := getTestCaseMetadataTags(tc.input)
		if diff := cmp.Diff(actual, tc.expected); diff != "" {
			t.Errorf(diff)
		}
	}
}
