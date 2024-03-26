// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	server "chromiumos/test/ctpv2/common/server_template"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"chromiumos/test/publish/cmd/publishserver/storage"

	"github.com/google/uuid"
	"go.chromium.org/chromiumos/config/go/test/api"
)

const (
	binName         = "use_flag_filter"
	fileName        = "tast_use_flags.txt"
	useFlagsDirBase = "/tmp/use_flag_filter/tast_use_flags_"
)

func mapToString(m map[string]map[string]bool) string {
	result := ""
	for k, v := range m {
		result += fmt.Sprintf("%s: {\n", k)
		for k2, v2 := range v {
			result += fmt.Sprintf("\t%s: %t,\n", k2, v2)
		}
		result += "},\n"
	}
	return result
}

// DownloadUseFlags downloads all use flags from a given gsUrl.
func DownloadUseFlags(ctx context.Context, gsURL string, gsClient *storage.GSClient, buildTarget string, log *log.Logger, useFlagsDir string) (map[string]bool, error) {

	destFilePath := filepath.Join(useFlagsDir, buildTarget)

	if err := gsClient.DownloadFile(ctx, gsURL, destFilePath); err != nil {
		log.Printf("File not present at gsUrl : %s", gsURL)
		return nil, nil
	}
	log.Printf("Downloaded file :%s", buildTarget)
	content, err := os.ReadFile(destFilePath)
	if err != nil {
		return nil, err
	}

	useFlagList := strings.Split(string(content), "\n")

	useFlagSet := make(map[string]bool)
	for _, item := range useFlagList {
		useFlagSet[item] = true
	}

	return useFlagSet, nil
}

func createDirectory(log *log.Logger) (string, error) {

	useFlagsDir := useFlagsDirBase + uuid.New().String()
	log.Printf("Creating dir for storing use flags ...")
	if err := os.MkdirAll(useFlagsDir, 0755); err != nil {
		return "", err
	}
	log.Printf("UseFlagsDir set to :%s", useFlagsDir)
	return useFlagsDir, nil
}

// GenerateUseFlagDict creates a dictionary for buildTarget and respective use flag list.
func generateUseFlagDict(ctx context.Context, req *api.InternalTestplan, log *log.Logger) (map[string]map[string]bool, error) {

	buildTargetGSMap, err := createBuildTargetGSMap(req)
	if err != nil {
		return nil, fmt.Errorf("Error generating board-gcsPath map: %s", err)
	}

	gsClient, err := storage.NewGSClient(ctx, "")
	if err != nil {
		return nil, err
	}

	useFlagsDir, err := createDirectory(log)
	if err != nil {
		return nil, fmt.Errorf("Error setting up parent directory for downloading files from gs: %s", err)
	}

	// useFlagDict will look like below, after collecting use flags for all buildTarget in suite info.
	// map[
	// 	"nissa-variant": map[
	// 		"amd64" : true,
	// 		"vulkan" : true,
	// 		"wpa3_sae" : true
	// 	],
	// 	"octopus-variant": map[
	// 		"arc" : true,
	// 		"arc-camera3" : true,
	// 		"arcvm" : true,
	// 		"cheets_user_64" : true
	// 	]
	// ]
	useFlagDict := make(map[string]map[string]bool)

	for buildTarget, gcsPath := range buildTargetGSMap {
		gcsPath = gcsPath + "/" + fileName
		useFlagSet, err := DownloadUseFlags(ctx, gcsPath, gsClient, buildTarget, log, useFlagsDir)
		if err != nil {
			return nil, err
		}
		// Should not add to dict if use flags was not found
		if useFlagSet == nil {
			continue
		}

		useFlagDict[buildTarget] = useFlagSet
	}

	defer gsClient.Close()
	log.Printf("useFlagDict: %s", mapToString(useFlagDict))

	return useFlagDict, nil
}

// CreateBuildTargetGSMap creates a map with board+variant value as key and gspath for image location as value
func createBuildTargetGSMap(req *api.InternalTestplan) (map[string]string, error) {

	buildTargetGSMap := make(map[string]string)
	targetRequirements := req.GetSuiteInfo().GetSuiteMetadata().GetTargetRequirements()
	for _, tr := range targetRequirements {
		buildTargetGSKey, err := retrieveBuildTargetKey(tr.GetHwRequirements().GetHwDefinition())
		if err != nil {
			return nil, fmt.Errorf("Error retrieving build target key from hwDef: %s", err)
		}
		buildTargetGSMap[buildTargetGSKey] = tr.GetSwRequirement().GetGcsPath()
	}
	log.Printf("BoardGsURLMap: %s", buildTargetGSMap)
	return buildTargetGSMap, nil
}

