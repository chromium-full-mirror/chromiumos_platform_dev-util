// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package policies

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"chromiumos/test/pre_process/cmd/pre-process/interfaces"
	"chromiumos/test/pre_process/cmd/pre-process/structs"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/api/iterator"
)

// PassRatePolicy is a helper struct for GetFile.
type PassRatePolicy struct {
	ctx                    context.Context
	req                    *api.FilterFlakyRequest_PassRatePolicy
	variant                string
	milestone              string
	forceMapEnable         map[string]struct{}
	forceMapDisable        map[string]struct{}
	requiredPassRate       float64
	requiredRuns           int
	numOfMilestones        int
	missingTcList          map[string]struct{}
	data                   map[string]structs.SignalFormat
	otherData              map[string]structs.SignalFormat
	requiredRecentPassRate float64
	requiredRecentRuns     int
}

type resSchema struct {
	Build                string
	Normalized_test      string
	Board                string
	Success_permille     float64
	Total_runs           int
	Fail_runs            int
	Rec_Success_Permille float64
	Rec_Total_Runs       int
}

// Helper method to turn a list into a set. Because Go doesn't have built in sets. nice.
func listToMap(forced []string) map[string]struct{} {
	listMap := make(map[string]struct{})
	for _, test := range forced {
		listMap[test] = struct{}{}
	}
	return listMap
}

// StabilityFromPolicy returns the stability information computed from the policy given. Uses BQ results directly for test history.
func StabilityFromPolicy(req *api.FilterFlakyRequest_PassRatePolicy, variant string, milestone string, tcList map[string]struct{}) (map[string]structs.SignalFormat, error) {
	policy := PassRatePolicy{
		req:                    req,
		variant:                variant,
		milestone:              milestone,
		forceMapEnable:         make(map[string]struct{}),
		forceMapDisable:        make(map[string]struct{}),
		requiredPassRate:       float64(req.PassRatePolicy.PassRate),
		numOfMilestones:        int(req.PassRatePolicy.NumOfMilestones),
		requiredRuns:           int(req.PassRatePolicy.MinRuns),
		requiredRecentPassRate: float64(req.PassRatePolicy.PassRateRecent),
		requiredRecentRuns:     int(req.PassRatePolicy.MinRunsRecent),
		missingTcList:          tcList,
		data:                   make(map[string]structs.SignalFormat),
		otherData:              make(map[string]structs.SignalFormat),
	}

	err := policy.stabilityFromPolicy()
	if err != nil {
		return nil, err
	}
	return policy.data, nil
}

func (q *PassRatePolicy) determineSignalFromQuery(testname string, passRate float64, recPassRate float64, recRuns int, minRuns int) bool {
	// If the test is force_enabled, do this.
	if _, found := q.forceMapEnable[testname]; found {
		log.Printf("testname %s marked as forced disabled.", testname)
		return true
	} else if _, found := q.forceMapDisable[testname]; found {
		return false
	} else if passRate >= float64(q.requiredPassRate) {
		return true
	} else if q.requiredPassRate != 0 && q.requiredRecentRuns != 0 {
		if recPassRate >= q.requiredPassRate && recRuns >= q.requiredRecentRuns {
			return true // Was bad, but recent signal is very strongly good.
		}
	}
	log.Printf("testname %s < required PassRate %v with %v: MARKED UNSTABLE (might still run due to overrides).\n", testname, q.requiredPassRate, passRate)
	return false
}

func (q *PassRatePolicy) stabilityFromPolicy() error {
	mileStone, _ := strconv.Atoi(q.milestone)
	mileStroneRegex := mileStoneRegex(q.numOfMilestones, mileStone)
	q.forceMapEnable = listToMap(q.req.PassRatePolicy.ForceEnabledTests)
	q.forceMapDisable = listToMap(q.req.PassRatePolicy.ForceDisabledTests)

	// Query using all possible milestones. We will only search for results in the current on the first iterations.
	bqIter, err := interfaces.QueryForResults(q.variant, mileStroneRegex)
	if err != nil {
		return fmt.Errorf("unable to determine stabily: %s", err)
	}

	// Iterate through the results.
	// TODO, consider combining results from different milestones.
	// Might be useful when # runs required is like "20"; but we have 10 from 2 different milestones.
	q.data = q.iterThroughData(bqIter, q.milestone)

	if len(q.missingTcList) > 0 && q.numOfMilestones > 0 {
		q.populateMissingTests()
	}

	return nil

}

func (q *PassRatePolicy) populateMissingTests() {
	for k := range q.missingTcList {
		_, ok := q.otherData[k]
		if ok {
			log.Printf("Populating test %s from previous milestone.\n", k)
			q.data[k] = q.otherData[k]
		}
	}
}

func (q *PassRatePolicy) iterThroughData(bqIter *bigquery.RowIterator, milestone string) map[string]structs.SignalFormat {
	data := make(map[string]structs.SignalFormat)
	for {
		var resp resSchema
		err := bqIter.Next(&resp)
		if err == iterator.Done { // from "google.golang.org/api/iterator"
			break
		}
		if err != nil {
			// Intentionally do not rause this error, just log it for now.
			// Sometimes something silly can happen and we get `NULL` from the query for some items,
			// this is where its going to be raised. For now, lets log && continue to not break every test.
			log.Printf("INFORMATIONAL ERR - could not decode error on loop: %s\n", err)
		}

		signal := q.determineSignalFromQuery(resp.Normalized_test, resp.Success_permille, resp.Rec_Success_Permille, resp.Rec_Total_Runs, int(q.req.PassRatePolicy.MinRuns))

		// Not enough runs? skip!
		if resp.Total_runs < int(q.req.PassRatePolicy.MinRuns) {
			continue
		}

		fmted := structs.SignalFormat{
			Runs:           resp.Total_runs,
			Failruns:       resp.Fail_runs,
			Passrate:       resp.Success_permille,
			Signal:         signal,
			PassrateRecent: resp.Rec_Success_Permille,
			RunsRecent:     resp.Rec_Total_Runs,
		}
		// Not in milestone save in memory for later use!
		if resp.Build != milestone {
			q.otherData[resp.Normalized_test] = fmted
			continue
		}
		data[resp.Normalized_test] = fmted
		// Found and is current milestone, remove from the missing list.
		delete(q.missingTcList, resp.Normalized_test)
	}
	return data
}
