// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package driver implements drivers to execute tests.
package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/chromiumos/config/go/test/api"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v2"
)

type paramMap = map[string]string

type deviceParams struct {
	deviceId string
	params   paramMap
}

func deviceParamMap(params []deviceParams) map[string]paramMap {
	res := make(map[string]paramMap)
	for _, device := range params {
		res[device.deviceId] = device.params
	}
	return res
}

type MoblyTestConfig struct {
	TestBeds []*TestBed `yaml:"TestBeds"`
}

type TestBed struct {
	Name        string       `yaml:"Name"`
	TestParams  *TestParams  `yaml:"TestParams"`
	Controllers *Controllers `yaml:"Controllers"`
}

type TestParams struct {
	Params paramMap `yaml:",inline"`
}

type Controllers struct {
	OpenWrtDevices    []*OpenWrtDevice     `yaml:"OpenWrtDevice,omitempty"`
	AndroidDevices    []*AndroidDevice     `yaml:"AndroidDevice,omitempty"`
	BtReferenceDevice []*BtReferenceDevice `yaml:"BtReferenceDevice,omitempty"`
	PassportHost      []*PassportHost      `yaml:"PassportHost,omitempty"`
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

type BtReferenceDevice struct {
	Hostname string   `yaml:"hostname"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Params   paramMap `yaml:",inline"`
}

type PassportHost struct {
	HostTopology *labapi.PasitHost
	Params       paramMap
}

func (p PassportHost) MarshalYAML() (interface{}, error) {
	topology, err := yamlFriendlyPb(p.HostTopology)
	if err != nil {
		return "", fmt.Errorf("failed to make protobuf yaml compatible: %w", err)
	}

	return struct {
		HostTopology interface{} `yaml:"host_topology"`
		Params       paramMap    `yaml:",inline"`
	}{
		HostTopology: topology,
		Params:       p.Params,
	}, nil
}

// Config Parameters from ExecutionMetadata
// Keyed by `config-params`
type ConfigParams struct {
	// Keyed by `test-params`
	TestParams paramMap
	// Values prefixed by `primary` or `secondary` will only apply to that device.
	// Keyed by `android-params`
	AndroidParams []deviceParams
	// Values prefixed by `primary-<suffix>` or `secondary-<suffix>` will only apply to that device.
	// Keyed by `openwrt-params`
	OpenWrtParams []deviceParams
	// Keyed by `btreference-params`
	BtReferenceParams []deviceParams
	// Keyed by `passport-params`
	PassportParams paramMap
}

func NewMoblyConfig(logger *log.Logger, serials []string, metadata []*api.Arg, devices []*labapi.Dut) *MoblyTestConfig {
	logger.Println("Generating Mobly Test Config")
	for _, arg := range metadata {
		logger.Println(arg.Flag, arg.Value)
	}
	configParams := ParseMetadata(logger, metadata)
	logger.Println("After parsing")
	for k, v := range configParams.TestParams {
		logger.Println(k, v)
	}

	for _, d := range configParams.AndroidParams {
		for k, v := range d.params {
			logger.Println(d.deviceId, k, v)
		}
	}
	for _, d := range configParams.OpenWrtParams {
		for k, v := range d.params {
			logger.Println(d, k, v)
		}
	}
	for _, d := range configParams.BtReferenceParams {
		for k, v := range d.params {
			logger.Println(d, k, v)
		}
	}

	Controllers := &Controllers{
		AndroidDevices:    GenerateAndroidDevices(serials, configParams.AndroidParams),
		OpenWrtDevices:    GenerateOpenWrtDevices(serials, configParams.OpenWrtParams),
		BtReferenceDevice: GenerateBtReferenceDevices(serials, configParams.BtReferenceParams),
		PassportHost:      GeneratePassportHost(logger, devices, configParams.PassportParams),
	}

	return &MoblyTestConfig{
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
}

func (c *MoblyTestConfig) Write(logger *log.Logger, dir string) (err error) {
	yamlData, err := yaml.Marshal(c)
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

func GeneratePassportHost(logger *log.Logger, devices []*labapi.Dut, passportParams paramMap) []*PassportHost {
	var passportHosts []*PassportHost
	for _, dut := range devices {
		if dut.GetChromeos() == nil || dut.GetChromeos().GetPasitHost() == nil {
			continue
		}
		topology := dut.GetChromeos().GetPasitHost()
		passportHosts = append(passportHosts, &PassportHost{
			HostTopology: topology,
			Params:       passportParams,
		})
	}

	// If test params indicate a passport device, but none was found then add one with an empty topology
	// so users can still specify based on command line.
	if len(passportHosts) == 0 && len(passportParams) > 0 {
		passportHosts = append(passportHosts, &PassportHost{
			Params:       passportParams,
			HostTopology: &labapi.PasitHost{},
		})
	}

	return passportHosts
}

func GenerateBtReferenceDevices(serials []string, btReferenceParams []deviceParams) []*BtReferenceDevice {
	devices := []*BtReferenceDevice{}
	deviceKeyToSerial := map[string]string{}
	for i, serial := range serials {
		if i == 0 {
			deviceKeyToSerial["primary"] = strings.TrimSuffix(serial, ":5555")
		} else {
			deviceKeyToSerial["secondary"] = strings.TrimSuffix(serial, ":5555")
		}
	}

	paramsForEachDevice := paramMap{}
	for _, device := range btReferenceParams {
		if device.deviceId == "all" {
			paramsForEachDevice = device.params
			break
		}
	}

	for _, device := range btReferenceParams {
		deviceKey := device.deviceId
		deviceParams := device.params
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

		devices = append(devices, &BtReferenceDevice{
			Hostname: deviceKey,
			Username: "root",
			Password: "test0000",
			Params:   params,
		})
	}

	return devices
}

func GenerateOpenWrtDevices(serials []string, openWrtParams []deviceParams) []*OpenWrtDevice {
	devices := []*OpenWrtDevice{}
	deviceKeyToSerial := map[string]string{}
	for i, serial := range serials {
		if i == 0 {
			deviceKeyToSerial["primary"] = strings.TrimSuffix(serial, ":5555")
		} else {
			deviceKeyToSerial["secondary"] = strings.TrimSuffix(serial, ":5555")
		}
	}

	paramsForEachDevice := paramMap{}
	for _, device := range openWrtParams {
		if device.deviceId == "all" {
			paramsForEachDevice = device.params
			break
		}
	}

	for _, device := range openWrtParams {
		deviceKey := device.deviceId
		deviceParams := device.params
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

func GenerateAndroidDevices(serials []string, androidParams []deviceParams) []*AndroidDevice {
	devices := []*AndroidDevice{}

	androidParamsMap := deviceParamMap(androidParams)
	paramsForEachDevice := androidParamsMap["all"]
	for i, serial := range serials {
		var role string
		params := paramMap{}
		for k, v := range paramsForEachDevice {
			params[k] = v
		}

		var deviceParams paramMap
		if i == 0 {
			role = "source_device"
			deviceParams = androidParamsMap["primary"]
		} else {
			role = "target_device"
			deviceParams = androidParamsMap["secondary"]
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
			configParams.TestParams = ParseArg(logger, arg, configParams.TestParams)
		case "android-params":
			logger.Println("android-params")
			configParams.AndroidParams = ParseDeviceArg(logger, arg)
		case "openwrt-params":
			logger.Println("openwrt-params")
			configParams.OpenWrtParams = ParseDeviceArg(logger, arg)
		case "btreference-params":
			logger.Println("btreference-params")
			configParams.BtReferenceParams = ParseDeviceArg(logger, arg)
		case "passport-params":
			logger.Println("passport-params")
			configParams.PassportParams = ParseArg(logger, arg, configParams.PassportParams)
		default:
			logger.Println("default")
			// Any additional test params dynamically passed in.
			if strings.HasPrefix(arg.Flag, "extra-test-params") {
				logger.Println("test-params")
				configParams.TestParams = ParseArg(logger, arg, configParams.TestParams)
			}
		}
	}

	return configParams
}

func ParseArg(logger *log.Logger, arg *api.Arg, argMap paramMap) paramMap {
	logger.Println("ParseArg")
	if argMap == nil {
		argMap = paramMap{}
	}

	for _, param := range strings.Split(arg.Value, ",") {
		logger.Println(param)
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if _, ok := argMap[key]; ok {
			logger.Printf("duplicate arg: %q, original value: %q, new value: %q", key, argMap[key], value)
		}
		argMap[key] = value
		logger.Println(key, value)
	}

	logger.Println("ParseArg Return")
	return argMap
}

func ParseDeviceArg(logger *log.Logger, arg *api.Arg) []deviceParams {
	logger.Println("ParseDeviceArg")
	argMap := make(map[string]paramMap)

	deviceKey := "all"
	argMap["all"] = paramMap{}
	keys := []string{"all"}
	for _, param := range strings.Split(arg.Value, ",") {
		logger.Println(param)
		parts := strings.Split(param, ":")
		if len(parts) == 1 {
			deviceKey = parts[0]
			argMap[deviceKey] = paramMap{}
			keys = append(keys, deviceKey)
		}
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		argMap[deviceKey][key] = value
		logger.Println(deviceKey, key, value)
	}

	params := []deviceParams{}
	for _, key := range keys {
		params = append(params, deviceParams{
			deviceId: key,
			params:   argMap[key],
		})
	}

	logger.Println("ParseDeviceArg Return")
	return params
}

// yamlFriendlyPb converts a protobuf message into an object that, when converted into
// yaml, can easily be unmarshalled into the original protobuf. Without this,
// the golang field names will be mangled by the default yaml marshaller.
//
//	e.g. go from:
//	  * DeviceSerial -> deviceserial
//	to
//	  * DeviceSerial -> device_serial
func yamlFriendlyPb(m proto.Message) (interface{}, error) {
	// If message is empty default to nil.
	if proto.Size(m) == 0 {
		return nil, nil
	}

	marshalOpts := protojson.MarshalOptions{
		// Optional, but helps keep marshalling in line with the rest of the testbed
		// .yaml file structs by switching to underscores vs camelCase.
		UseProtoNames: true,
	}

	// Convert to .json and then unmarshal into a new object whose fields will now
	// be marshalled
	asJson := marshalOpts.Format(proto.MessageV2(m))

	var res interface{}
	if err := json.Unmarshal([]byte(asJson), &res); err != nil {
		return nil, err
	}
	return res, nil
}