// RetrieveBuildTargetKey retrieves ChromeOS buildTarget+variant from the hwDef
func retrieveBuildTargetKey(req []*api.SwarmingDefinition) (string, error) {
	for _, hwDef := range req {
		if hwDef.GetDutInfo().GetChromeos() != nil {
			board := hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget()
			retrieveBuildTargetKey := board
			if hwDef.GetVariant() != "" {
				retrieveBuildTargetKey += "-" + hwDef.GetVariant()
			}
			return retrieveBuildTargetKey, nil
		}
	}
	return "", nil
}
func checkIfTestHwDepQualify(includeFlags []string, excludeFlags []string, useFlagSet map[string]bool) bool {
	// Check if all items in includeFlags are present in useFlagSet
	for _, flag := range includeFlags {
		if !useFlagSet[flag] {
			return false
		}
	}
	// Check if any items in excludeFlags are present in useFlagSet
	for _, flag := range excludeFlags {
		if useFlagSet[flag] {
			return false
		}
	}
	return true
}

func checkIfAnyTestHwDepsQualify(useFlagSet map[string]bool, testHwDeps []*api.TestCase_BuildDeps, log *log.Logger) bool {
	for _, buildDep := range testHwDeps {
		includeFlags, excludeFlags := createIncludeAndExcludeUseFlagList(buildDep, log)
		if checkIfTestHwDepQualify(includeFlags, excludeFlags, useFlagSet) {
			return true
		}
	}
	return false
}

func createIncludeAndExcludeUseFlagList(buildDep *api.TestCase_BuildDeps, log *log.Logger) ([]string, []string) {
	var includeFlag []string
	var excludeFlag []string

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error parsing use flag expression from test metadata %s", buildDep.GetValue())
			includeFlag = nil
			excludeFlag = nil
		}
	}()

	condition := strings.Split(buildDep.GetValue(), ",")
	for _, item := range condition {
		if len(item) > 0 && item[0] == '!' {
			excludeFlag = append(excludeFlag, item[1:])
		} else {
			includeFlag = append(includeFlag, item)
		}
	}
	return includeFlag, excludeFlag
}

func filterHwRequirementsBasedOnUseFlag(hwReqs []*api.HWRequirements, useFlagDict map[string]map[string]bool, testHwDeps []*api.TestCase_BuildDeps, log *log.Logger) error {

	for _, hwReq := range hwReqs {
		// hwDef will be removed if no buildDeps in test metadata is present in hwDef's use flag set in useFlagDict
		var filteredHwDefs []*api.SwarmingDefinition
		hwDefs := hwReq.GetHwDefinition()

		for _, hwDef := range hwDefs {

			// fetch use flag set for hwDef. Key is BuildTarget+variant.
			buildTargetKey := hwDef.GetDutInfo().GetChromeos().GetDutModel().GetBuildTarget()
			if hwDef.GetVariant() != "" {
				buildTargetKey += "-" + hwDef.GetVariant()
			}

			if hwDef.GetDutInfo().GetChromeos() != nil && useFlagDict[buildTargetKey] != nil {
				// check if any test buildDeps is present in the set.
				useFlagSet := useFlagDict[buildTargetKey]
				qualify := checkIfAnyTestHwDepsQualify(useFlagSet, testHwDeps, log)
				if qualify {
					filteredHwDefs = append(filteredHwDefs, hwDef)
				} else {
					log.Printf("DUT with build target : %s removed", buildTargetKey)
				}
			} else {
				log.Printf("DUT skipped check, due to missing use flag list or DUT is not of ChromeOS type")
				filteredHwDefs = append(filteredHwDefs, hwDef)
			}
		}
		// update hwDef list
		hwReq.HwDefinition = filteredHwDefs
	}
	return nil
}

func updateTestCases(req *api.InternalTestplan, useFlagDict map[string]map[string]bool, log *log.Logger) error {

	for _, testCase := range req.GetTestCases() {
		if len(testCase.GetMetadata().GetTestCase().GetBuildDependencies()) > 0 {
			log.Printf("Pruning HwDef for test : %s", testCase.GetName())
			err := filterHwRequirementsBasedOnUseFlag(testCase.GetHwRequirements(), useFlagDict, testCase.GetMetadata().GetTestCase().GetBuildDependencies(), log)
			if err != nil {
				return fmt.Errorf("Error filtering HwReqs: %s", err)
			}
		} else {
			log.Printf("Skippping as test : %s, doesn't  have any build deps", testCase.GetName())
		}
	}
	return nil
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	ctx := context.Background()

	useFlagDict, err := generateUseFlagDict(ctx, req, log)
	if err != nil {
		return nil, fmt.Errorf("Error generating board-use flags map: %s", err)
	}

	err = updateTestCases(req, useFlagDict, log)
	if err != nil {
		return nil, fmt.Errorf("Error during use flag based test cases pruning: %s", err)
	}

	return req, nil
}

func main() {
	err := server.Server(executor, binName)
	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}
