// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"
	"testing"

	"go.chromium.org/chromiumos/config/go/test/api"
	dut_api "go.chromium.org/chromiumos/config/go/test/lab/api"
)

func buildTestProto(boardName string, modelName string) *api.SwarmingDefinition {
	dut := &dut_api.Dut{}

	Cros := &dut_api.Dut_ChromeOS{DutModel: &dut_api.DutModel{
		BuildTarget: boardName,
		ModelName:   modelName,
	}}
	dut.DutType = &dut_api.Dut_Chromeos{Chromeos: Cros}

	return &api.SwarmingDefinition{DutInfo: dut}
}

func buildTestProtoDeps(boardName string, modelName string, deps []string) *api.SwarmingDefinition {
	dut := &dut_api.Dut{}

	Cros := &dut_api.Dut_ChromeOS{DutModel: &dut_api.DutModel{
		BuildTarget: boardName,
		ModelName:   modelName,
	}}
	dut.DutType = &dut_api.Dut_Chromeos{Chromeos: Cros}

	return &api.SwarmingDefinition{DutInfo: dut, SwarmingLabels: deps}
}

func buildTestProtoWithProvisionInfo(boardName string, modelName string, provInfo string) *api.SwarmingDefinition {
	dut := &dut_api.Dut{}

	Cros := &dut_api.Dut_ChromeOS{DutModel: &dut_api.DutModel{
		BuildTarget: boardName,
		ModelName:   modelName,
	}}
	dut.DutType = &dut_api.Dut_Chromeos{Chromeos: Cros}
	return &api.SwarmingDefinition{DutInfo: dut,
		ProvisionInfo: []*api.ProvisionInfo{
			&api.ProvisionInfo{
				// This is the image variant to use for the current target
				Identifier: provInfo,
			},
		},
	}
}
func buildTestProtoWithProvisionInfoAndDeps(boardName string, modelName string, provInfo string, deps []string) *api.SwarmingDefinition {
	dut := &dut_api.Dut{}

	Cros := &dut_api.Dut_ChromeOS{DutModel: &dut_api.DutModel{
		BuildTarget: boardName,
		ModelName:   modelName,
	}}
	dut.DutType = &dut_api.Dut_Chromeos{Chromeos: Cros}
	return &api.SwarmingDefinition{DutInfo: dut,

		SwarmingLabels: deps,
		ProvisionInfo: []*api.ProvisionInfo{
			&api.ProvisionInfo{
				// This is the image variant to use for the current target
				Identifier: provInfo,
			},
		},
	}
}

func TestAllItemsIn(t *testing.T) {
	if !allItemsIn([]string{"foo"}, []string{"foo"}) {
		t.Fatal("[foo] not found in [foo] when should be")
	}
	if !allItemsIn([]string{"foo"}, []string{"foo", "bar"}) {
		t.Fatal("[foo] not found in [foo, bar] when should be")
	}
	if !allItemsIn([]string{}, []string{}) {
		t.Fatal("[] not found in [] when should be")
	}
	if allItemsIn([]string{"foo", "bar"}, []string{"foo"}) {
		t.Fatal("[foo, bar] found in [foo] when should not be")
	}

}

func TestFindMatches(t *testing.T) {
	swarmingLabels := []string{"foo"}

	hwDef0a := buildTestProtoDeps("foo", "bar", swarmingLabels)
	hwDef0 := buildTestProto("foo", "bar")
	hwDef01 := buildTestProto("foo", "bar1")
	hwDef02 := buildTestProto("foo", "bar2")

	hw1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0},
	}

	hw1a := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0a},
	}

	hw1AndHw2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0, hwDef02},
	}

	hw1AndHw2AndHw3 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0, hwDef01, hwDef02},
	}

	// This test assumes flattenList works; which has its own unittest coverage.
	// Note: to ensure proper test coverage of the "findMatches" method, we must use flattenList
	// to generate the data rather than hand crafting.
	eqMap := flattenList([]*api.HWRequirements{hw1AndHw2AndHw3})
	flatHWUUIDMap := make(map[uint64]*hwInfo)

	for k, v := range eqMap {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	}

	if len(findMatches(hw1AndHw2, flatHWUUIDMap)) != 2 {
		t.Fatal("Map did not match both items from test class")
	}
	if len(findMatches(hw1a, flatHWUUIDMap)) != 0 {
		t.Fatal("Map matched items with dependencies when it should not have")
	}

	flatWithDeps := flattenList([]*api.HWRequirements{hw1a})
	flatHWUUIDMap = make(map[uint64]*hwInfo)

	for k, v := range flatWithDeps {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	}

	if len(findMatches(hw1, flatHWUUIDMap)) != 1 {
		t.Fatal("Did not find match when should have")
	}

}

