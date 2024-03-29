// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.chromium.org/chromiumos/config/go/test/api"
	testapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func GenerateInternalTestPlan(builDepsTest1 []*api.TestCase_BuildDeps, buildDepsTest2 []*api.TestCase_BuildDeps) *api.InternalTestplan {
	nissa := &testapi.Dut_ChromeOS{DutModel: &testapi.DutModel{
		BuildTarget: "nissa",
		ModelName:   "craaskov",
	}}

	octopus := &testapi.Dut_ChromeOS{DutModel: &testapi.DutModel{
		BuildTarget: "octopus",
		ModelName:   "bluebird",
	}}

	dedede := &testapi.Dut_ChromeOS{DutModel: &testapi.DutModel{
		BuildTarget: "dedede",
		ModelName:   "crota",
	}}

	android := &testapi.Dut_Android{DutModel: &testapi.DutModel{
		BuildTarget: "android",
		ModelName:   "android",
	}}

	internalTestPlan := &api.InternalTestplan{
		TestCases: []*api.CTPTestCase{
			{
				Name: "test1",
				Metadata: &api.TestCaseMetadata{
					TestCase: &api.TestCase{
						Id: &api.TestCase_Id{
							Value: "test1",
						},
						BuildDependencies: builDepsTest1,
					},
				},
				HwRequirements: []*api.HWRequirements{
					{
						HwDefinition: []*api.SwarmingDefinition{
							{
								DutInfo: &testapi.Dut{
									DutType: &testapi.Dut_Chromeos{Chromeos: octopus},
								},
							},
						},
					},
					{
						HwDefinition: []*api.SwarmingDefinition{
							{
								DutInfo: &testapi.Dut{
									DutType: &testapi.Dut_Chromeos{Chromeos: nissa},
								},
							},
						},
					},
					{
						HwDefinition: []*api.SwarmingDefinition{
							{
								DutInfo: &testapi.Dut{
									DutType: &testapi.Dut_Android_{Android: android},
								},
							},
						},
					},
				},
			},
			{
				Name: "test2",
				Metadata: &api.TestCaseMetadata{
					TestCase: &api.TestCase{
						Id: &api.TestCase_Id{
							Value: "test2",
						},
						BuildDependencies: buildDepsTest2,
					},
				},
				HwRequirements: []*api.HWRequirements{
					{
						HwDefinition: []*api.SwarmingDefinition{
							{
								DutInfo: &testapi.Dut{
									DutType: &testapi.Dut_Chromeos{Chromeos: dedede},
								},
							},
						},
					},
				},
			},
		},
		SuiteInfo: &api.SuiteInfo{
			SuiteMetadata: &api.SuiteMetadata{
				TargetRequirements: []*api.TargetRequirements{
					{
						HwRequirements: &api.HWRequirements{
							HwDefinition: []*api.SwarmingDefinition{
								{
									DutInfo: &testapi.Dut{
										DutType: &testapi.Dut_Chromeos{Chromeos: nissa},
									},
								},
							},
						},
						SwRequirement: &api.LegacySW{
							Build:   "release",
							GcsPath: "gs://chromeos-image-archive/nissa-release/R124-15808.0.0",
							KeyValues: []*api.KeyValue{
								{
									Key:   "chromeos_build",
									Value: "nissa-release/R124-15808.0.0",
								},
							},
						},
					},
					{
						HwRequirements: &api.HWRequirements{
							HwDefinition: []*api.SwarmingDefinition{
								{
									DutInfo: &testapi.Dut{
										DutType: &testapi.Dut_Chromeos{Chromeos: octopus},
									},
								},
							},
						},
						SwRequirement: &api.LegacySW{
							Build:   "release",
							GcsPath: "gs://chromeos-image-archive/octopus-release/R124-15810.0.0",
							KeyValues: []*api.KeyValue{
								{
									Key:   "chromeos_build",
									Value: "nissa-release/R124-15808.0.0",
								},
							},
						},
					},
				},
			},
			SuiteRequest: &api.SuiteRequest{
				SuiteRequest: &api.SuiteRequest_TestSuite{
					TestSuite: &api.TestSuite{
						Name: "bluetooth",
						Spec: &api.TestSuite_TestCaseTagCriteria_{
							TestCaseTagCriteria: &api.TestSuite_TestCaseTagCriteria{
								Tags: []string{"suite:bluetooth"},
							},
						},
					},
				},
				MaximumDuration: durationpb.New(142200),
				TestArgs:        "None",
				AnalyticsName:   "Bluetooth_Stable_Nightly",
			},
		},
	}
	return internalTestPlan
}

