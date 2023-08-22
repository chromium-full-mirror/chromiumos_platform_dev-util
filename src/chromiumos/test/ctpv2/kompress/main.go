// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/mitchellh/hashstructure"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/chromiumos/config/go/test/api"
)

// TrRequest represents the what will become a TR request:
// Where it will be 1 request that contains 1 --> many tests.
// This request will be built up to contain all needed information to make a Tr(v2) request.
type TrRequest struct {
	req *api.HWRequirements
	tcs []*api.CTPTestCase
}

// loading is used in lab avalability such that devices with the same HW;
// but different SW requirements, still share the same pool of physical devices.
type loading struct {
	value int
}

type hwInfo struct {
	req               *api.HWRequirements
	labLoading        *loading
	numInCurrentShard int
}

// Kv structs are useful for gobased sorting.
type kv struct {
	key   uint64
	value int
}

type distroCfg struct {
	isUnitTest      bool
	unitTestDevices int
	maxInShard      int
}

type middleOutData struct {
	// Originally defined HW to TC
	hwToTCMap map[uint64][]string

	// Map between original HW objects, and the flat HW equivs
	// Example: [sha123]: [sha1, sha2, sha3]
	hwEquivalenceMap map[uint64][]uint64

	// the Sha of the HWRequirements object from a TC, pointing back to the HWRequirements object
	hwUUIDMap map[uint64]*api.HWRequirements

	cfg distroCfg

	// Flat HWUUID to flat HW
	flatHWUUIDMap map[uint64]*hwInfo

	// The TestCase name pointing to its respective object
	tcUUIDMap map[string]*api.CTPTestCase

	finalAssignments map[uint64][][]string
}

// newMiddleOutData returns a struct of the middleOutData with the data init'd but empty.
func newMiddleOutData() *middleOutData {
	mo := &middleOutData{
		hwToTCMap:        make(map[uint64][]string),
		hwEquivalenceMap: make(map[uint64][]uint64),
		hwUUIDMap:        make(map[uint64]*api.HWRequirements),
		cfg:              distroCfg{},
		flatHWUUIDMap:    make(map[uint64]*hwInfo),
		tcUUIDMap:        make(map[string]*api.CTPTestCase),

		finalAssignments: make(map[uint64][][]string),
	}
	return mo

}

// middleOut creates TRRequest(S) from a ctpv2 internal test plan.
func middleOut(resp *api.InternalTestplan, cfg distroCfg) ([]*TrRequest, error) {
	solverData := newMiddleOutData()
	solverData.cfg = cfg
	for _, tc := range resp.GetTestCases() {
		tcUUID := tc.GetName()
		// Drop all of the HW fluff in the TC for memory sakes.
		tcForMap := &api.CTPTestCase{
			Name:     tc.GetName(),
			Metadata: tc.GetMetadata(),
		}
		solverData.tcUUIDMap[tcUUID] = tcForMap
		for _, hw := range tc.HwRequirements {
			// Note: Each `hw` is still a repeated list of HW *options* for the test.
			hash := addHWtohwUUIDMap(solverData.hwUUIDMap, hw)
			for k, v := range flattenList([]*api.HWRequirements{hw}) {
				err := addHWtoFlatHWUUIDMap(solverData.flatHWUUIDMap, k, v)
				if err != nil {
					return nil, err
				}
			}
			solverData.hwToTCMap[hash] = append(solverData.hwToTCMap[hash], tcUUID)
		}
	}

	// Goal here is to make the hwEquivalenceMap; such that when we go to distribute the hwToTCMap
	// We can lookup what HW options are available.
	for key, hwOptions := range solverData.hwUUIDMap {
		solverData.hwEquivalenceMap[key] = findMatches(hwOptions, solverData.flatHWUUIDMap)
	}

	return createTrRequests(greedyDistro(solverData), solverData)
}