func TestAssignHardware(t *testing.T) {
	selectedDevice := uint64(1)
	selectedDevice2 := uint64(2)

	expandCurrentShard := false
	flatUUIDLoadingMap := make(map[uint64]*hwInfo)

	hwDef0 := buildTestProto("foo", "bar")

	hwDef1 := buildTestProtoWithProvisionInfo("foo", "bar", "kn")

	reqs := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0},
	}

	reqs2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef1},
	}

	l := &loading{value: 2}
	flatUUIDLoadingMap[selectedDevice] = &hwInfo{
		req:               reqs,
		labLoading:        l,
		numInCurrentShard: 0,
	}
	flatUUIDLoadingMap[selectedDevice2] = &hwInfo{
		req:               reqs2,
		labLoading:        l,
		numInCurrentShard: 0,
	}

	shardedtc := []string{"1"}
	shardedtc2 := []string{"2"}
	shardedtc3 := []string{"3"}

	cfg := distroCfg{
		maxInShard: 2,
	}

	solverData := newMiddleOutData()
	solverData.cfg = cfg
	solverData.flatHWUUIDMap = flatUUIDLoadingMap

	assignHardware(solverData, selectedDevice, expandCurrentShard, shardedtc)
	if flatUUIDLoadingMap[selectedDevice].labLoading.value != 1 {
		t.Fatalf("Assigning a device did not reduce its lab loading")
	}
	if flatUUIDLoadingMap[selectedDevice].numInCurrentShard != 1 {
		t.Fatalf("Assigning an empty shard 1 test did not increase its num in shard count")
	}

	assignHardware(solverData, selectedDevice, true, shardedtc2)
	if flatUUIDLoadingMap[selectedDevice].labLoading.value != 1 {
		t.Fatalf("Assigning a device did not reduce its lab loading")
	}
	if flatUUIDLoadingMap[selectedDevice].numInCurrentShard != 0 {
		t.Fatalf("Filling a shard did not reset the count")
	}

	assignHardware(solverData, selectedDevice, false, shardedtc3)
	if flatUUIDLoadingMap[selectedDevice].labLoading.value != 0 {
		t.Fatalf("Assigning a device did not reduce its lab loading")
	}
	if flatUUIDLoadingMap[selectedDevice].numInCurrentShard != 1 {
		t.Fatalf("Filling a shard did not reset the count")
	}

	assignHardware(solverData, selectedDevice2, false, shardedtc)
	if flatUUIDLoadingMap[selectedDevice].labLoading.value != -1 {
		t.Fatalf("Same HW; different groupping should share the same lab resource but didn't")
	}
	if flatUUIDLoadingMap[selectedDevice].numInCurrentShard != 1 {
		t.Fatalf("new grouping should only show 1 in shard")
	}
	fmt.Println(flatUUIDLoadingMap[selectedDevice].numInCurrentShard)
	fmt.Println(flatUUIDLoadingMap[selectedDevice].labLoading.value)

	expected1 := [][]string{
		[]string{"1", "2"},
		[]string{"3"},
	}

	expected2 := [][]string{
		[]string{"1"},
	}

	if !reflect.DeepEqual(solverData.finalAssignments[selectedDevice], expected1) {
		t.Fatalf("incorrect hw assignments1")
	}

	if !reflect.DeepEqual(solverData.finalAssignments[selectedDevice2], expected2) {
		t.Fatalf("incorrect hw assignments2")
	}

}

