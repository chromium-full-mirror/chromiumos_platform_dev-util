// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
	"gopkg.in/yaml.v2"
)

type paramMap = map[string]string
type deviceParamMap = map[string]paramMap

type MoblyTestConfig struct {
	TestBeds []*TestBed `yaml:"TestBeds"`
}

type TestBed struct {
	Name        string       `yaml:"Name"`
	TestParams  *TestParams  `yaml:"TestParams"`
	Controllers *Controllers `yaml:"Controllers"`
}

type Controllers struct {
	OpenWrtDevices []*OpenWrtDevice `yaml:"OpenWrtDevice,omitempty"`
	AndroidDevices []*AndroidDevice `yaml:"AndroidDevice,omitempty"`
}

type AndroidDevice struct {
	Serial string   `yaml:"serial"`
	Role   string   `yaml:"role"`
	Params paramMap `yaml:",inline"`
}

type OpenWrtDevice struct {
	Hostname string   `yaml:"hostname"`
	Params   paramMap `yaml:",inline"`
}

type TestParams struct {
	Params paramMap `yaml:",inline"`
}

// Config Parameters from ExecutionMetadata
// Keyed by `config-params`
type ConfigParams struct {
	// Keyed by `test-params`
	TestParams paramMap
	// Values prefixed by `primary` or `secondary` will only apply to that device.
	// Keyed by `android-params`
	AndroidParams deviceParamMap
	// Values prefixed by `primary-<suffix>` or `secondary-<suffix>` will only apply to that device.
	// Keyed by `openwrt-params`
	OpenWrtParams deviceParamMap
}

func GenerateMoblyConfig(logger *log.Logger, dir string, serials []string, metadata []*api.Arg) (err error) {
	logger.Println("Generating Mobly Test Config")
	for _, arg := range metadata {
		logger.Println(arg.Flag, arg.Value)
	}
	configParams := ParseMetadata(logger, metadata)
	logger.Println("After parsing")
	for k, v := range configParams.TestParams {
		logger.Println(k, v)
	}

	for d, m := range configParams.AndroidParams {
		for k, v := range m {
			logger.Println(d, k, v)
		}
	}
	for d, m := range configParams.OpenWrtParams {
		for k, v := range m {
			logger.Println(d, k, v)
		}
	}

	Controllers := &Controllers{
		AndroidDevices: GenerateAndroidDevices(serials, configParams.AndroidParams),
		OpenWrtDevices: GenerateOpenWrtDevices(serials, configParams.OpenWrtParams),
	}

	Config := &MoblyTestConfig{
		TestBeds: []*TestBed{
			{
				Name:        "LocalTestBed",
				Controllers: Controllers,
				TestParams: &TestParams{
					Params: configParams.TestParams,
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(Config)
	if err != nil {
		err = errors.Annotate(err, "failed to marshal yaml config").Err()
		return
	}

	logger.Println(string(yamlData))

	fileName := "test_config.yml"
	yamlPath := filepath.Join(dir, fileName)
	err = os.WriteFile(yamlPath, yamlData, 0644)
	if err != nil {
		err = errors.Annotate(err, "failed to write yaml config").Err()
		return
	}

	return
}

func GenerateOpenWrtDevices(serials []string, openWrtParams deviceParamMap) []*OpenWrtDevice {
	devices := []*OpenWrtDevice{}
	deviceKeyToSerial := map[string]string{}
	for i, serial := range serials {
		if i == 0 {
			deviceKeyToSerial["primary"] = strings.TrimSuffix(serial, ":5555")
		} else {
			deviceKeyToSerial["secondary"] = strings.TrimSuffix(serial, ":5555")
		}
	}

	paramsForEachDevice := openWrtParams["all"]

	for deviceKey, deviceParams := range openWrtParams {
		if deviceKey == "all" {
			continue
		}

		deviceKey = strings.ReplaceAll(deviceKey, "primary", deviceKeyToSerial["primary"])
		deviceKey = strings.ReplaceAll(deviceKey, "secondary", deviceKeyToSerial["secondary"])

		params := paramMap{}
		for k, v := range paramsForEachDevice {
			params[k] = v
		}

		for k, v := range deviceParams {
			params[k] = v
		}

		devices = append(devices, &OpenWrtDevice{
			Hostname: deviceKey,
			Params:   params,
		})
	}

	return devices
}

func GenerateAndroidDevices(serials []string, androidParams deviceParamMap) []*AndroidDevice {
	devices := []*AndroidDevice{}

	paramsForEachDevice := androidParams["all"]

	for i, serial := range serials {
		var role string
		params := paramMap{}
		for k, v := range paramsForEachDevice {
			params[k] = v
		}

		var deviceParams paramMap
		if i == 0 {
			role = "source_device"
			deviceParams = androidParams["primary"]
		} else {
			role = "target_device"
			deviceParams = androidParams["secondary"]
		}
		for k, v := range deviceParams {
			params[k] = v
		}

		device := &AndroidDevice{
			Serial: serial,
			Role:   role,
			Params: params,
		}
		devices = append(devices, device)
	}

	return devices
}

func ParseMetadata(logger *log.Logger, metadata []*api.Arg) *ConfigParams {
	logger.Println("ParseMetadata")
	configParams := &ConfigParams{}

	for _, arg := range metadata {
		switch arg.Flag {
		case "test-params":
			logger.Println("test-params")
			configParams.TestParams = ParseArg(logger, arg)
		case "android-params":
			logger.Println("android-params")
			configParams.AndroidParams = ParseDeviceArg(logger, arg)
		case "openwrt-params":
			logger.Println("openwrt-params")
			configParams.OpenWrtParams = ParseDeviceArg(logger, arg)
		default:
			logger.Println("default")
		}
	}

	return configParams
}

func ParseArg(logger *log.Logger, arg *api.Arg) paramMap {
	logger.Println("ParseArg")
	argMap := paramMap{}

	for _, param := range strings.Split(arg.Value, ",") {
		logger.Println(param)
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		argMap[key] = value
		logger.Println(key, value)
	}

	logger.Println("ParseArg Return")
	return argMap
}

func ParseDeviceArg(logger *log.Logger, arg *api.Arg) deviceParamMap {
	logger.Println("ParseDeviceArg")
	argMap := deviceParamMap{}

	deviceKey := "all"
	argMap["all"] = paramMap{}
	for _, param := range strings.Split(arg.Value, ",") {
		logger.Println(param)
		parts := strings.Split(param, ":")
		if len(parts) == 1 {
			deviceKey = parts[0]
			argMap[deviceKey] = paramMap{}
		}
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		argMap[deviceKey][key] = value
		logger.Println(deviceKey, key, value)
	}

	logger.Println("ParseDeviceArg Return")
	return argMap
}
