// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"testing"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/api/metadata"
	"go.chromium.org/chromiumos/config/go/test/artifact"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestNewEQCPublishService(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey("NewEQCPublishService", t, func() {
		Convey("Valid request", func() {
			validEQCInfo := &artifact.EqcInfo{
				EqcHash:               "A310930190d11",
				EqcName:               "IntelRaptorLakeKernelNext",
				EqcCategoryExpression: "WifiBtChipset_Soc_Kernel_80211_AX",
			}
			validMetadata := &metadata.PublishRdbMetadata{
				TestResult: &artifact.TestResult{
					TestInvocation: &artifact.TestInvocation{
						EqcInfo: validEQCInfo,
					},
				},
			}
			validMetadataAny, _ := anypb.New(validMetadata)
			req := &api.PublishRequest{
				Metadata: validMetadataAny,
			}

			s, err := NewEQCPublishService(ctx, req)

			So(err, ShouldBeNil)
			So(s, ShouldNotBeNil)
			So(s.eqcInfo, ShouldResembleProto, validEQCInfo)

			s.Close()
		})

		Convey("Nil request", func() {
			s, err := NewEQCPublishService(ctx, nil)

			So(err, ShouldNotBeNil)
			So(s, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "request is nil")
		})

		Convey("Nil metadata", func() {
			req := &api.PublishRequest{}
			s, err := NewEQCPublishService(ctx, req)

			So(err, ShouldNotBeNil)
			So(s, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid nil source message")
		})

		Convey("Invalid metadata type", func() {
			req := &api.PublishRequest{
				Metadata: &anypb.Any{}, // Empty Any proto
			}

			s, err := NewEQCPublishService(ctx, req)

			So(err, ShouldNotBeNil)
			So(s, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "unpacking the metadata")
		})
	})
}

func TestEQCRowSave(t *testing.T) {
	t.Parallel()

	eqcHash := "A310930190d11"
	eqcName := "IntelRaptorLakeKernelNext"
	eqcCategoryExpression := "WifiBtChipset_Soc_Kernel_80211_AX"

	Convey("EQCRow Save", t, func() {
		Convey("Valid entry", func() {
			entry := &EQCRow{
				EQCHash:               eqcHash,
				EQCName:               eqcName,
				EQCCategoryExpression: eqcCategoryExpression,
				EQCDimensions: map[string]string{
					"soc":      "value1",
					"wifiChip": "value2",
				},
			}

			wantValue := map[string]bigquery.Value{
				"eqc_hash":                eqcHash,
				"eqc_name":                eqcName,
				"eqc_category_expression": eqcCategoryExpression,
				"eqc_dimensions":          `{"soc":"value1","wifiChip":"value2"}`, // Note: order may vary
			}

			gotValue, gotInsertID, err := entry.Save()

			So(err, ShouldBeNil)
			So(gotInsertID, ShouldEqual, wantValue["eqc_hash"])
			So(gotValue["eqc_hash"], ShouldEqual, wantValue["eqc_hash"])
			So(gotValue["eqc_name"], ShouldEqual, eqcName)
			So(gotValue["eqc_category_expression"], ShouldEqual, wantValue["eqc_category_expression"])

			// Compare maps, ignoring key order in eqc_dimensions.
			gotDimensions := gotValue["eqc_dimensions"]
			wantDimensions := wantValue["eqc_dimensions"]

			var gotDimMap, wantDimMap map[string]string
			err = json.Unmarshal([]byte(gotDimensions.(string)), &gotDimMap)
			So(err, ShouldBeNil)
			err = json.Unmarshal([]byte(wantDimensions.(string)), &wantDimMap)
			So(err, ShouldBeNil)
			So(gotDimMap, ShouldResemble, wantDimMap)

		})

		Convey("Empty dimensions", func() {
			entry := &EQCRow{
				EQCHash:               eqcHash,
				EQCName:               eqcName,
				EQCCategoryExpression: eqcCategoryExpression,
				EQCDimensions:         make(map[string]string), // Empty map
			}

			wantValue := map[string]bigquery.Value{
				"eqc_hash":                eqcHash,
				"eqc_name":                eqcName,
				"eqc_category_expression": eqcCategoryExpression,
				"eqc_dimensions":          `{}`,
			}

			gotValue, _, err := entry.Save()

			So(err, ShouldBeNil)
			So(gotValue, ShouldResemble, wantValue)
		})

		Convey("Nil dimensions", func() {
			entry := &EQCRow{
				EQCHash:               eqcHash,
				EQCName:               eqcName,
				EQCCategoryExpression: eqcCategoryExpression,
				EQCDimensions:         nil,
			}

			wantValue := map[string]bigquery.Value{
				"eqc_hash":                eqcHash,
				"eqc_name":                eqcName,
				"eqc_category_expression": eqcCategoryExpression,
				"eqc_dimensions":          `{}`, // Should be treated as an empty map
			}

			gotValue, _, err := entry.Save()

			So(err, ShouldBeNil)
			So(gotValue, ShouldResemble, wantValue)
		})
	})
}
