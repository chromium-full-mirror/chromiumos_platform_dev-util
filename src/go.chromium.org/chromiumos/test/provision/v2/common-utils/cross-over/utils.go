// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	common_utils "go.chromium.org/chromiumos/test/provision/v2/common-utils"
	"go.chromium.org/chromiumos/test/util/adb"
	"golang.org/x/crypto/ssh"
)

type OS_TYPE int

const (
	ANDROID OS_TYPE = iota
	CROS
	UNKNOWN
)

const connectionResetErr = "connection reset by peer"

func connectWithTimeout(addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	result := make(chan *ssh.Client)
	errChan := make(chan error)

	go func() {
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			errChan <- err
		} else {
			result <- client
		}
	}()

	select {
	case client := <-result:
		return client, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("SSH timed out")
	}
}

func checkSSH(logger *log.Logger, dutAddress string, config *ssh.ClientConfig, retryCount int, retryInterval time.Duration) bool {
	logger.Println("Checking for ssh connection. w. retry count: ", retryCount)
	for ; retryCount >= 0; retryCount-- {
		logger.Println("attempting to SSH dail the device")

		_, err := connectWithTimeout(dutAddress, config, 5*time.Second)
		if err == nil {
			logger.Println("SSH dail the device successfull")

			return true
		}
		logger.Println("Got error while ssh. Retrying...:", retryInterval)
		time.Sleep(retryInterval)
	}
	return false
}

func checkADB(logger *log.Logger, dutAddress string, timeout time.Duration) bool {
	logger.Println("Checking for adb connection. w/ timeout: ", timeout)
	err := adb.RetrySetupAdb(logger, dutAddress, timeout)
	if err != nil {
		return false
	}
	if err := adb.TeardownAdb(logger, dutAddress); err != nil {
		logger.Println("Warning: TeardownAdb failed: ", err)
	}
	return true
}

func IsCROS(logger *log.Logger, dutAddress string) bool {
	config := GetSSHConfig()
	return checkSSH(logger, dutAddress, config, 5, 2*time.Second)
}

func DetectOS(logger *log.Logger, dutAddress string) (OS_TYPE, error) {
	logger.Println("Detecting OS type")
	config := GetSSHConfig()
	// Perform fast connection check.
	if checkSSH(logger, dutAddress, config, 2, 2*time.Second) {
		return CROS, nil
	}
	if checkADB(logger, dutAddress, 5*time.Second) {
		return ANDROID, nil
	}

	// Perform long connection check.
	if checkSSH(logger, dutAddress, config, 5, 10*time.Second) {
		return CROS, nil
	}
	if checkADB(logger, dutAddress, 15*time.Second) {
		return ANDROID, nil
	}
	return UNKNOWN, fmt.Errorf("could not detect underlining os for dut")
}

func callServodRetry(ctx context.Context, log *log.Logger, key, value string, params *CrossOverParameters) error {
	err := callServod(ctx, log, key, value, params)
	if err != nil && strings.Contains(err.Error(), connectionResetErr) {
		log.Println("Got connection reset error in CallServod, restarting servod process and trying again. ", err)
		if err = startServod(ctx, log, params.ServoNexusClient, params.Dut); err != nil {
			return fmt.Errorf("failed to restart servod %v", err)
		}
		return callServod(ctx, log, key, value, params)
	}
	return err
}

func callServod(ctx context.Context, log *log.Logger, key, value string, params *CrossOverParameters) error {
	log.Println("Making CallServo with key " + key + " and value " + value)
	req := &api.CallServodRequest{}
	req.Method = api.CallServodRequest_SET
	req.Args = []*xmlrpc.Value{
		{
			ScalarOneof: &xmlrpc.Value_String_{
				String_: key,
			},
		},
		{
			ScalarOneof: &xmlrpc.Value_String_{
				String_: value,
			},
		},
	}
	req.ServoHostPath = params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	if strings.Contains(req.ServoHostPath, satlab) {
		req.ServodDockerContainerName = params.Dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	}
	req.ServodPort = params.Dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	log.Printf("\nMaking CallServod Call with req %v", req)
	response, err := params.ServoNexusClient.CallServod(ctx, req)
	if err != nil {
		return err
	}
	switch result := response.GetResult().(type) {
	case *api.CallServodResponse_Failure_:
		return fmt.Errorf("Call Servod failed error: " + result.Failure.ErrorMessage)
	case *api.CallServodResponse_Success_:
		log.Printf("\nCallServod Call succeeded with response %v", response)
		return nil
	default:
		return fmt.Errorf("unexpected result type for long running operation")
	}
}

func startServod(ctx context.Context, log *log.Logger, servoNexusClient api.ServodServiceClient, dut *lab_api.Dut) error {
	req := &api.StartServodRequest{}
	req.Board = dut.GetChromeos().GetDutModel().GetBuildTarget()
	req.Model = dut.GetChromeos().GetDutModel().GetModelName()
	req.SerialName = dut.GetChromeos().GetServo().GetSerial()
	req.ServodPort = dut.GetChromeos().GetServo().GetServodAddress().GetPort()
	req.ServoHostPath = dut.GetChromeos().GetServo().GetServodAddress().GetAddress() + sshPort
	if strings.Contains(req.ServoHostPath, satlab) {
		req.ServodDockerContainerName = dut.GetChromeos().GetServo().GetServodAddress().GetAddress()
	}
	log.Printf("\nCalling start servod with request %v.", req)
	op, err := servoNexusClient.StartServod(ctx, req)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = common_utils.WaitLongRunningOp(ctx, log, op)
	return err
}
