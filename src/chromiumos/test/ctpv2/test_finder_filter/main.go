// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	server "chromiumos/test/ctpv2/common/server_template"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"google.golang.org/grpc"
)

const (
	address = "localhost"
	binName = "test_finder_filter"
)

func toCTPTestCase(metadata *api.TestCaseMetadata) *api.CTPTestCase {
	hwReqs := api.HWRequirements{}
	for _, dep := range metadata.GetTestCase().GetDependencies() {
		hwReqs.SwarmingLabels = append(hwReqs.SwarmingLabels, dep.GetValue())
	}
	return &api.CTPTestCase{
		Name:           metadata.GetTestCase().GetName(),
		HwRequirements: []*api.HWRequirements{&hwReqs},
		Metadata:       metadata,
	}
}
func toTestFinderRequest(testPlan *api.InternalTestplan) (*api.CrosTestFinderRequest, error) {
	// TODO... switch
	testSuite, ok := testPlan.GetSuiteInfo().GetSuiteRequest().GetSuiteRequest().(*api.SuiteRequest_TestSuite)
	if !ok {
		return nil, errors.New("SuiteRequest is not TestSuite")
	}
	return &api.CrosTestFinderRequest{
		TestSuites:       []*api.TestSuite{testSuite.TestSuite},
		MetadataRequired: true,
	}, nil
}

// FillTestCases will make cros-test-finder request from SuiteInfo.SuiteRequest,
// and fill in the TestCases field with test cases information from the response.
func fillTestCases(ctx context.Context, testPlan *api.InternalTestplan, client api.TestFinderServiceClient, log *log.Logger) error {
	req, err := toTestFinderRequest(testPlan)
	if err != nil {
		return fmt.Errorf("Failed to Translate Request: %s", err)
	}

	log.Printf("Serving Request: %s", req)
	resp, err := client.FindTests(ctx, req)
	log.Printf("Got Response: %s", resp)
	if err != nil {
		return fmt.Errorf("Failed to call cros-test-finder API: %s", err)
	}

	// TODO loop through testSuites.
	if len(resp.GetTestSuites()) == 0 {
		return nil
	}
	metadataList, ok := resp.GetTestSuites()[0].Spec.(*api.TestSuite_TestCasesMetadata)
	if !ok {
		return errors.New("no test cases metadata in the response")
	}
	log.Println("Going to translate...")
	for _, metadata := range metadataList.TestCasesMetadata.GetValues() {
		log.Println(metadata)
		testPlan.TestCases = append(testPlan.TestCases, toCTPTestCase(metadata))
	}
	return nil
}

func startTestFinderServer(log *log.Logger) (string, error) {
	args := []string{"server", "-port", "0"}
	cmd := exec.Command("cros-test-finder", args...)
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("unable to start test-finder: %s", err)
	}
	port, err := defaultDiscoverPort()
	if err != nil {
		return "", fmt.Errorf("unable to find test-finder port: %s", err)
	}

	log.Printf("found test-finder port at :%s", port)
	return port, nil
}

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	port, err := startTestFinderServer(log)
	if err != nil {
		log.Println("err starting TestFinder")
		return req, err
	}
	log.Printf("Started Test Finder Server. @ %s:%s\n", address, port)
	ctx := context.Background()
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", address, port), grpc.WithInsecure())

	log.Println("Connected to test Finder Server")
	if err != nil {
		log.Printf("err while dialing: %s", err)
	}
	defer conn.Close()
	client := api.NewTestFinderServiceClient(conn)

	log.Println("Created Test Finder Client")

	err = fillTestCases(ctx, req, client, log)
	if err != nil {
		return req, err
	}
	return req, nil
}

func wait1() {
	time.Sleep(1 * time.Second)
}

func waitForMetadata() error {
	timeOut := time.After(5 * time.Second)
	for {
		select {
		case <-timeOut:
			return fmt.Errorf("Unable to find %s server info", binName)
		default:
			cmd := exec.Command("/bin/bash", "-c", "source ~/.cftmeta && echo $SERVICE_NAME")
			out, err := cmd.Output()
			if err != nil {
				wait1()
			}
			if strings.TrimSpace(fmt.Sprintf("%s", out)) == binName {
				return nil
			}
			wait1()
		}
	}
}

func defaultDiscoverPort() (string, error) {

	err := waitForMetadata()
	if err != nil {
		return "", err
	}
	fmt.Println("Metadata source found.")

	cmd := exec.Command("/bin/bash", "-c", "source ~/.cftmeta && echo $SERVICE_PORT")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%v with stderr: %s", err, out)
	}

	return strings.TrimSpace(fmt.Sprintf("%s", out)), nil
}

func main() {
	err := server.Server(executor, binName)
	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}
