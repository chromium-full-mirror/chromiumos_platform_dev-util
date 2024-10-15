// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// DLC constants and helpers
package cross_over

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/util/adb"
	"golang.org/x/crypto/ssh"
)

type OS_TYPE int

const (
	ANDROID OS_TYPE = iota
	CROS
	UNKNOWN
)

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

func DetectOS(logger *log.Logger, dutAddress string) (OS_TYPE, error) {
	logger.Println("Detecting OS type")
	config := GetSSHConfig()
	// Perform fast connection check.
	if checkSSH(logger, dutAddress, config, 2, 2*time.Second) {
		return CROS, nil
	}
	if checkADB(logger, dutAddress, 10*time.Second) {
		return ANDROID, nil
	}

	// Perform long connection check.
	if checkSSH(logger, dutAddress, config, 5, 10*time.Second) {
		return CROS, nil
	}
	if checkADB(logger, dutAddress, 3*time.Minute) {
		return ANDROID, nil
	}
	return UNKNOWN, fmt.Errorf("could not detect underlining os for dut")
}

func callServod(ctx context.Context, log *log.Logger, servodClient api.ServodServiceClient, key, value, servoHostPath, servodDockerContainerName string, servodPort int32) error {
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
	req.ServoHostPath = servoHostPath
	req.ServodDockerContainerName = servodDockerContainerName
	req.ServodPort = servodPort
	log.Printf("\nMaking CallServod Call with req %v", req)
	response, err := servodClient.CallServod(ctx, req)
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
