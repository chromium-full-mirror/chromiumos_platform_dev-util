// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	storage_path "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	api1 "go.chromium.org/chromiumos/config/go/test/lab/api"
	testapi "go.chromium.org/chromiumos/config/go/test/lab/api"
)

// Run the filter locally by building the binary ex- ./use-flag-filter -p.
// Copy the port and update it in this client below and run the client.
// Update the request as per requirement.

func main() {
	// Set up the connection to the gRPC server
	conn, err := grpc.NewClient("localhost:33757", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a new instance of the gRPC client
	client := api.NewGenericFilterServiceClient(conn)

	skyrim := &testapi.Dut_ChromeOS{DutModel: &testapi.DutModel{
		BuildTarget: "skyrim",
		ModelName:   "frostflow",
	}}
	octopus := &testapi.Dut_ChromeOS{DutModel: &testapi.DutModel{
		BuildTarget: "octopus",
		ModelName:   "bluebird",
	}}
	android := &testapi.Dut_Android{DutModel: &testapi.DutModel{
		BuildTarget: "android",
		ModelName:   "android",
	}}
	metadata, err := anypb.New(&api.CrOSProvisionMetadata{})
	if err != nil {
		log.Fatalf("Error creating anypb.Any: %v", err)
	}

	req := &api.InternalTestplan{
		TestCases: []*api.CTPTestCase{
			{
				Name: "abc",
				Metadata: &api.TestCaseMetadata{
					TestCase: &api.TestCase{
						Id: &api.TestCase_Id{
							Value: "abc",
						},
						BuildDependencies: []*api.TestCase_BuildDeps{
							{
								Value: "pita",
							},
							{
								Value: "varun",
							},
						},
					},
				},
				SchedulingUnitOptions: []*api.SchedulingUnitOptions{
					{
						SchedulingUnits: []*api.SchedulingUnit{
							{
								PrimaryTarget: &api.Target{
									SwarmingDef: &api.SwarmingDefinition{
										DutInfo: &api1.Dut{
											DutType: &api1.Dut_Chromeos{Chromeos: octopus},
										},
										SwarmingLabels: []string{
											"label-peripheral_wifi_state:WORKING",
											"label-wificell:True",
										},
									},
									SwReq: &api.LegacySW{},
								},
								DynamicUpdateLookupTable: map[string]string{
									"board":       "skyrim",
									"installPath": "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
								},
							},
						},
					},
					{
						SchedulingUnits: []*api.SchedulingUnit{
							{
								PrimaryTarget: &api.Target{
									SwarmingDef: &api.SwarmingDefinition{
										DutInfo: &api1.Dut{
											DutType: &api1.Dut_Chromeos{Chromeos: skyrim},
										},
										SwarmingLabels: []string{
											"label-peripheral_wifi_state:WORKING",
											"label-wificell:True",
										},
									},
									SwReq: &api.LegacySW{},
								},
								DynamicUpdateLookupTable: map[string]string{
									"board":       "skyrim",
									"installPath": "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
								},
							},
						},
					},
					{
						SchedulingUnits: []*api.SchedulingUnit{
							{
								PrimaryTarget: &api.Target{
									SwarmingDef: &api.SwarmingDefinition{
										DutInfo: &api1.Dut{
											DutType: &api1.Dut_Android_{Android: android},
										},
										SwarmingLabels: []string{
											"label-peripheral_wifi_state:WORKING",
											"label-wificell:True",
										},
									},
									SwReq: &api.LegacySW{},
								},
								DynamicUpdateLookupTable: map[string]string{
									"board":       "skyrim",
									"installPath": "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
								},
							},
						},
					},
				},
			},
		},
		SuiteInfo: &api.SuiteInfo{
			SuiteMetadata: &api.SuiteMetadata{
				Pool: "wificell_perbuild",
				ExecutionMetadata: &api.ExecutionMetadata{
					Args: []*api.Arg{},
				},
				SchedulerInfo: &api.SchedulerInfo{
					Scheduler: 2,
					QsAccount: "unmanaged_p4",
				},
				DynamicUpdates: []*api.UserDefinedDynamicUpdate{
					{
						FocalTaskFinder: &api.FocalTaskFinder{
							Finder: &api.FocalTaskFinder_First_{
								First: &api.FocalTaskFinder_First{
									TaskType: 2,
								},
							},
						},
						UpdateAction: &api.UpdateAction{
							Action: &api.UpdateAction_Insert_{
								Insert: &api.UpdateAction_Insert{
									InsertType: 1,
									Task: &api.CrosTestRunnerDynamicRequest_Task{
										OrderedContainerRequests: []*api.ContainerRequest{
											{
												DynamicIdentifier: "crosDutServer_primary",
												Container: &api.Template{
													Container: &api.Template_CrosDut{},
												},
												DynamicDeps: []*api.DynamicDep{
													{
														Key:   "crosDut.cacheServer",
														Value: "device_primary.dut.cacheServer.address",
													},
													{
														Key:   "crosDut.dutAddress",
														Value: "device_primary.dutserver",
													},
												},
												ContainerImageKey: "cros-dut",
											},
											{
												DynamicIdentifier: "cros-provision_primary",
												Container: &api.Template{
													Container: &api.Template_Generic{
														Generic: &api.GenericTemplate{
															BinaryName: "cros-provision",
															BinaryArgs: []string{
																"server",
																"-port",
																"0",
															},
															DockerArtifactDir: "/tmp/provisionservice",
															AdditionalVolumes: []string{
																"/creds:/creds",
															},
														},
													},
												},
												ContainerImageKey: "cros-dut",
											},
										},
										Task: &api.CrosTestRunnerDynamicRequest_Task_Provision{
											Provision: &api.ProvisionTask{
												ServiceAddress: &testapi.IpEndpoint{},
												StartupRequest: &api.ProvisionStartupRequest{},
												InstallRequest: &api.InstallRequest{
													ImagePath: &storage_path.StoragePath{
														HostType: 2,
														Path:     "${installPath}",
													},
													Metadata: metadata,
												},
												DynamicDeps: []*api.DynamicDep{
													{
														Key:   "serviceAddress",
														Value: "cros-provision_primary",
													},
													{
														Key:   "startupRequest.dut",
														Value: "device_primary.dut",
													},
													{
														Key:   "startupRequest.dutServer",
														Value: "crosDutServer_primary",
													},
												},
												Target:            "primary",
												DynamicIdentifier: "cros-provision_primary",
											},
										},
									},
								},
							},
						},
					},
				},
				SchedulingUnits: []*api.SchedulingUnit{
					{
						PrimaryTarget: &api.Target{
							SwarmingDef: &api.SwarmingDefinition{
								DutInfo: &api1.Dut{
									DutType: &api1.Dut_Chromeos{Chromeos: skyrim},
								},
								ProvisionInfo: []*api.ProvisionInfo{
									{
										InstallRequest: &api.InstallRequest{
											ImagePath: &storage_path.StoragePath{
												HostType: 2,
												Path:     "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
											},
										},
									},
								},
							},
							SwReq: &api.LegacySW{
								Build:   "release",
								GcsPath: "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
								KeyValues: []*api.KeyValue{
									{
										Key:   "chromeos_build_gcs_bucket",
										Value: "chromeos-image-archive",
									},
									{
										Key:   "chromeos_build",
										Value: "skyrim-release/R128-15964.4.0",
									},
								},
							},
						},
						DynamicUpdateLookupTable: map[string]string{
							"board":       "skyrim",
							"installPath": "gs://chromeos-image-archive/skyrim-release/R128-15964.4.0",
						},
					},
				},
			},
			SuiteRequest: &api.SuiteRequest{
				SuiteRequest: &api.SuiteRequest_TestSuite{
					TestSuite: &api.TestSuite{
						Name: "wifi_endtoend",
						Spec: &api.TestSuite_TestCaseTagCriteria_{
							TestCaseTagCriteria: &api.TestSuite_TestCaseTagCriteria{
								Tags: []string{"suite:wifi_endtoend"},
							},
						},
					},
				},
				MaximumDuration: durationpb.New(142200),
				TestArgs:        "None",
				AnalyticsName:   "wifi_endtoend_perbuild__wificell_perbuild__wifi_endtoend__tauto__new_build",
			},
		},
	}

	response, err := client.Execute(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to call RPC method: %v", err)
	}

	// Process the response
	fmt.Println("Response:", response)
}
