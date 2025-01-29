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
			wantEQCInfo := &artifact.EqcInfo{
				EqcHash:               "9073744604696850342",
				EqcName:               "Cometlake-U__INTEL_HRP2_AX201__5.15",
				EqcCategoryExpression: "WifiBtChipset_Soc_Kernel_Intel",
				EqcDimensions: map[string]string{
					"dlm:soc":               "Cometlake-U",
					"image:_kernel_version": "5.15",
					"wireless_field":        "INTEL_HRP2_AX201",
				},
			}
			eqcInfoMap := map[string]string{
				"eqcCategoryExpression": "WifiBtChipset_Soc_Kernel_Intel",
				"eqcDimensions":         "{\"dlm:soc\":\"Cometlake-U\",\"image:_kernel_version\":\"5.15\",\"wireless_field\":\"INTEL_HRP2_AX201\"}",
				"eqcHash":               "9073744604696850342",
				"eqcName":               "Cometlake-U__INTEL_HRP2_AX201__5.15",
			}
			rdbMetadata := &metadata.PublishRdbMetadata{
				PublishKeys: []*api.PublishKey{
					{
						Subject:   EQCSubjectKey,
						KeyValues: eqcInfoMap,
					},
				},
			}

			validMetadataAny, err := anypb.New(rdbMetadata)
			So(err, ShouldBeNil)

			req := &api.PublishRequest{
				Metadata: validMetadataAny,
			}

			s, err := NewEQCPublishService(ctx, req)

			So(err, ShouldBeNil)
			So(s, ShouldNotBeNil)
			So(s.eqcInfo, ShouldResembleProto, wantEQCInfo)

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

// Generate unit tests for the function eqcInfo(req *api.PublishRequest)
func TestEqcInfo(t *testing.T) {
	t.Parallel()

	Convey("Valid request with EQC info", t, func() {
		wantEQCInfo := &artifact.EqcInfo{
			EqcHash:               "9073744604696850342",
			EqcName:               "Cometlake-U__INTEL_HRP2_AX201__5.15",
			EqcCategoryExpression: "{\"value\":{\"combinatorial\":{\"subcategories\":[{\"value\":{\"enumerated\":{\"classes\":[{\"value\":{\"name\":\"swarming:label-wifi_state:NORMAL\",\"expression\":{\"property\":{\"propertyPath\":\"swarming:label-wifi_state\",\"strEqual\":\"NORMAL\"}}}}]}}},{\"name\":\"WifiBtChipset_Soc_Kernel\"}]}}}",
			EqcDimensions: map[string]string{
				"dlm:soc":               "Cometlake-U",
				"image:_kernel_version": "5.15",
				"wireless_field":        "INTEL_HRP2_AX201",
			},
		}
		eqcInfoMap := map[string]string{
			"eqcCategoryExpression": "{\"value\":{\"combinatorial\":{\"subcategories\":[{\"value\":{\"enumerated\":{\"classes\":[{\"value\":{\"name\":\"swarming:label-wifi_state:NORMAL\",\"expression\":{\"property\":{\"propertyPath\":\"swarming:label-wifi_state\",\"strEqual\":\"NORMAL\"}}}}]}}},{\"name\":\"WifiBtChipset_Soc_Kernel\"}]}}}",
			"eqcDimensions":         "{\"dlm:soc\":\"Cometlake-U\",\"image:_kernel_version\":\"5.15\",\"wireless_field\":\"INTEL_HRP2_AX201\"}",
			"eqcHash":               "9073744604696850342",
			"eqcName":               "Cometlake-U__INTEL_HRP2_AX201__5.15",
		}
		rdbMetadata := &metadata.PublishRdbMetadata{
			PublishKeys: []*api.PublishKey{
				{
					Subject:   EQCSubjectKey,
					KeyValues: eqcInfoMap,
				},
			},
		}

		validMetadataAny, err := anypb.New(rdbMetadata)
		So(err, ShouldBeNil)

		req := &api.PublishRequest{
			Metadata: validMetadataAny,
		}
		gotEQCInfo, err := eqcInfo(req)
		So(err, ShouldBeNil)
		So(gotEQCInfo, ShouldResembleProto, wantEQCInfo)
	})

	Convey("Valid request without EQC info", t, func() {
		validMetadataAny, err := anypb.New(&metadata.PublishRdbMetadata{})
		So(err, ShouldBeNil)

		req := &api.PublishRequest{
			Metadata: validMetadataAny,
		}
		gotEQCInfo, err := eqcInfo(req)

		// Expect an empty EqcInfo
		So(err, ShouldBeNil)
		So(gotEQCInfo, ShouldResembleProto, &artifact.EqcInfo{})
	})

	Convey("Nil request", t, func() {
		gotEQCInfo, err := eqcInfo(nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "unpacking the metadata")
		So(gotEQCInfo, ShouldBeNil)
	})

	Convey("Invalid metadata", t, func() {
		req := &api.PublishRequest{
			Metadata: &anypb.Any{}, // Empty Any proto
		}
		gotEQCInfo, err := eqcInfo(req)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "unpacking the metadata")
		So(gotEQCInfo, ShouldBeNil)

	})
}