func GenerateBoardUseFlagDict(boardsToAdd []string) map[string]map[string]bool {

	useFlagDict := make(map[string]map[string]bool)

	useFlagsForNissa := []string{
		"common", "nissa",
	}

	useFlagsForOctopus := []string{
		"common", "octopus",
	}

	useFlagsForDedede := []string{
		"common", "dedede",
	}

	nissaSet := make(map[string]bool)
	for _, str := range useFlagsForNissa {
		nissaSet[str] = true
	}

	octopusSet := make(map[string]bool)
	for _, str := range useFlagsForOctopus {
		octopusSet[str] = true
	}

	dededeSet := make(map[string]bool)
	for _, str := range useFlagsForDedede {
		dededeSet[str] = true
	}

	for _, board := range boardsToAdd {
		if board == "nissa" {
			useFlagDict["nissa"] = nissaSet
		} else if board == "octopus" {
			useFlagDict["octopus"] = octopusSet
		} else if board == "dedede" {
			useFlagDict["dedede"] = dededeSet
		}
	}

	return useFlagDict

}

func JSONSerialize(any interface{}) string {
	json, _ := json.Marshal(any)
	return string(json)
}

// All test build deps expression qualify. No hwDef removed.
func TestUpdateTestCases_buildDeps_in_use_flags_dict(t *testing.T) {
	buildDepsTest1 := []*api.TestCase_BuildDeps{
		{
			Value: "common,!notpresent",
		},
	}
	internalTestPlan := GenerateInternalTestPlan(buildDepsTest1, buildDepsTest1)
	filteredInternalTestPlan := proto.Clone(internalTestPlan).(*api.InternalTestplan)
	emptyLogger := log.New(io.Discard, "", 0)
	err := updateTestCases(filteredInternalTestPlan, GenerateBoardUseFlagDict([]string{"nissa", "octopus", "dedede"}), emptyLogger)
	if err != nil {
		t.Errorf("updateTestCases failed: %v", err)
		return
	}

	assert.Equal(t, JSONSerialize(filteredInternalTestPlan), JSONSerialize(internalTestPlan))
}

// Few test build deps expression qualify. Octopus hwDef removed.
func TestUpdateTestCases_buildDeps_in_use_flags_dict_partial_match(t *testing.T) {
	buildDepsTest1 := []*api.TestCase_BuildDeps{
		{
			Value: "nissa,!invalid-use-flag",
		},
		{
			Value: "invalid-use-flag",
		},
	}
	buildDepsTest2 := []*api.TestCase_BuildDeps{
		{
			Value: "dedede",
		},
	}
	internalTestPlan := GenerateInternalTestPlan(buildDepsTest1, buildDepsTest2)
	filteredInternalTestPlan := proto.Clone(internalTestPlan).(*api.InternalTestplan)
	emptyLogger := log.New(io.Discard, "", 0)
	err := updateTestCases(filteredInternalTestPlan, GenerateBoardUseFlagDict([]string{"octopus", "nissa", "dedede"}), emptyLogger)
	if err != nil {
		t.Errorf("updateTestCases failed: %v", err)
		return
	}
	for _, testCase := range internalTestPlan.GetTestCases() {
		if testCase.GetName() == "test1" {
			for _, hwReqs := range testCase.GetHwRequirements() {
				var newHwDef []*api.SwarmingDefinition
				for _, hwDef := range hwReqs.GetHwDefinition() {
					if hwDef.GetDutInfo().GetChromeos() == nil || hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget() != "octopus" {
						newHwDef = append(newHwDef, hwDef)
					}
				}
				hwReqs.HwDefinition = newHwDef
			}
		}
	}

	assert.Equal(t, JSONSerialize(filteredInternalTestPlan), JSONSerialize(internalTestPlan))
}

