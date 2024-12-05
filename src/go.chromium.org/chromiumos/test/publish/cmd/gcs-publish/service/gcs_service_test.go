// Copyright 2024 The ChromiumOS Authors.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/mock_storage"
	"go.chromium.org/chromiumos/test/publish/cmd/publishserver/storage"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArchiveXTSResults(t *testing.T) {
	Convey("Test ArchiveXTSResults", t, func() {
		ctx := context.Background()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockStorage := mock_storage.NewMockStorageClientInterface(mockCtrl)
		gsClient := storage.NewGSTestClient(mockStorage)
		tempDir := t.TempDir()
		testCases := []struct {
			name              string
			localArtifactPath string
			resultDir         string
			metadata          *api.XtsArchiverMetadata
			createZipFile     bool
			createResultFile  bool
			wantErr           error
			wantMockCalls     []*gomock.Call
		}{
			{
				name:              "success ChromeOS run",
				localArtifactPath: tempDir,
				resultDir:         filepath.Join(tempDir, "test_prefix", "cros-test", "results", "tauto", "results-1-cheets_CTS_12.internal.all.android.dpi", "cheets_CTS_12.android.dpi", "results", "cts-results", "2024.11.17_11.51.17.458_8891"),
				metadata: &api.XtsArchiverMetadata{
					AlRun:                false,
					ApfeGcsPrefix:        "gs://apfe-bucket",
					ResultsGcsPrefix:     "gs://result-bucket",
					Build:                "brya-release/R100-14543.0.0",
					Product:              "brya.brya",
					ParentSwarmingTaskId: "parent_job_id",
				},
				createZipFile:    true,
				createResultFile: true,
				wantMockCalls: []*gomock.Call{
					mockStorage.EXPECT().Write(
						gomock.Any(),
						filepath.Join(tempDir, "test_prefix", "cros-test", "results", "tauto", "results-1-cheets_CTS_12.internal.all.android.dpi", "cheets_CTS_12.android.dpi", "results", "cts-results", "2024.11.17_11.51.17.458_8891.zip"),
						storage.GSObject{
							Bucket: "apfe-bucket",
							Object: "brya.brya-release/R100-14543.0.0/parent_job_id/cheets_CTS_12.android.dpi/tauto_2024.11.17_11.51.17.458_8891/2024.11.17_11.51.17.458_8891.zip",
						}).Return(nil),
					mockStorage.EXPECT().Write(
						gomock.Any(), filepath.Join(tempDir, "test_prefix", "cros-test", "results", "tauto", "results-1-cheets_CTS_12.internal.all.android.dpi", "cheets_CTS_12.android.dpi", "results", "cts-results", "2024.11.17_11.51.17.458_8891", "test_result.xml.gz"),
						storage.GSObject{
							Bucket: "result-bucket",
							Object: "cheets_CTS_12.android.dpi/tauto_2024.11.17_11.51.17.458_8891/test_result.xml.gz",
						}).Return(nil),
				},
			},
		}

		for _, tc := range testCases {
			Convey(tc.name, func() {
				if tc.createZipFile {
					zipPath := fmt.Sprintf("%s.zip", tc.resultDir)
					if err := os.MkdirAll(tc.resultDir, 0755); err != nil {
						t.Fatalf("failed to create directory structure for file: %v", err)
					}
					if _, err := os.Create(zipPath); err != nil {
						t.Fatalf("failed to create test zip file: %v", err)
					}
				}
				if tc.createResultFile {
					resultPath := filepath.Join(tc.resultDir, "test_result.xml")
					createFile(t, resultPath)
				}
				defer os.RemoveAll(tc.resultDir)

				gs := &GcsPublishService{
					Client:                      gsClient,
					LocalArtifactPath:           tc.localArtifactPath,
					ServiceAccountCredsFilePath: "/path/to/credentials",
					XtsArchiverMetadata:         tc.metadata,
				}
				err := gs.ArchiveXTSResults(ctx)

				gomock.InOrder(tc.wantMockCalls...)
				So(err, ShouldResemble, tc.wantErr)

			})
		}
	})
}
