// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	server "chromiumos/test/ctpv2/common/server_template"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/config/go/test/api"

	"chromiumos/test/test_finder"
)

const (
	address       = "localhost"
	binName       = "test_finder_filter"
	testFinderDir = "/tmp/test-finder"
	mdDir         = "/tmp/test/metadata"
)

func toCTPTestCase(metadata *api.TestCaseMetadata) *api.CTPTestCase {

	return &api.CTPTestCase{
		Name:     metadata.GetTestCase().GetName(),
		Metadata: metadata,
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

// fillTestCases will make cros-test-finder request from SuiteInfo.SuiteRequest,
// and fill in the TestCases field with test cases information from the response.
func fillTestCases(ctx context.Context, testPlan *api.InternalTestplan, resp *api.CrosTestFinderResponse, log *log.Logger) error {

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

func executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	ctx := context.Background()

	testFinderResponse, err := startAndRunTestFinder(req, log)
	if err != nil {
		log.Println("Error with TestFinder: ", err)
		return req, err
	}
	log.Println("Called TestFinder successfully:", fmt.Sprintf("%s", testFinderResponse))

	err = fillTestCases(ctx, req, testFinderResponse, log)
	if err != nil {
		log.Println("Error with setting internalPlan: ", err)
		return req, err
	}
	return req, nil
}

func startAndRunTestFinder(testPlan *api.InternalTestplan, log *log.Logger) (*api.CrosTestFinderResponse, error) {
	err := os.Mkdir(testFinderDir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	req, err := toTestFinderRequest(testPlan)
	if err != nil {
		return nil, fmt.Errorf("Failed to Translate Request: %s", err)
	}
	log.Printf("made req: %s", req)

	response, err := test_finder.FindTests(log, req, mdDir)
	if err != nil {
		return nil, fmt.Errorf("unable to run test-finder: %s", err)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed2 to call test-finder: %s", err)
	}

	log.Printf("Got Response: %s", response)

	return response, nil

}

func getLog() string {
	fileContents, _ := ioutil.ReadFile(fmt.Sprintf("%s/log.txt", testFinderDir))
	return string(fileContents)
}

// writeTestFinderInput writes a CrosTestFinderRequest json.
func writeTestFinderInput(file string, req *api.CrosTestFinderRequest) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("fail to create file: %s", err)
	}
	m := jsonpb.Marshaler{}
	if err := m.Marshal(f, req); err != nil {
		return fmt.Errorf("fail to marshal request file: %s", err)
	}
	return nil
}

func main() {
	err := server.Server(executor, binName)
	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}