func TestFindMatchesProv(t *testing.T) {
	// Test that a hwDef with provision info is not matched unless the provision info is also the same.
	swarmingLabels := []string{"foo"}

	hwDef0 := buildTestProto("foo", "bar")
	hwDef01 := buildTestProto("foo", "bar1")

	hwDef02 := buildTestProto("foo", "bar2")

	hwDef03 := buildTestProtoWithProvisionInfo("foo", "bar2", "prov")

	hwDef03Deps := buildTestProtoWithProvisionInfoAndDeps("foo", "bar2", "prov", swarmingLabels)

	hwDef0WDeps := buildTestProtoDeps("foo", "bar", swarmingLabels)
	testWithDeps := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0WDeps},
	}

	hw2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef03},
	}

	testWithProvAndDeps := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef03Deps},
	}
	testClass := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0, hwDef02},
	}

	classWithNoDeps := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0, hwDef01, hwDef02, hwDef03},
	}

	// This test assumes flattenList works; which has its own unittest coverage.
	// Note: to ensure proper test coverage of the "findMatches" method, we must use flattenList
	// to generate the data rather than hand crafting.
	flatMap := flattenList([]*api.HWRequirements{classWithNoDeps})
	flatHWUUIDMap := make(map[uint64]*hwInfo)

	for k, v := range flatMap {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	}

	if len(findMatches(testClass, flatHWUUIDMap)) != 2 {
		t.Fatal("Map did not match both items from test class")
	}
	if len(findMatches(testWithDeps, flatHWUUIDMap)) != 0 {
		t.Fatal("Map matched items with dependencies when it should not have")
	}

	// Test we get a match with provision and subdeps.
	flatMap = flattenList([]*api.HWRequirements{testWithProvAndDeps})
	flatHWUUIDMap = make(map[uint64]*hwInfo)

	for k, v := range flatMap {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	}
	if len(findMatches(hw2, flatHWUUIDMap)) != 1 {
		t.Fatal("Map matched items with dependencies when it should not have")
	}
}

func TestSharedDeviceLabLoadingDifferentProvision(t *testing.T) {
	SwarmingDef0 := buildTestProtoWithProvisionInfo("foo", "bar", "prov1")
	SwarmingDef1 := buildTestProtoWithProvisionInfo("foo", "bar", "prov2")
	SwarmingDef2 := buildTestProtoWithProvisionInfo("foo", "bar", "prov3")
	SwarmingDef3 := buildTestProtoWithProvisionInfo("foo", "bar", "prov4")

	hwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1},
	}
	hwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2},
	}
	hwDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef3},
	}

	hwEqMap := make(map[uint64][]uint64)
	hwUUIDMap := make(map[uint64]*api.HWRequirements)
	HwHash0 := uint64(0)
	HwHash1 := uint64(1)
	HwHash2 := uint64(2)

	hwEqMap[HwHash0] = []uint64{HwHash0}
	hwEqMap[HwHash1] = []uint64{HwHash1}
	hwEqMap[HwHash2] = []uint64{HwHash2}

	hwUUIDMap[HwHash0] = hwDef0
	hwUUIDMap[HwHash1] = hwDef1
	hwUUIDMap[HwHash2] = hwDef2
	hwEqMap[HwHash0] = []uint64{HwHash0}
	hwEqMap[HwHash0] = []uint64{HwHash1}
	hwEqMap[HwHash2] = []uint64{HwHash2}

	newEq, flatUUIDLoadingMap := flattenEqMap(hwEqMap, hwUUIDMap)

	solverData := newMiddleOutData()
	solverData.cfg = distroCfg{
		unitTestDevices: 2,
		maxInShard:      2}
	solverData.flatHWUUIDMap = flatUUIDLoadingMap
	solverData.hwEquivalenceMap = newEq
	populateLabAvalability(solverData)

	selectedDevice, expandCurrentShard := getDevices(solverData, 2, HwHash1)

	flatUUIDLoadingMap[selectedDevice].numInCurrentShard = 1
	if expandCurrentShard {
		t.Fatalf("First test should go into new shard and did not")
	}

	selectedDevice2, expandCurrentShard := getDevices(solverData, 1, HwHash1)

	if selectedDevice != selectedDevice2 {
		t.Fatalf("Shard was not filled when it should have been")
	}

	// Reset the shard, and reduce the number of devices for this by 1.
	flatUUIDLoadingMap[selectedDevice].numInCurrentShard = 0
	flatUUIDLoadingMap[selectedDevice].labLoading.value--

	for _, v := range flatUUIDLoadingMap {
		if v.labLoading.value != 1 {
			t.Fatalf("All devices share same hardware thus all should be reduced by 1")
		}
	}

	_, expandCurrentShard = getDevices(solverData, 1, HwHash1)

	if expandCurrentShard {

		t.Fatalf("Should not be same shard.")
	}

}

