// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/util/common"
	"log"
)

// GetGfxInfo packages graphics/hardware_probe labels in the GfxLabels member of GetGfxInfoResponse
func GetGfxInfo(logger *log.Logger, dutClient api.DutServiceClient) (*api.GetGfxInfoResponse, error) {
	reportingLabelsJson, err := common.RunCmd(context.Background(), "/usr/local/graphics/hardware_probe", []string{"--labels-reporting"}, dutClient)

	if err != nil {
		logger.Printf("gfx hardware_probe cmd FAILED: %s\n", err)
		reportingLabelsJson = ""
	}

	labelsMap, err := labelsInputToStringMap(reportingLabelsJson)
	if err != nil {
		logger.Printf("json conversion FAILED: %s\n", err)
	}

	resp := &api.GetGfxInfoResponse{GfxLabels: labelsMap}
	return resp, nil
}

// labelsInputToStringMap converts a json object string into map[string]string
func labelsInputToStringMap(labelsJson string) (map[string]string, error) {
	labelsBytes := []byte(labelsJson)

	var labelsObj interface{}
	var err = json.Unmarshal(labelsBytes, &labelsObj)
	if err != nil {
		return nil, err
	}

	labelsMap, ok := labelsObj.(map[string]interface{})
	if !ok {
		err := errors.New("labels json is expected to be an object")
		return nil, err
	}

	var labelsMapStr map[string]string = make(map[string]string)
	for k, v := range labelsMap {
		labelsMapStr[k] = fmt.Sprintf("%v", v)
	}

	return labelsMapStr, nil
}
