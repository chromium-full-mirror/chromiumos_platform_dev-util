// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package compatibility

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	"google.golang.org/protobuf/testing/protocmp"
)

var programAttr = &testpb.DutAttribute{Id: &testpb.DutAttribute_Id{Value: "attr-program"}}

func createCoverageRule(companions []*testpb.DutTarget) *testpb.CoverageRule {
	coverageRule := &testpb.CoverageRule{
		TestSuites: []*testpb.TestSuite{
			{
				Spec: &testpb.TestSuite_TestCaseIds{
					TestCaseIds: &testpb.TestCaseIdList{
						TestCaseIds: []*testpb.TestCase_Id{
							{
								Value: "suite-single-dut",
							},
						},
					},
				},
			},
		},
		DutTargets: []*testpb.DutTarget{
			{
				Criteria: []*testpb.DutCriterion{
					{
						AttributeId: &testpb.DutAttribute_Id{
							Value: "attr-program",
						},
						Values: []string{"boardA", "boardB", "boardC"},
					},
					{
						AttributeId: &testpb.DutAttribute_Id{
							Value: "swarming-pool",
						},
						Values: []string{"DUT_POOL_QUOTA"},
					},
				},
			},
		},
	}
	coverageRule.DutTargets = append(coverageRule.GetDutTargets(), companions...)
	return coverageRule
}

func createCompanion(programs []string, isAndroid bool) *testpb.DutTarget {
	dutTarget := &testpb.DutTarget{
		Criteria: []*testpb.DutCriterion{
			{
				AttributeId: &testpb.DutAttribute_Id{
					Value: "attr-program",
				},
				Values: programs,
			},
		},
	}
	if isAndroid {
		dutTarget.ProvisionConfig = getAndroidProvisionConfig()
	}
	return dutTarget
}

func getAndroidProvisionConfig() *testpb.ProvisionConfig {
	return &testpb.ProvisionConfig{
		Companion: &testpb.CompanionConfig{
			Config: &testpb.CompanionConfig_Android_{
				Android: &testpb.CompanionConfig_Android{
					GmsCorePackage: "latest-stable",
				},
			},
		},
	}
}

func TestCompanion_chooseCompanions_singleDutTarget(t *testing.T) {
	rules := createCoverageRule([]*testpb.DutTarget{})
	companions, err := chooseCompanions(0, rules, programAttr)
	if err != nil {
		t.Fatal(err)
	}
	var expected []*testplans.TestCompanion = nil
	if diff := cmp.Diff(expected, companions, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected companionInfos returned (-want +got): %s", diff)
	}
}

func TestCompanion_chooseCompanions_multiDutTarget_equalProgramSize(t *testing.T) {
	rules := createCoverageRule([]*testpb.DutTarget{
		createCompanion([]string{"companionBoardA", "companionBoardB", "companionBoardC"}, false),
		createCompanion([]string{"pixelA", "pixelB", "pixelC"}, true),
	})
	companions, err := chooseCompanions(1, rules, programAttr)
	if err != nil {
		t.Fatal(err)
	}

	expected := []*testplans.TestCompanion{
		{
			Board: "companionBoardB",
		},
		{
			Board:  "pixelB",
			Config: getAndroidProvisionConfig().GetCompanion(),
		},
	}
	if diff := cmp.Diff(expected, companions, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected companionInfos returned (-want +got): %s", diff)
	}
}

func TestCompanion_chooseCompanions_multiDutTarget_nonEqualProgramSize(t *testing.T) {
	rules := createCoverageRule([]*testpb.DutTarget{
		createCompanion([]string{"companionBoardA", "companionBoardB"}, false),
		createCompanion([]string{"pixelA"}, true),
	})
	companions, err := chooseCompanions(2, rules, programAttr)
	if err != nil {
		t.Fatal(err)
	}

	expected := []*testplans.TestCompanion{
		{
			Board: "companionBoardA",
		},
		{
			Board:  "pixelA",
			Config: getAndroidProvisionConfig().GetCompanion(),
		},
	}
	if diff := cmp.Diff(expected, companions, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected companionInfos returned (-want +got): %s", diff)
	}
}

func TestCompanion_chooseCompanions_multiDutTarget_noPrograms(t *testing.T) {
	rules := createCoverageRule([]*testpb.DutTarget{
		{}, // empty companion with no attributes
	})
	companions, err := chooseCompanions(2, rules, programAttr)
	if err == nil {
		t.Fatalf("expected error for invalid input, and compaions returned: %v", companions)
	}

	expectedErr := "DutCriteria must contain at least one \"attr-program\" attribute"
	if diff := cmp.Diff(expectedErr, err.Error(), protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected error returned (-want +got): %s", diff)
	}
}