func TestGreedyDistro(t *testing.T) {
	SwarmingDef0 := buildTestProto("def1", "mod1")
	SwarmingDef1 := buildTestProto("def2", "mod2")
	SwarmingDef2 := buildTestProto("def3", "mod3")
	SwarmingDef3 := buildTestProto("def4", "mod4")

	hwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1},
	}
	hwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2},
	}
	hwDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef3},
	}

	flatHwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0},
	}
	flatHwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef1},
	}
	flatHwDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef2},
	}

	flatHwDef3 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef3},
	}

	hwEqMap := make(map[uint64][]uint64)
	hwUUIDMap := make(map[uint64]*api.HWRequirements)
	HwHash0 := uint64(0)
	HwHash1 := uint64(1)

	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1}
	hwEqMap[HwHash1] = []uint64{HwHash1}

	hwUUIDMap[HwHash0] = hwDef0
	hwUUIDMap[HwHash1] = hwDef1
	HwHash2 := uint64(2)
	hwUUIDMap[HwHash2] = hwDef2
	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1, HwHash2}
	hwEqMap[HwHash2] = []uint64{HwHash2}

	hwToTCMap := make(map[uint64][]string)

	hwToTCMap[HwHash0] = []string{"a", "b", "c"}
	hwToTCMap[HwHash1] = []string{"d", "e"}
	hwToTCMap[HwHash2] = []string{"f", "g", "h", "i", "j", "k"}

	cfg := distroCfg{isUnitTest: true, unitTestDevices: 3, maxInShard: 50}

	eqMap := flattenList([]*api.HWRequirements{hwDef0, hwDef1, hwDef2})
	flatHWUUIDMap := make(map[uint64]*hwInfo)

	for k, v := range eqMap {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	}

	flatEqMap := make(map[uint64][]uint64)

	for key, hwOptions := range hwUUIDMap {
		flatEqMap[key] = findMatches(hwOptions, flatHWUUIDMap)
	}

	solverData := newMiddleOutData()
	solverData.hwToTCMap = hwToTCMap
	solverData.hwEquivalenceMap = flatEqMap
	solverData.hwUUIDMap = hwUUIDMap
	solverData.cfg = cfg
	solverData.flatHWUUIDMap = flatHWUUIDMap

	finalAssignments := greedyDistro(solverData)

	expected := []*allowedAssignment{
		&allowedAssignment{
			tc: []string{"a", "b", "c", "d", "e"},
			hw: flatHwDef0,
		},
		&allowedAssignment{
			tc: []string{"a", "b", "c", "d", "e"},
			hw: flatHwDef1,
		},
		&allowedAssignment{
			tc: []string{"a", "b", "c", "d", "e"},
			hw: flatHwDef2,
		},
		&allowedAssignment{
			tc: []string{"a", "b", "c", "f", "g", "h", "i", "j", "k"},
			hw: flatHwDef3,
		},
	}

	expectedTcs := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	correct, err := validateDistro(finalAssignments, flatHWUUIDMap, cfg, expectedTcs, expected)
	if !correct {
		t.Fatal(err)
	}

	// Rerun the same test with different settings, ensure things still "work"
	cfg = distroCfg{
		isUnitTest:      true,
		maxInShard:      5,
		unitTestDevices: 5,
	}
	solverData.cfg = cfg

	solverData.finalAssignments = make(map[uint64][][]string)
	finalAssignments = greedyDistro(solverData)
	correct, err = validateDistro(finalAssignments, flatHWUUIDMap, cfg, expectedTcs, expected)
	if !correct {
		t.Fatal(err)
	}
}