// No test hwDef expression qualifies. All ChromeOS Dut type hwDef removed.
func TestUpdateTestCases_buildDeps_not_in_use_flags_dict(t *testing.T) {
	buildDepsTest1 := []*api.TestCase_BuildDeps{
		{
			Value: "common,notpresent",
		},
		{
			Value: "!common",
		},
	}
	internalTestPlan := GenerateInternalTestPlan(buildDepsTest1, buildDepsTest1)
	filteredInternalTestPlan := proto.Clone(internalTestPlan).(*api.InternalTestplan)
	emptyLogger := log.New(io.Discard, "", 0)
	err := updateTestCases(filteredInternalTestPlan, GenerateBoardUseFlagDict([]string{"nissa", "octopus", "dedede"}), emptyLogger)
	if err != nil {
		t.Errorf("updateTestCases failed: %v", err)
		return
	}
	for _, testCase := range internalTestPlan.GetTestCases() {
		for _, hwReqs := range testCase.GetHwRequirements() {
			var newHwDef []*api.SwarmingDefinition
			for _, hwDef := range hwReqs.GetHwDefinition() {
				if hwDef.GetDutInfo().GetChromeos() == nil {
					newHwDef = append(newHwDef, hwDef)
				}
			}
			hwReqs.HwDefinition = newHwDef
		}
	}
	assert.Equal(t, JSONSerialize(filteredInternalTestPlan), JSONSerialize(internalTestPlan))
}

// Partial test hwDef expression qualifies. No ChromeOS Dut type hwDef removed.
func TestUpdateTestCases_buildDeps_expression_qualifies(t *testing.T) {
	buildDepsTest1 := []*api.TestCase_BuildDeps{
		{
			Value: "common,notpresent",
		},
		{
			Value: "common",
		},
	}
	internalTestPlan := GenerateInternalTestPlan(buildDepsTest1, buildDepsTest1)
	filteredInternalTestPlan := proto.Clone(internalTestPlan).(*api.InternalTestplan)
	emptyLogger := log.New(io.Discard, "", 0)
	err := updateTestCases(filteredInternalTestPlan, GenerateBoardUseFlagDict([]string{"nissa", "octopus", "dedede"}), emptyLogger)
	if err != nil {
		t.Errorf("updateTestCases failed: %v", err)
		return
	}
	assert.Equal(t, JSONSerialize(filteredInternalTestPlan), JSONSerialize(internalTestPlan))
}

// No use flag dict. All hwDef kept.
func TestUpdateTestCases_use_flag_dict_missing_board(t *testing.T) {
	buildDepsTest1 := []*api.TestCase_BuildDeps{
		{
			Value: "uncommon",
		},
	}
	internalTestPlan := GenerateInternalTestPlan(buildDepsTest1, buildDepsTest1)
	filteredInternalTestPlan := proto.Clone(internalTestPlan).(*api.InternalTestplan)
	emptyLogger := log.New(io.Discard, "", 0)
	err := updateTestCases(filteredInternalTestPlan, GenerateBoardUseFlagDict([]string{""}), emptyLogger)
	if err != nil {
		t.Errorf("updateTestCases failed: %v", err)
		return
	}

	assert.Equal(t, JSONSerialize(filteredInternalTestPlan), JSONSerialize(internalTestPlan))
}