// createTrRequests translates a final {hw:[[shard], [shard]]} map into a flat list of TrRequests.
func createTrRequests(distro map[uint64][][]string, solverData *middleOutData) ([]*TrRequest, error) {
	TrRequests := []*TrRequest{}
	for k, shards := range distro {
		for _, tcs := range shards {
			shardedtcs := []*api.CTPTestCase{}
			for _, tc := range tcs {
				lltc, ok := solverData.tcUUIDMap[tc]
				if !ok {
					return nil, fmt.Errorf("tc assigned but was not given, something critically wrong happened to end up here")
				}
				shardedtcs = append(shardedtcs, lltc)
			}
			trReq := &TrRequest{
				req: solverData.flatHWUUIDMap[k].req,
				tcs: shardedtcs,
			}
			TrRequests = append(TrRequests, trReq)
		}
	}
	return nil, nil
}

// Given the class map, and the tc map, gather the lab loading for the devices, and make our best guess at assigning tests to HW.
func greedyDistro(solverData *middleOutData) map[uint64][][]string {
	populateLabAvalability(solverData)
	orderedHw := hwSearchOrdering(solverData.hwEquivalenceMap)

	// Starting with the HW classes with the least amount of  equivalent classes
	for _, hwS := range orderedHw {
		// Find the TCS that require this class, and assign them devices.
		hwHash := hwS.key
		tcs := solverData.hwToTCMap[hwHash]
		// Currently we will not try anymore than basic sharding.
		// As in, we won't attempt to "fill" a pod, then spill over.
		// Its either "you can take all these tests" or we get a new pod.
		shards := shard(tcs, solverData.cfg.maxInShard)
		for _, shardedtc := range shards {
			selectedDevice, expandCurrentShard := getDevices(solverData, len(shardedtc), hwHash)
			assignHardware(solverData, selectedDevice, expandCurrentShard, shardedtc)
		}
	}
	return solverData.finalAssignments
}

// assignHardware will add the tests to the selectedDevice, being aware if it should go into a non-filled hard, or a new one.
// assignHardware will also decrement the number of devices remaining every time device is assigned tests.
func assignHardware(solverData *middleOutData, selectedDevice uint64, expandCurrentShard bool, shardedtc []string) {
	if expandCurrentShard {
		lastElement := len(solverData.finalAssignments[selectedDevice])
		solverData.finalAssignments[selectedDevice][lastElement-1] = append(solverData.finalAssignments[selectedDevice][lastElement-1], shardedtc...)
		solverData.flatHWUUIDMap[selectedDevice].numInCurrentShard += len(shardedtc) // Show the status of the current shard. Might be wrong.
		if solverData.flatHWUUIDMap[selectedDevice].numInCurrentShard == solverData.cfg.maxInShard {
			solverData.flatHWUUIDMap[selectedDevice].numInCurrentShard = 0
		}
	} else {
		_, ok := solverData.finalAssignments[selectedDevice]
		if !ok {
			solverData.finalAssignments[selectedDevice] = [][]string{}
		}

		solverData.finalAssignments[selectedDevice] = append(solverData.finalAssignments[selectedDevice], shardedtc)
		solverData.flatHWUUIDMap[selectedDevice].labLoading.value-- // Reduce the # of open devices by 1.
		// If the shard is not full, mark it as such.
		if len(shardedtc) != solverData.cfg.maxInShard {
			solverData.flatHWUUIDMap[selectedDevice].numInCurrentShard += len(shardedtc)
		}
	}
}

// findMatches: find options for the given hwOption from the given flatHWUUIDMap
func findMatches(hwOption *api.HWRequirements, flatHWUUIDMap map[uint64]*hwInfo) []uint64 {
	matches := []uint64{}
	for sha, foundHW := range flatHWUUIDMap {
		if isParent(foundHW.req, hwOption) {
			matches = append(matches, sha)
		}
	}
	return matches
}