func TestIsParent(t *testing.T) {
	parent := &api.HWRequirements{}
	child := &api.HWRequirements{}

	if !isParent(parent, child) {
		t.Fatalf("Empty parent/child should be true")
	}

	swarmingLabels := []string{"foo"}
	hwDef0 := buildTestProto("foo", "bar")
	hwDef001WDeps := buildTestProtoWithProvisionInfoAndDeps("foo", "bar", "prov1", swarmingLabels)
	hwDef002 := buildTestProtoWithProvisionInfo("foo", "bar", "prov2")

	hw1aprov1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef001WDeps},
	}
	hw1prov2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef002},
	}
	// Can 1 run 2?
	if isParent(hw1aprov1, hw1prov2) {
		t.Fatalf("Match found even with different provision info")
	}
	if isParent(hw1prov2, hw1aprov1) {
		t.Fatalf("Match found even with different provision info")
	}

	// In this case, the parent has more swarming labels defined than the child.
	hwDef0WDeps := buildTestProtoDeps("foo", "bar", swarmingLabels)

	hw1a := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0WDeps},
	}
	hw1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0},
	}
	// Should match
	if !isParent(hw1a, hw1) {
		t.Fatalf("Parent with extra deps should be able to run child")
	}
	// Should not match.
	if isParent(hw1, hw1a) {
		t.Fatalf("child with extra deps should not be able to use limited parent")
	}

	// Test different models will not match in either direction.
	hwDef02 := buildTestProto("foo", "bar2")
	hw2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef02},
	}
	if isParent(hw1a, hw2) {
		t.Fatalf("Parent with different model should not be possible")
	}
	if isParent(hw2, hw1a) {
		t.Fatalf("child with different model should not be possible.")
	}

	// Test that when given hw1 || hw2, it can be run on hw1a
	hw1OrHw2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{hwDef0, hwDef02},
	}
	if !isParent(hw1a, hw1OrHw2) {
		t.Fatalf("Case where every parent is in model should be true.")
	}

	// Test that when given hw1a it CANNOT be run on hw1 || hw2
	if isParent(hw1OrHw2, hw1a) {
		t.Fatalf("Parent with cases child doesn't have should not be true but is..")
	}

	// Identical content, but in different order should still match.
	parentLabels := append(swarmingLabels, "foo2")
	// Multiple parents singular child
	parent = &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{
			buildTestProtoDeps("foo", "bar2", parentLabels),
			buildTestProtoDeps("foo2", "bar2", parentLabels),
			buildTestProtoDeps("foo2", "bar1", parentLabels)},
	}
	child = &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{
			buildTestProtoDeps("foo2", "bar2", parentLabels),
			buildTestProtoDeps("foo", "bar2", parentLabels),
			buildTestProtoDeps("foo2", "bar1", parentLabels)},
	}
	if !isParent(parent, child) {
		t.Fatalf("mixed ordered should still result in true.")
	}

}

func TestFlattenList(t *testing.T) {
	SwarmingDef0 := buildTestProto("foo", "bar")
	SwarmingDef1 := buildTestProto("foo", "bar2")
	SwarmingDef2 := buildTestProto("foo2", "bar2")
	hwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1},
	}

	hwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef1, SwarmingDef2},
	}
	allHw := []*api.HWRequirements{hwDef0, hwDef1}

	expectedDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0},
	}

	expectedDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef1},
	}
	expectedDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef2},
	}

	flatList := flattenList(allHw)
	for _, given := range []*api.HWRequirements{expectedDef0, expectedDef1, expectedDef2} {
		found := false
		for _, hw := range flatList {
			if reflect.DeepEqual(hw, given) {
				found = true
			}
		}
		if !found {
			t.Fatalf("Swarming def: %s not found in list", given)
		}
	}
	if len(flatList) != 3 {
		t.Fatalf("len prob")
	}
}

