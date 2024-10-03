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
)

type OS_TYPE int

const (
	ANDROID OS_TYPE = iota
	CROS
)

func DetectOS(logger *log.Logger, dutAddress string) (OS_TYPE, error) {
	// TODO: If its already an android DUT, then we can optimize and avoid calling SetupAdb two times, here and in FoilPreInitState.
	if err := adb.RetrySetupAdb(logger, dutAddress, 3*time.Minute); err != nil {
		// ADB failure implies DUT is CROS.
		// TODO: Do second validation that DUT is cros by invoking GetConnectionWithRetry.
		return CROS, nil
	}
	if err := adb.TeardownAdb(logger, dutAddress); err != nil {
		logger.Println("Warning: TeardownAdb failed: ", err)
	}
	return ANDROID, nil
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