// isParent determines if the first is fully encompassing of the second object.
func isParent(parentSrc *api.HWRequirements, childSrc *api.HWRequirements) bool {
	/*
		In other words, if the parent has at least all of the fields of the child, it can run the child.
	*/
	for _, parent := range parentSrc.HwDefinition {
		found := false
		for _, child := range childSrc.HwDefinition {

			if !allItemsIn(child.GetSwarmingLabels(), parent.GetSwarmingLabels()) {
				return false
			}

			if reflect.DeepEqual(parent.DutInfo, child.DutInfo) && reflect.DeepEqual(parent.ProvisionInfo, child.ProvisionInfo) {
				found = true
			}
		}
		// If a parent is not found, then this can't be a match.
		if !found {
			return false
		}

	}
	return true
}

// Check all items from 1 are in 2
func allItemsIn(item1 []string, item2 []string) bool {
	for _, string1 := range item1 {
		found := false
		for _, string2 := range item2 {
			if string1 == string2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// shard will device the list into a list of lists where each item in the list length of maxInShard
// eg: [1,2,3,4], maxInShard=2 --> [[1,2], [3,4]]
func shard(tests []string, maxInShard int) (shards [][]string) {
	for maxInShard < len(tests) {
		tests, shards = tests[maxInShard:], append(shards, tests[0:maxInShard:maxInShard])
	}
	return append(shards, tests)
}

// TODO (azrahman@; integrate with swarming API)
func labAvalability(*api.HWRequirements) int {
	// returns the number of devices which can meet the requirement.
	// will need to include pool
	return -1
}

// Will add the amount of devices in the lab to each of the HW items.
func populateLabAvalability(solverData *middleOutData) {
	hwFound := make(map[uint64]*loading)
	for _, value := range solverData.flatHWUUIDMap {
		hash, _ := hashstructure.Hash(value.req.HwDefinition[0].DutInfo, hashstructure.FormatV2, nil)
		_, exists := hwFound[hash]
		if exists {
			value.labLoading = hwFound[hash]
			continue
		}

		if solverData.cfg.unitTestDevices != 0 {
			hwFound[hash] = &loading{value: solverData.cfg.unitTestDevices}
		} else {
			value.labLoading = &loading{value: labAvalability(value.req)}
		}
		value.labLoading = hwFound[hash]
	}
}

// translate the given hwEquivalenceMap into a 2d map:
// EG [id1=[1 || 2] || id2=[3 || 4]] needs to be [[1, 2, 3, 4]]
// hwEquivalenceMap is like id1 = [id1, id2]
// now needs to be like id1 = [newid1, newid2, newid3, newid4]
func flattenEqMap(hwEquivalenceMap map[uint64][]uint64, hwUUIDMap map[uint64]*api.HWRequirements) (map[uint64][]uint64, map[uint64]*hwInfo) {
	newHWUUIDMap := make(map[uint64]*hwInfo)
	newHwEquivalenceMap := make(map[uint64][]uint64)

	for hw, results := range hwEquivalenceMap {
		var allHw []*api.HWRequirements
		newHwEquivalenceMap[hw] = []uint64{}

		allHw = append(allHw, hwUUIDMap[hw])
		for _, hash := range results {
			allHw = append(allHw, hwUUIDMap[hash])
		}

		// flattened is now the uuid for [newid1: obj, etc]
		flattened := flattenList(allHw)

		for k, v := range flattened {
			newHWUUIDMap[k] = &hwInfo{req: v}
			newHwEquivalenceMap[hw] = append(newHwEquivalenceMap[hw], k)
		}
	}
	return newHwEquivalenceMap, newHWUUIDMap
}

// flattenList translates [hw.requirements=[1||2], hw.requirements=[3||4]] into [1,2,3,4]
func flattenList(allHw []*api.HWRequirements) map[uint64]*api.HWRequirements {
	flatHW := make(map[uint64]*api.HWRequirements)

	for _, hw := range allHw {

		for _, innerHW := range hw.HwDefinition {
			flattened := proto.Clone(hw).(*api.HWRequirements)
			flattened.HwDefinition = []*api.SwarmingDefinition{innerHW}

			h, _ := hashstructure.Hash(flattened, hashstructure.FormatV2, nil)
			flatHW[h] = flattened
		}

	}
	return flatHW
}

// getDevices finds a device from the devicepool + hwEquivalenceMap to satsify the need for the test
// It will first look for a matching device with a non-full shard that fits,
// otherwise it will look for a device with the most availability in the lab.
func getDevices(solverData *middleOutData, numTests int, hwHash uint64) (selectedDevice uint64, append bool) {
	// This is a pretty expensive approach to sharding:
	// We will always check all devices to see if they have room in a non-empty shard.
	// So even when we fully fill a device, or it hasn't been touched, we still check it.
	// First, try to fill a non-empty shard
	devices := solverData.hwEquivalenceMap[hwHash]
	for _, device := range devices {
		// if the shard is empty, we need to use the labloading process block
		// not the shard filler.
		if solverData.flatHWUUIDMap[device].numInCurrentShard < 1 {
			continue
		}

		if solverData.flatHWUUIDMap[device].numInCurrentShard+numTests <= solverData.cfg.maxInShard {
			selectedDevice = device
			return selectedDevice, true
		}
	}

	// If that cannot be done, then just pick the device with the most available.
	maxAvalibleFound := math.MinInt32
	for _, device := range devices {
		if solverData.flatHWUUIDMap[device].labLoading.value > maxAvalibleFound {
			maxAvalibleFound = solverData.flatHWUUIDMap[device].labLoading.value
			selectedDevice = device
		}
	}

	return selectedDevice, false
}

// hwSearchOrdering: given the flatUUIDLoadingMap, return the order of least common to most common boards/eqs.
func hwSearchOrdering(flatEqMap map[uint64][]uint64) []kv {
	/*
		 For example: hwEqMap {hw1: [hw1,hw2,hw3], hw3: [hw3]}

		we want to allow hw3 to find matches first. This is to give a chance for all tests which specifically require hw3
		to take prioty on that hw.

		Example problem we are trying to solve (given the hwEq map above)
			- labLoading provides there are 2 devices for each type
			- There is a maximum of 2 tests per shard allowed
			- The HW to TC Loading:
				hw1: [tc1, tc2, ... 8]
				hw3: [tc1 .. 4]

			If we allowed hw1 to be solved first, it could potentially be assigned HW3
			(as it has the same avalbility as HW1/Hw2). Then when the Hw3 TC come to be assigned, all the HW3 devices have been
			loaded.

	*/

	var sortedStruct []kv
	for k, v := range flatEqMap {
		sortedStruct = append(sortedStruct, kv{key: k, value: len(v)})
	}

	sort.Slice(sortedStruct, func(i, j int) bool {
		return sortedStruct[i].value < sortedStruct[j].value
	})
	return sortedStruct
}

// addHWtohwUUIDMap is a helper method to inject into the map without overwriting the keys existing values.
func addHWtohwUUIDMap(hwUUIDMap map[uint64]*api.HWRequirements, hw *api.HWRequirements) uint64 {
	hash, _ := hashstructure.Hash(hw, hashstructure.FormatV2, nil)
	_, hwExists := hwUUIDMap[hash]
	// Add the HW to the lookup map if not seen before.
	if !hwExists {
		hwUUIDMap[hash] = hw
	}
	return hash
}

// addHWtoFlatHWUUIDMap is a helper method to inject into the map without overwriting the keys existing values.
func addHWtoFlatHWUUIDMap(flatHWUUIDMap map[uint64]*hwInfo, k uint64, v *api.HWRequirements) error {
	_, exists := flatHWUUIDMap[k]
	// Add the HW to the lookup map if not seen before.
	if !exists {
		flatHWUUIDMap[k] = &hwInfo{req: v}
	} else if !reflect.DeepEqual(flatHWUUIDMap[k].req, v) {
		flatHWUUIDMap[k] = &hwInfo{req: v}
		return fmt.Errorf("mismatch")
	}
	return nil
}