func TestFlattenEqMap(t *testing.T) {
	SwarmingDef0 := buildTestProto("foo", "bar")
	SwarmingDef1 := buildTestProto("foo", "bar2")
	SwarmingDef2 := buildTestProto("foo2", "bar2")
	SwarmingDef3 := buildTestProto("foo3", "bar3")

	hwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1},
	}
	hwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2},
	}
	hwDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef3},
	}

	hwEqMap := make(map[uint64][]uint64)
	hwUUIDMap := make(map[uint64]*api.HWRequirements)
	HwHash0 := uint64(0)
	HwHash1 := uint64(1)

	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1}
	hwEqMap[HwHash1] = []uint64{HwHash1}

	hwUUIDMap[HwHash0] = hwDef0
	hwUUIDMap[HwHash1] = hwDef1

	newEq, newUUID := flattenEqMap(hwEqMap, hwUUIDMap)

	if len(newEq[HwHash0]) != 3 {
		t.Fatalf("Did not find the expected # of classes in the new Equiv map.")
	}
	fmt.Println(newEq)
	fmt.Println(newUUID)
	for _, k := range newEq[HwHash0] {
		value, ok := newUUID[k]

		if !ok {
			t.Fatalf("class missing from lookup map")
		}
		found := false

		for _, given := range []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2} {
			if reflect.DeepEqual(value.req.HwDefinition[0], given) {
				found = true
			}
		}
		if !found {
			t.Fatalf("HW definition missing from flattened eq map")
		}
	}

	// Test a third class properly flattens into the first class, but not the second
	HwHash2 := uint64(2)
	hwUUIDMap[HwHash2] = hwDef2
	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1, HwHash2}
	hwEqMap[HwHash2] = []uint64{HwHash2}
	newEq, newUUID = flattenEqMap(hwEqMap, hwUUIDMap)

	if len(newEq[HwHash0]) != 4 {
		t.Fatalf("Did not find the expected # of classes in the new Equiv map.")
	}

	if len(newEq[1]) != 3 {
		t.Fatalf("Did not find the expected # of classes in the new Equiv map.")
	}

	if len(newEq[HwHash2]) != 1 {
		t.Fatalf("Did not find the expected # of classes in the new Equiv map.")
	}

	for _, k := range newEq[HwHash0] {
		value, ok := newUUID[k]

		if !ok {
			t.Fatalf("class missing from lookup map")
		}
		found := false

		for _, given := range []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2, SwarmingDef3} {
			if reflect.DeepEqual(value.req.HwDefinition[0], given) {
				found = true
			}
		}
		if !found {
			t.Fatalf("HW definition missing from flattened eq map")
		}
	}

}

func TestGetDevices(t *testing.T) {
	SwarmingDef0 := buildTestProto("foo", "bar")
	SwarmingDef1 := buildTestProto("foo", "bar2")
	SwarmingDef2 := buildTestProto("foo2", "bar2")
	SwarmingDef3 := buildTestProto("foo3", "bar3")

	hwDef0 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1},
	}
	hwDef1 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef0, SwarmingDef1, SwarmingDef2},
	}
	hwDef2 := &api.HWRequirements{
		HwDefinition: []*api.SwarmingDefinition{SwarmingDef3},
	}

	hwEqMap := make(map[uint64][]uint64)
	hwUUIDMap := make(map[uint64]*api.HWRequirements)
	HwHash0 := uint64(0)
	HwHash1 := uint64(1)

	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1}
	hwEqMap[HwHash1] = []uint64{HwHash1}

	hwUUIDMap[HwHash0] = hwDef0
	hwUUIDMap[HwHash1] = hwDef1
	HwHash2 := uint64(2)
	hwUUIDMap[HwHash2] = hwDef2
	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1, HwHash2}
	hwEqMap[HwHash2] = []uint64{HwHash2}

	newEq, flatUUIDLoadingMap := flattenEqMap(hwEqMap, hwUUIDMap)

	solverData := newMiddleOutData()
	solverData.cfg = distroCfg{
		unitTestDevices: 2,
		maxInShard:      2}
	solverData.flatHWUUIDMap = flatUUIDLoadingMap
	solverData.hwEquivalenceMap = newEq
	populateLabAvalability(solverData)

	selectedDevice, expandCurrentShard := getDevices(solverData, 1, HwHash1)

	flatUUIDLoadingMap[selectedDevice].numInCurrentShard = 1
	if expandCurrentShard {
		t.Fatalf("First test should go into new shard and did not")
	}

	selectedDevice2, expandCurrentShard := getDevices(solverData, 1, HwHash1)

	if selectedDevice != selectedDevice2 {
		t.Fatalf("Shard was not filled when it should have been")
	}

	// Shard is full, so reset it and remove 1 from lab loading.
	flatUUIDLoadingMap[selectedDevice].numInCurrentShard = 0
	flatUUIDLoadingMap[selectedDevice].labLoading.value--

	selectedDevice3, expandCurrentShard := getDevices(solverData, 1, HwHash1)
	if selectedDevice3 == selectedDevice2 {
		t.Fatalf("New shard should be on different device for balancing")
	}
	if expandCurrentShard {
		t.Fatalf("First test in shard should not mark to expand current.")
	}

}

