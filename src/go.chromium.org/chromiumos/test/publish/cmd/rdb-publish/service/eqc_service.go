// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/bigquery"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	"google.golang.org/api/iterator"
)

const (
	// TesthausGCPProject Testhaus GCP project ID.
	TesthausGCPProject = "cros-test-analytics"

	// TestInfoDataset Test info BigQuery dataset name.
	TestInfoDataset = "testinfo"

	// EQCInfoTable EqC info BigQuery table name.
	EQCInfoTable = "eqc_info"

	// EQCSubjectKey 3D EqC subject key used to fetch EqC info from the
	// scheduling publish key map.
	EQCSubjectKey = "3D"
)

// EQCPublishService 3D EqC publish service to interact with BigQuery.
// Note that the caller should call the Close function to close the bqClient
// before the service is done.
type EQCPublishService struct {
	// bqClient the underlying BigQuery client.
	bqClient *bigquery.Client

	// eqcInfo the EqC info related to the current test results.
	eqcInfo *artifact.EqcInfo
}

// EQCRow represents a row in the EqC info BigQuery table.
type EQCRow struct {
	EQCHash               string
	EQCName               string
	EQCCategoryExpression string
	EQCDimensions         map[string]string
}

// NewEQCPublishService create a new EqC publish service to interact with
// BigQuery.
func NewEQCPublishService(ctx context.Context, req *api.PublishRequest) (*EQCPublishService, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is nil")
	}

	client, err := bigquery.NewClient(ctx, TesthausGCPProject)
	if err != nil {
		return nil, fmt.Errorf("creating the BQ client: %w", err)
	}

	eqcInfo, err := eqcInfo(req)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("extracting the EqC info from test results: %w", err)
	}

	return &EQCPublishService{
		bqClient: client,
		eqcInfo:  eqcInfo,
	}, nil
}

// Close closes the EqC publish service.
func (eps *EQCPublishService) Close() error {
	err := eps.bqClient.Close()
	if err != nil {
		return fmt.Errorf("close BQ client: %w", err)
	}

	return nil
}

// ExportToBQ exports the EqC info to BigQuery.
func (eps *EQCPublishService) ExportToBQ(ctx context.Context) error {
	eqcInfo := eps.eqcInfo
	log.Printf("Exporting the EqC info: %v to BQ", eqcInfo)

	if eqcInfo == nil {
		log.Printf("Skipping the BQ export because the EqC info is empty")
		return nil
	}

	exist, err := eps.checkExists(ctx, eqcInfo)
	if err != nil {
		return fmt.Errorf("checking if the EqC info already exists: %w", err)
	}

	if exist {
		log.Printf("Skipping the BQ export because the EqC info already exists: %v", eqcInfo)
		return nil
	}

	// Inserts the EqC info because it doesn't exist in the BQ table.
	eqcRow := &EQCRow{
		EQCHash:               eqcInfo.EqcHash,
		EQCName:               eqcInfo.EqcName,
		EQCCategoryExpression: eqcInfo.EqcCategoryExpression,
		EQCDimensions:         eqcInfo.EqcDimensions,
	}

	inserter := eps.bqClient.Dataset(TestInfoDataset).Table(EQCInfoTable).Inserter()
	if err := inserter.Put(ctx, eqcRow); err != nil {
		return fmt.Errorf("exporting the EqC info: %v to BQ: %w", eqcInfo, err)
	}

	log.Printf("Successfully exported the EqC info: %v to BQ", eqcInfo)
	return nil
}

// checkExists checks if the given EqC info already exists in the BQ table.
func (eps *EQCPublishService) checkExists(ctx context.Context, eqcInfo *artifact.EqcInfo) (bool, error) {
	// Constructs the query to check if the EqC info already exists.
	q := eps.bqClient.Query(fmt.Sprintf("SELECT * FROM %s.%s.%s WHERE eqc_hash = @eqc_hash", TesthausGCPProject, TestInfoDataset, EQCInfoTable))
	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "eqc_hash",
			Value: eqcInfo.EqcHash,
		},
	}

	// Runs the query.
	job, err := q.Run(ctx)
	if err != nil {
		return false, fmt.Errorf("running the BQ query: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return false, fmt.Errorf("waiting for the BQ job: %w", err)
	}
	if err := status.Err(); err != nil {
		return false, fmt.Errorf("checking the BQ job status: %w", err)
	}

	// Checks if any rows were returned.
	it, err := job.Read(ctx)
	if err != nil {
		return false, fmt.Errorf("reading the BQ job results: %w", err)
	}

	err = it.Next(&EQCRow{})
	if err == iterator.Done {
		log.Printf("EqC info doesn't exist in the BQ: %v", eqcInfo)
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("checking if the EqC info already exists: %w", err)
	} else {
		log.Printf("EqC info already exists in the BQ: %v", eqcInfo)
		return true, nil
	}
}

// Save implements the ValueSaver interface for the EQCRow struct.
func (e *EQCRow) Save() (map[string]bigquery.Value, string, error) {
	if e.EQCDimensions == nil {
		e.EQCDimensions = make(map[string]string)
	}

	dimensions, err := json.Marshal(e.EQCDimensions)
	if err != nil {
		return nil, "", fmt.Errorf("marshalling the EqC dimensions: %w", err)
	}

	return map[string]bigquery.Value{
		"eqc_hash":                e.EQCHash,
		"eqc_name":                e.EQCName,
		"eqc_category_expression": e.EQCCategoryExpression,
		"eqc_dimensions":          string(dimensions),
	}, e.EQCHash, nil
}

// eqcInfo extracts the EqC info from the rdb metadata in the publish
// request.
func eqcInfo(req *api.PublishRequest) (*artifact.EqcInfo, error) {
	metadata, err := UnpackMetadata(req)
	if err != nil {
		return nil, fmt.Errorf("unpacking the metadata in the publish request: %w", err)
	}

	eqcInfoMap := make(map[string]string)
	for _, items := range metadata.GetPublishKeys() {
		if items.GetSubject() == EQCSubjectKey {
			eqcInfoMap = items.GetKeyValues()
			break
		}
	}

	if len(eqcInfoMap) == 0 {
		return &artifact.EqcInfo{}, nil
	}

	eqcInfo := &artifact.EqcInfo{
		EqcHash:               eqcInfoMap["eqcHash"],
		EqcName:               eqcInfoMap["eqcName"],
		EqcCategoryExpression: eqcInfoMap["eqcCategoryExpression"],
	}

	if dimensionsJSON, ok := eqcInfoMap["eqcDimensions"]; ok {
		dimensionsMap := make(map[string]string)
		if err := json.Unmarshal([]byte(dimensionsJSON), &dimensionsMap); err != nil {
			return nil, fmt.Errorf("unmarshalling the EqC dimensions: %w", err)
		}
		eqcInfo.EqcDimensions = dimensionsMap
	} else {
		eqcInfo.EqcDimensions = make(map[string]string)
	}

	return eqcInfo, nil
}
