// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/fileutils"
	"go.chromiumos.org/chromiumos/platform/dev/contrib/cros_openwrt/image_builder/log"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	ChromeOSConnectivityTestArtifactsStorageBucket = "chromeos-connectivity-test-artifacts"
	baseStorageDir                                 = "wifi_router"
	imagesDirStoragePath                           = "openwrt_images"
	configStoragePathProd                          = "wifi_router_config_prod.json"
	configStoragePathTest                          = "wifi_router_config_test.json"
)

var sharedManagerInstance *Manager

type ManagerConfig struct {
	StorageBucketName      string
	UseProdConfig          bool
	GCSCredentialsFilePath string
}

type Manager struct {
	useProdConfig    bool
	gcsBucket        *storage.BucketHandle
	wifiRouterConfig *labapi.WifiRouterConfig
}

func SharedManagerInstance() *Manager {
	return sharedManagerInstance
}

func SetSharedManagerInstance(m *Manager) {
	sharedManagerInstance = m
}

func NewManager(ctx context.Context, managerConfig *ManagerConfig) (*Manager, error) {
	m := &Manager{
		useProdConfig: managerConfig.UseProdConfig,
	}
	var err error
	var gcsClient *storage.Client
	if managerConfig.GCSCredentialsFilePath == "" {
		log.Logger.Println("Creating new GCS client using application-default credentials")
		gcsClient, err = storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new GCS client using application default credentials (you need to run 'gcloud auth application-default login' first or use a cred file): %w", err)
		}
	} else {
		log.Logger.Printf("Creating new GCS client using credentials file %q\n", managerConfig.GCSCredentialsFilePath)
		gcsClient, err = storage.NewClient(ctx, option.WithCredentialsFile(managerConfig.GCSCredentialsFilePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create new GCS client with cred file %q: %w", managerConfig.GCSCredentialsFilePath, err)
		}
	}
	m.gcsBucket = gcsClient.Bucket(managerConfig.StorageBucketName)
	return m, nil
}

func (m *Manager) imageFileObject(deviceName string, imageUUID string, fileName string) (string, *storage.ObjectHandle) {
	deviceName = m.cleanObjectNamePathSegment(deviceName)
	imageObjName := m.objectName(fmt.Sprintf("%s/%s/%s/%s", imagesDirStoragePath, deviceName, imageUUID, fileName))
	imageObj := m.gcsBucket.Object(imageObjName)
	return m.fullObjectPath(imageObj), imageObj
}

func (m *Manager) cleanObjectNamePathSegment(path string) string {
	replaceRegex := regexp.MustCompile(`([\[\]*?:"<>|/#\s]|_)+`)
	return replaceRegex.ReplaceAllString(path, "_")
}

func (m *Manager) InitializeNewWifiRouterConfig(ctx context.Context, forceOverwriteExisting bool) error {
	if !forceOverwriteExisting {
		configObjectPath, _ := m.ConfigObject()
		if err := m.FetchConfig(ctx); err != nil && err != storage.ErrObjectNotExist {
			return fmt.Errorf("failed to check if a wifi router config already exists in GCS at %q: %w", configObjectPath, err)
		}
		if m.wifiRouterConfig != nil {
			return fmt.Errorf("a wifi router config already exists in GCS at %q and forceOverwriteExisting is false", configObjectPath)
		}
	}
	m.wifiRouterConfig = &labapi.WifiRouterConfig{}
	return m.saveConfig(ctx)
}

func (m *Manager) ConfigObject() (string, *storage.ObjectHandle) {
	var configStoragePath string
	if m.useProdConfig {
		configStoragePath = configStoragePathProd
	} else {
		configStoragePath = configStoragePathTest
	}
	configObj := m.gcsBucket.Object(m.objectName(configStoragePath))
	return m.fullObjectPath(configObj), configObj
}

func (m *Manager) fullObjectPath(obj *storage.ObjectHandle) string {
	return fmt.Sprintf("gs://%s/%s", obj.BucketName(), obj.ObjectName())
}

func (m *Manager) objectName(relativePath string) string {
	return fmt.Sprintf("%s/%s", baseStorageDir, relativePath)
}

func (m *Manager) localObject(fullObjectPath string) (*storage.ObjectHandle, error) {
	baseStorageDirPath := m.fullObjectPath(m.gcsBucket.Object(m.objectName("")))
	if fullObjectPath == baseStorageDirPath {
		return nil, fmt.Errorf("invalid object path %q", fullObjectPath)
	}
	if !strings.HasPrefix(fullObjectPath, baseStorageDirPath) {
		return nil, fmt.Errorf("object path %q does not reside in the project storage dir %q", fullObjectPath, baseStorageDirPath)
	}
	relativePath := strings.TrimPrefix(fullObjectPath, baseStorageDirPath)
	return m.gcsBucket.Object(m.objectName(relativePath)), nil
}