func validateDistro(finalAssignments map[uint64][][]string, flatUUIDLoadingMap map[uint64]*hwInfo, cfg distroCfg, expectedTcs []string, expected []*allowedAssignment) (bool, string) {
	hwCount := make(map[uint64]int)
	foundTcs := []string{}

	for hw, tc := range finalAssignments {

		flatTcs := []string{}
		for _, innerTcs := range tc {
			if _, found := hwCount[hw]; found {
				hwCount[hw]++
			} else {
				hwCount[hw] = 1
			}
			if len(innerTcs) > cfg.maxInShard {

				fmt.Println("shard size bad", innerTcs, cfg.maxInShard)
				return false, "Shard size exceeded"
			}
			flatTcs = append(flatTcs, innerTcs...)
		}
		foundTcs = append(foundTcs, flatTcs...)
		if !validateCorrectHwAssignment(flatUUIDLoadingMap[hw].req, flatTcs, expected) {
			return false, fmt.Sprintf("TC: %s Had wrong hw assignment", flatTcs)
		}
	}

	for hw, hwcount := range hwCount {
		if flatUUIDLoadingMap[hw].labLoading.value+hwcount != cfg.unitTestDevices {
			return false, fmt.Sprintf("Lab loading incorrect: %v", flatUUIDLoadingMap[hw].labLoading)
		}
	}
	if !listEqual(expectedTcs, foundTcs) {
		return false, fmt.Sprintf("Some Test(s) missing from assignment: got: %v expected: %v", foundTcs, expectedTcs)
	}
	return true, ""
}

func listEqual(expected []string, found []string) bool {

	anyMissing := false
	for _, tc := range expected {
		tcFound := false
		for _, foundTC := range found {
			if tc == foundTC {
				tcFound = true
			}
		}
		if !tcFound {
			fmt.Printf("TC: %s not found in given list: %s\n", tc, found)
			anyMissing = true
		}
	}
	return !anyMissing
}

func validateCorrectHwAssignment(givenHw *api.HWRequirements, flatTcs []string, expected []*allowedAssignment) bool {
	found := []string{}
	for _, e := range expected {
		// If we find the givenHW in the expected HW list,
		// Look for Tc matches
		if isMatch(givenHw, e.hw) {
			for _, tc := range flatTcs {
				for _, test := range e.tc {
					if tc == test {
						found = append(found, tc)
					}
				}
			}
		}
	}
	return reflect.DeepEqual(found, flatTcs)
}

func isMatch(a *api.HWRequirements, b *api.HWRequirements) bool {
	if getBuildTarget(a) != getBuildTarget(b) {
		return false
	}
	if getModelName(a) != getModelName(b) {
		return false
	}
	return true
}

type allowedAssignment struct {
	tc []string
	hw *api.HWRequirements
}

func TestSorting(t *testing.T) {
	hwEqMap := make(map[uint64][]uint64)
	HwHash0 := uint64(0)
	HwHash1 := uint64(1)
	HwHash2 := uint64(2)
	HwHash3 := uint64(3)

	hwEqMap[HwHash0] = []uint64{HwHash0, HwHash1, HwHash2}
	hwEqMap[HwHash1] = []uint64{HwHash1, HwHash2}
	hwEqMap[HwHash2] = []uint64{HwHash2}
	hwEqMap[HwHash3] = []uint64{HwHash0, HwHash1, HwHash2, HwHash3}

	f := hwSearchOrdering(hwEqMap)

	expected := []uint64{HwHash2, HwHash1, HwHash0, HwHash3}
	for i, k := range f {
		if expected[i] != k.key {
			t.Fatalf("HW did not order least to most common. Expected: %v, Got: %v", expected, f)
		}
	}
}

func getBuildTarget(target *api.HWRequirements) string {
	return target.HwDefinition[0].GetDutInfo().GetChromeos().DutModel.BuildTarget
}

func getModelName(target *api.HWRequirements) string {
	return target.HwDefinition[0].GetDutInfo().GetChromeos().DutModel.ModelName
}