func (m *Manager) FetchConfig(ctx context.Context) error {
	configObjectPath, configObj := m.ConfigObject()
	reader, err := configObj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return err
		}
		return fmt.Errorf("failed to open config object %q from GCS: %w", configObjectPath, err)
	}
	configJson, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read config object %q from GCS: %w", configObjectPath, err)
	}
	if err := reader.Close(); err != nil {
		return fmt.Errorf("failed to read config object %q from GCS: %w", configObjectPath, err)
	}
	config := &labapi.WifiRouterConfig{}
	if err := protojson.Unmarshal(configJson, config); err != nil {
		return fmt.Errorf("failed to unmarshal config object %q from GCS as a WifiRouterConfig: %w", configObjectPath, err)
	}
	log.Logger.Printf("Successfully retrieved WifiRouterConfig from %q\n", configObjectPath)
	m.wifiRouterConfig = config
	return nil
}

func (m *Manager) saveConfig(ctx context.Context) error {
	configObjectPath, configObj := m.ConfigObject()
	configJson, err := fileutils.MarshalPrettyJSON(m.wifiRouterConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	writer := configObj.NewWriter(ctx)
	if _, err := writer.Write(configJson); err != nil {
		return fmt.Errorf("failed to upload new config object %q to GCS: %w", configObjectPath, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to upload new config object %q to GCS: %w", configObjectPath, err)
	}
	log.Logger.Printf("Saved new WifiRouterConfig to %q:\n%s\n", configObjectPath, configJson)
	return nil
}

func (m *Manager) OpenWrtConfig() *labapi.WifiRouterConfig {
	if m.wifiRouterConfig.Openwrt == nil {
		m.wifiRouterConfig.Openwrt = make(map[string]*labapi.OpenWrtWifiRouterDeviceConfig)
	}
	openWrtConfig := &labapi.WifiRouterConfig{
		Openwrt: m.wifiRouterConfig.Openwrt,
	}
	return openWrtConfig
}

func (m *Manager) UpdateOpenWrtConfig(ctx context.Context, config *labapi.WifiRouterConfig, forceUpdate bool) (*labapi.WifiRouterConfig, error) {
	if err := m.validateOpenWrtConfig(config); err != nil {
		err = fmt.Errorf("invalid OpenWrt config: %w", err)
		if forceUpdate {
			log.Logger.Printf("OpenWrt config validation failed, but continuing anyways as forceUpdate is true: %v\n", err)
		} else {
			return nil, err
		}
	}
	m.wifiRouterConfig.Openwrt = config.Openwrt
	if err := m.saveConfig(ctx); err != nil {
		return nil, fmt.Errorf("failed to update OpenWrt config in storage: %w", err)
	}
	return m.OpenWrtConfig(), nil
}

func (m *Manager) validateOpenWrtConfig(config *labapi.WifiRouterConfig) error {
	if config.Openwrt == nil {
		return fmt.Errorf("config.OpenWrt is nil")
	}
	for deviceName, deviceConfig := range config.Openwrt {
		if len(deviceConfig.Images) == 0 {
			return fmt.Errorf("config.OpenWrt[%q].Images is empty", deviceName)
		}
		if deviceConfig.CurrentImageUuid == "" {
			return fmt.Errorf("config.OpenWrt[%q].CurrentImageUuid is empty", deviceName)
		}
		imageUUIDs := make(map[string]bool)
		imageMinVersions := make(map[string]bool)
		for i, imageConfig := range deviceConfig.Images {
			if imageConfig.ImageUuid == "" {
				return fmt.Errorf("config.OpenWrt[%q].Images[%d].ImageUuid is empty", deviceName, i)
			}
			if imageConfig.ArchivePath == "" {
				return fmt.Errorf("config.OpenWrt[%q].Images[%d].ArchivePath is empty", deviceName, i)
			}
			if imageConfig.MinDutReleaseVersion == "" {
				return fmt.Errorf("config.OpenWrt[%q].Images[%d].MinDutReleaseVersion is empty", deviceName, i)
			}
			if _, ok := imageUUIDs[imageConfig.ImageUuid]; ok {
				return fmt.Errorf("config.OpenWrt[%q].Images has multiple images with the same ImageUuid %q", deviceName, imageConfig.ImageUuid)
			}
			imageUUIDs[imageConfig.ImageUuid] = true
			if _, ok := imageMinVersions[imageConfig.MinDutReleaseVersion]; ok {
				return fmt.Errorf("config.OpenWrt[%q].Images has multiple images with the same MinDutReleaseVersion %q", deviceName, imageConfig.MinDutReleaseVersion)
			}
			imageMinVersions[imageConfig.MinDutReleaseVersion] = true
			if _, err := m.localObject(imageConfig.ArchivePath); err != nil {
				return fmt.Errorf("config.OpenWrt[%q].Images[%d].ArchivePath is invalid: %w", deviceName, i, err)
			}
		}
		if _, ok := imageUUIDs[deviceConfig.CurrentImageUuid]; !ok {
			return fmt.Errorf("config.OpenWrt[%q].CurrentImageUuid (%q) does not match any image in config.OpenWrt[%q].Images", deviceName, deviceConfig.CurrentImageUuid, deviceName)
		}
		if deviceConfig.NextImageUuid != "" {
			if _, ok := imageUUIDs[deviceConfig.NextImageUuid]; !ok {
				return fmt.Errorf("config.OpenWrt[%q].NextImageUuid (%q) does not match any image in config.OpenWrt[%q].Images", deviceName, deviceConfig.NextImageUuid, deviceName)
			}
		}
	}
	return nil
}

func (m *Manager) OpenWrtRouterDeviceConfig(deviceName string) (*labapi.OpenWrtWifiRouterDeviceConfig, error) {
	openWrtConfig := m.OpenWrtConfig()
	if len(openWrtConfig.Openwrt) == 0 {
		return nil, errors.New("no devices are configured")
	}
	deviceConfig, ok := openWrtConfig.Openwrt[deviceName]
	if !ok {
		var allDeviceNames []string
		for name, _ := range openWrtConfig.Openwrt {
			allDeviceNames = append(allDeviceNames, fmt.Sprintf("%q", name))
		}
		return nil, fmt.Errorf("no config found for device %q; configured devices: %s", deviceName, strings.Join(allDeviceNames, ", "))
	}
	return deviceConfig, nil
}

func (m *Manager) UploadImageArchive(ctx context.Context, archiveFilePath, deviceName, imageUuid, minDutReleaseVersion string) (*labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, error) {
	imageObjPath, imageObj := m.imageFileObject(deviceName, imageUuid, filepath.Base(archiveFilePath))
	log.Logger.Printf("Uploading image archive from %q to %q", archiveFilePath, imageObjPath)
	inFile, err := os.Open(archiveFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file at %q: %w", archiveFilePath, err)
	}
	inFileReader := fileutils.NewContextualReaderWrapper(ctx, inFile)
	storageWriter := imageObj.NewWriter(ctx)
	bytesWritten, err := io.Copy(storageWriter, inFileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to upload new image file to %q: %w", imageObjPath, err)
	}
	if err := storageWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to upload new image file to %q: %w", imageObjPath, err)
	}
	_ = inFile.Close()
	log.Logger.Printf("Successfully uploaded %d bytes to %q", bytesWritten, imageObjPath)
	imageConfig := &labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage{
		ImageUuid:            imageUuid,
		MinDutReleaseVersion: minDutReleaseVersion,
		ArchivePath:          imageObjPath,
	}
	return imageConfig, nil
}

func (m *Manager) DownloadImageArchive(ctx context.Context, imageConfig *labapi.OpenWrtWifiRouterDeviceConfig_OpenWrtOSImage, dstDir string) (string, error) {
	dstFilePath := filepath.Join(dstDir, filepath.Base(imageConfig.ArchivePath))
	imageArchiveObj, err := m.localObject(imageConfig.ArchivePath)
	if err != nil {
		return "", fmt.Errorf("invalid image archive path %q: %w", imageConfig.ArchivePath, err)
	}
	log.Logger.Printf("Downloading image archive %q\n", imageConfig.ArchivePath)
	if err := m.downloadObjectToFile(ctx, imageArchiveObj, dstFilePath); err != nil {
		return "", fmt.Errorf("failed to download image archive %q to %q", imageConfig.ArchivePath, dstFilePath)
	}
	absDstFilePath, err := filepath.Abs(dstFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute file path of installed image relative path %q: %w", dstFilePath, err)
	}
	log.Logger.Printf("Successfully downloaded image to %q\n", absDstFilePath)
	return absDstFilePath, nil
}

func (m *Manager) downloadObjectToFile(ctx context.Context, obj *storage.ObjectHandle, dstFilePath string) error {
	dstFileDir := path.Dir(dstFilePath)
	if dstFileDir != "" {
		if err := os.MkdirAll(dstFileDir, fileutils.DefaultDirPermissions); err != nil {
			return fmt.Errorf("failed to make dirs for file %q: %w", dstFilePath, err)
		}
	}
	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", dstFilePath, err)
	}
	outFileWriter := fileutils.NewContextualWriterWrapper(ctx, dstFile)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return err
		}
		return fmt.Errorf("failed to open GCS object: %w", err)
	}
	if _, err := io.Copy(outFileWriter, reader); err != nil {
		return fmt.Errorf("failed to download GCS object: %w", err)
	}
	if err := reader.Close(); err != nil {
		return fmt.Errorf("failed to download GCS object: %w", err)
	}
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", dstFilePath, err)
	}
	return nil
}
