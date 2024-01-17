// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package firmwareservice

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"chromiumos/test/provision/lib/servo_lib"
	"chromiumos/test/provision/lib/servoadapter"
	common_utils "chromiumos/test/provision/v2/common-utils"

	"github.com/pkg/errors"
	conf "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	"gopkg.in/yaml.v3"
)

// FirmwareVersions holds the ro and rw versions read from a firmware binary.
type FirmwareVersions struct {
	AP struct {
		Versions struct {
			RO string `yaml:"ro"`
			RW string `yaml:"rw"`
		} `yaml:"versions"`
	} `yaml:"host"`
	EC struct {
		Versions struct {
			RO string `yaml:"ro"`
			RW string `yaml:"rw"`
		} `yaml:"versions"`
	} `yaml:"ec"`
}

// FirmwareService implements ServiceInterface
type FirmwareService struct {
	// In case of flashing over SSH, |connection| connects to the DUT.
	// In case of flashing over Servo, |connection| connects to the ServoHost.
	connection common_utils.ServiceAdapterInterface

	// DetailedRequest fields
	mainRwPath *conf.StoragePath
	mainRoPath *conf.StoragePath
	ecRoPath   *conf.StoragePath

	board, model string

	force bool

	ecChip string

	// imagesMetadata is a map from gspath -> ImageArchiveMetadata.
	// Allows to avoid redownloading/reprocessing archives.
	imagesMetadata map[string]ImageArchiveMetadata

	useServo bool
	// servoConfig provides dut-controls and programmer argument for flashing
	servoConfig *servo_lib.ServoConfig
	// wrapper around servoClient connection with extra servod-related functions
	servoConnection servoadapter.ServoHostInterface
	servoPort       int

	CacheServer      url.URL
	ExpectedVersions FirmwareVersions
}

type versionJSON struct {
	Default FirmwareVersions `yaml:"default"`
}

// NewFirmwareService initializes a FirmwareService.
func NewFirmwareService(ctx context.Context, dutAdapter common_utils.ServiceAdapterInterface,
	servoClient api.ServodServiceClient, cacheServer url.URL, board, model string,
	useServo bool, req *api.InstallRequest) (*FirmwareService, error) {
	metadata := new(api.FirmwareProvisionInstallMetadata)
	if req.GetMetadata().MessageIs(metadata) {
		if err := req.GetMetadata().UnmarshalTo(metadata); err != nil {
			return nil, InvalidRequestErr(errors.Wrap(err, "unmarshalling metadata"))
		}
	}
	detailedRequest := metadata.FirmwareConfig

	fws := FirmwareService{
		connection:     dutAdapter,
		board:          board,
		model:          model,
		force:          false,
		useServo:       useServo,
		imagesMetadata: make(map[string]ImageArchiveMetadata),
		CacheServer:    cacheServer,
	}

	// Firmware may be updated in write-protected mode, where only 'rw' regions
	// would be update, or write-protection may be disabled (dangerous) in order
	// to update 'ro' regions.
	//
	// The only 'rw' firmware is the main one, aka AP firmware, which also syncs
	// EC 'rw'. It will be flashed with write protection turned on.
	fws.mainRwPath = detailedRequest.MainRwPayload.GetFirmwareImagePath()

	// Read-only firmware images include AP, EC, and PD firmware, and will be
	// flashed with write protection turned off.
	fws.mainRoPath = detailedRequest.MainRoPayload.GetFirmwareImagePath()
	fws.ecRoPath = detailedRequest.EcRoPayload.GetFirmwareImagePath()

	if useServo {
		fws.prepareServoConnection(ctx, servoClient)
	}

	fws.PrintRequestInfo()

	if !fws.UpdateRo() && !fws.UpdateRw() {
		return nil, InvalidRequestErr(errors.New("no paths to images specified"))
	}

	return &fws, nil
}

// Confirms that cros-servod connection is functional, and fills the following
// fields in FirmwareService:
//   - servoConfig
//   - servoConnection
//   - ecChip
//   - servoPort
func (fws *FirmwareService) prepareServoConnection(ctx context.Context, servoClient api.ServodServiceClient) error {
	if servoClient == nil {
		return InvalidRequestErr(errors.New("servo use is requested, but servo client not provided"))
	}

	// Note: dut.GetChromeos().Servo.ServodAddress.Port is the port of cros-servod
	// service, we seem to be missing port of servod itself.
	// Always use default servod port 9999 for now.
	if fws.servoPort == 0 {
		fws.servoPort = 9999
	}

	fws.servoConnection = servoadapter.NewServoHostAdapterFromExecCmder(fws.board,
		fws.model,
		fws.servoPort,
		servoClient)

	// Ask servod for servo_type. This implicitly checks if servod is running
	// and servo connection is working.
	servoTypeStr, err := fws.servoConnection.GetVariable(ctx, "servo_type")
	if err != nil {
		return UnreachablePreProvisionErr(errors.Errorf("failed to get servo_type: %w. "+
			"Is servod running on port %v and connected to the DUT?",
			err, fws.servoPort)) // TODO(sfrolov): add UnreachableCrosServodErr
	}

	// ask servod for serial number of the connected DUT.
	servoType := servo_lib.NewServoType(servoTypeStr)

	if servoType.IsMultipleServos() {
		// Handle dual servo.
		// We need CCD if ec_ro is set, otherwise, servo_micro will be faster.
		preferCCD := false
		if fws.ecRoPath != nil {
			preferCCD = true
		}
		servoSubtypeStr := servoType.PickServoSubtype(preferCCD)
		servoType = servo_lib.NewServoType(servoSubtypeStr)
	}

	serial, err := fws.servoConnection.GetVariable(ctx, servoType.GetSerialNumberOption())
	if err != nil {
		return UnreachablePreProvisionErr(errors.Errorf("failed to get serial number variable: %w", err.Error())) // TODO: cros-servod is unreachable
	}

	// finally, get the correct config
	fws.servoConfig, err = servo_lib.GetServoConfig(fws.board, serial, servoType)
	if err != nil {
		return UnreachablePreProvisionErr(errors.Errorf("failed to get servo config: %w", err.Error())) // TODO: cros-servod is unreachable
	}

	fws.ecChip, err = fws.servoConnection.GetVariable(ctx, "ec_chip")
	if err != nil {
		return UnreachablePreProvisionErr(errors.Errorf("failed to get ec_chip variable: %w", err.Error())) // TODO: cros-servod is unreachable
	}
	return nil
}

// PrintRequestInfo logs details of the provisioning operation.
func (fws *FirmwareService) PrintRequestInfo() {
	informationString := "provisioning "

	images := []string{}
	if fws.mainRwPath != nil {
		images = append(images, "AP(RW)")
	}
	if fws.mainRoPath != nil {
		images = append(images, "AP(RO)")
	}
	if fws.ecRoPath != nil {
		images = append(images, "EC(RO)")
	}
	informationString += strings.Join(images, " and ") + " firmware"

	flashMode := "SSH"
	if fws.useServo {
		flashMode = fmt.Sprintf("Servo %s", (fws.servoConfig.ServoType))
	}
	informationString += " over " + flashMode + ". "

	informationString += "Board: " + fws.board + ". "

	informationString += "Model: " + fws.model + ". "

	informationString += fmt.Sprintf("Force: %v.", fws.force)

	log.Println("[FW Provisioning]", informationString)
}

// UpdateRw returns whether the read/write firmware is being operated on.
func (fws *FirmwareService) UpdateRw() bool {
	return fws.mainRwPath != nil
}

// UpdateRo returns whether the read-only firmware is being operated on.
func (fws *FirmwareService) UpdateRo() bool {
	return (fws.mainRoPath != nil) || (fws.ecRoPath != nil)
}

// GetBoard returns board of the DUT to provision. Returns empty string if board is not known.
func (fws *FirmwareService) GetBoard() string {
	return fws.board
}

// RestartDut restarts the DUT using one of the available mechanisms.
// Preferred restart method is to send "power_state:reset" command to servod.
// If servod restarting failed/not available in the environment,
// then this function will try to SSH to the DUT and run `restart`,
// if |requireServoReset| is false. If |requireServoReset| is True, restart over
// SSH will not be attempted, and the function will return an error.
func (fws *FirmwareService) RestartDut(ctx context.Context, requireServoReset bool) error {
	waitForReconnect := func(conn common_utils.ServiceAdapterInterface) error {
		// Attempts to run `true` on the DUT over SSH until |reconnectRetries| attempts.
		const reconnectRetries = 10
		const reconnectAttemptWait = 10 * time.Second
		const reconnectFailPause = 10 * time.Second
		var restartErr error
		for i := 0; i < reconnectRetries; i++ {
			reconnectCtx, reconnCancel := context.WithTimeout(ctx, reconnectAttemptWait)
			defer reconnCancel()
			_, restartErr = conn.RunCmd(reconnectCtx, "true", nil)
			if restartErr == nil {
				log.Println("[FW Provisioning: Restart DUT] reestablished connection to the DUT after restart.")
				return nil
			}
			time.Sleep(reconnectFailPause)
		}
		log.Printf("[FW Provisioning: Restart DUT] timed out waiting for DUT to restart: %v\n", restartErr)
		return restartErr
	}

	if requireServoReset && fws.servoConnection == nil {
		return errors.New("servo restart is required but servo connection not available")
	}
	// over Servo first
	if fws.servoConnection != nil {
		log.Printf("[FW Provisioning: Restart DUT] restarting DUT with \"dut-control power_state:reset\" over servo.\n")
		servoRestartErr := fws.servoConnection.RunDutControl(ctx, []string{"power_state:reset"})
		if servoRestartErr == nil {
			if fws.connection != nil {
				// If SSH connection to DUT was available, wait until it's back up again.
				return waitForReconnect(fws.connection)
			}
			waitDuration := 30 * time.Second
			log.Printf("[FW Provisioning: Restart DUT] waiting for %v for DUT to finish rebooting.\n", waitDuration.String())
			time.Sleep(waitDuration)
			powerState, getPowerStateErr := fws.servoConnection.GetVariable(ctx, "ec_system_powerstate")
			if getPowerStateErr != nil {
				log.Printf("[FW Provisioning: Restart DUT] failed to get power state after reboot: %v\n", getPowerStateErr)
			} else {
				log.Printf("[FW Provisioning: Restart DUT] DUT power state after reboot: %v\n", powerState)
			}
			return getPowerStateErr
		}
		log.Printf("[FW Provisioning: Restart DUT] failed to restart DUT via Servo: %v.\n", servoRestartErr)
		if requireServoReset {
			return servoRestartErr
		}
	}

	// over SSH if allowed by |requireServoReset|
	if fws.connection != nil {
		log.Printf("[FW Provisioning: Restart DUT] restarting DUT over SSH.")
		fws.connection.Restart(ctx)
		return waitForReconnect(fws.connection)
	}
	return errors.New("failed to restart: no SSH connection to the DUT")
}

// GetModel returns model of the DUT to provision. Returns empty string if model is not known.
func (fws *FirmwareService) GetModel() string {
	return fws.model
}

// CleanupOnFailure is called if one of service's states failes to Execute() and
// currenly cleans up the temporary files.
func (fws *FirmwareService) CleanupOnFailure(states []common_utils.ServiceState, executionErr error) error {
	return fws.DeleteArchiveDirectories()
}

// DeleteArchiveDirectories deletes files on the servo host or DUT.
func (fws *FirmwareService) DeleteArchiveDirectories() error {
	var cleanedDevice common_utils.ServiceAdapterInterface
	if fws.useServo {
		// If servo is used, the files will be located on the ServoHost.
		cleanedDevice = fws.servoConnection
	} else {
		// If SSH is used, the files will be located on the DUT in /tmp/
		// It's not strictly necessary to delete them before reboot.
		cleanedDevice = fws.connection
	}

	var allErrors []string
	for _, imgMetadata := range fws.imagesMetadata {
		err := cleanedDevice.DeleteDirectory(context.Background(), imgMetadata.ArchiveDir)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("failed to delete %v: %v", imgMetadata.ArchiveDir, err))
		}
	}

	if len(allErrors) > 0 {
		return errors.New(strings.Join(allErrors, ". "))
	}

	return nil
}

// FlashWithFutility flashes the DUT using "futility" tool.
// futility will be run with "--mode=recovery".
// if |rwOnly| is true, futility will flash only RW regions.
// if |rwOnly| is false, futility will flash both RW and RO regions.
// futilityArgs must include argument(s) that provide path(s) to the images.
//
// If flashing over ssh, simply calls runFutility().
// If flashing over servo, also runs pre- and post-flashing dut-controls.
func (fws *FirmwareService) FlashWithFutility(ctx context.Context, rwOnly bool, futilityImageArgs []string) error {
	if err := fws.ExtractFirmwareVersions(ctx, rwOnly, futilityImageArgs); err != nil {
		return errors.Wrap(err, "extract versions")
	}
	if fws.useServo {
		return fws.servoFlash(ctx, rwOnly, futilityImageArgs)
	}
	return fws.sshFlash(ctx, rwOnly, futilityImageArgs)
}

// ExtractFirmwareVersions uses futility to get the version numbers from the images.
func (fws *FirmwareService) ExtractFirmwareVersions(ctx context.Context, rwOnly bool, futilityImageArgs []string) error {
	if len(futilityImageArgs) == 0 {
		return errors.New("unable to gte version: no futility Image args provided")
	}

	futilityArgs := futilityImageArgs
	futilityArgs = append([]string{"update", "--manifest"}, futilityArgs...)

	connection := fws.GetConnectionToFlashingDevice()
	out, err := connection.RunCmd(ctx, "futility", futilityArgs)
	if err != nil {
		return errors.Wrap(err, "failed to exec futility")
	}
	versions := versionJSON{}
	err = yaml.Unmarshal([]byte(out), &versions)
	if err != nil {
		return errors.Wrap(err, "failed to parse futility manifest")
	}

	if !rwOnly && versions.Default.AP.Versions.RO != "" {
		fws.ExpectedVersions.AP.Versions.RO = versions.Default.AP.Versions.RO
	}
	if versions.Default.AP.Versions.RW != "" {
		fws.ExpectedVersions.AP.Versions.RW = versions.Default.AP.Versions.RW
	}
	if !rwOnly && versions.Default.EC.Versions.RO != "" {
		fws.ExpectedVersions.EC.Versions.RO = versions.Default.EC.Versions.RO
	}
	if versions.Default.EC.Versions.RW != "" {
		fws.ExpectedVersions.EC.Versions.RW = versions.Default.EC.Versions.RW
	}
	return nil
}

// ActiveFirmwareVersions returns the firmware versions currently running on the DUT.
func (fws *FirmwareService) ActiveFirmwareVersions(ctx context.Context) (*FirmwareVersions, error) {
	var versions FirmwareVersions
	// Run crossystem ro_fwid fwid to get AP versions
	out, err := fws.connection.RunCmd(ctx, "crossystem", []string{"ro_fwid", "fwid"})
	if err != nil {
		return nil, errors.Wrap(err, "crossystem failed")
	}
	apVers := strings.SplitN(out, " ", 2)
	if len(apVers) != 2 {
		return nil, errors.Errorf("incorrect crossystem output: %q", out)
	}
	versions.AP.Versions.RO = apVers[0]
	versions.AP.Versions.RW = apVers[1]

	// Run ectool version to get EC versions
	out, err = fws.connection.RunCmd(ctx, "ectool", []string{"version"})
	if err != nil {
		// Wilco, etc.
		if strings.Contains(out, "Very likely this board doesn't have a Chromium EC") {
			return &versions, nil
		}
		return nil, errors.Wrap(err, "ectool failed")
	}
	roRE, err := regexp.Compile(`RO version:\s*(\S+)\s*\n`)
	if err != nil {
		return nil, errors.Wrap(err, "invalid roRE")
	}
	rwRE, err := regexp.Compile(`RW version:\s*(\S+)\s*\n`)
	if err != nil {
		return nil, errors.Wrap(err, "invalid rwRE")
	}
	if m := roRE.FindStringSubmatch(out); m != nil {
		versions.EC.Versions.RO = m[1]
	} else {
		return nil, errors.Errorf("EC RO version not found in %q", out)
	}
	if m := rwRE.FindStringSubmatch(out); m != nil {
		versions.EC.Versions.RW = m[1]
	} else {
		return nil, errors.Errorf("EC RW version not found in %q", out)
	}
	return &versions, nil
}

func (fws *FirmwareService) sshFlash(ctx context.Context, rwOnly bool, futilityImageArgs []string) error {
	return fws.runFutility(ctx, rwOnly, futilityImageArgs)
}

func (fws *FirmwareService) servoFlash(ctx context.Context, rwOnly bool, futilityImageArgs []string) error {
	dutOnErr := fws.servoConnection.RunAllDutControls(ctx, fws.servoConfig.DutOn)
	if dutOnErr != nil {
		return errors.Errorf("failed to run pre-flashing dut-controls: %w", dutOnErr)
	}

	flashErr := fws.runFutility(ctx, rwOnly, futilityImageArgs)

	// Attempt to run post-flashing dut-controls regardless of flashErr.
	dutOffErr := fws.servoConnection.RunAllDutControls(ctx, fws.servoConfig.DutOff)
	if flashErr != nil {
		return flashErr
	}
	if dutOffErr != nil {
		return errors.Errorf("failed to run post-flashing dut-controls: %w", dutOffErr)
	}
	return nil
}

func (fws *FirmwareService) runFutility(ctx context.Context, rwOnly bool, futilityImageArgs []string) error {
	if len(futilityImageArgs) == 0 {
		return errors.New("unable to flash: no futility Image args provided")
	}

	futilityArgs := futilityImageArgs
	futilityArgs = append([]string{"update", "--mode=recovery"}, futilityArgs...)

	if rwOnly {
		futilityArgs = append(futilityArgs, "--wp=1")
	} else {
		futilityArgs = append(futilityArgs, "--wp=0")
	}

	if fws.force {
		futilityArgs = append(futilityArgs, "--force")
	}

	if fws.useServo {
		futilityArgs = append(futilityArgs, "-p", fws.servoConfig.Programmer)
		futilityArgs = append(futilityArgs, fws.servoConfig.ExtraArgs...)
		// TODO(sfrolov): extra args from fw-config.json
	}

	connection := fws.GetConnectionToFlashingDevice()
	if _, err := connection.RunCmd(ctx, "futility", futilityArgs); err != nil {
		return err
	}
	return nil
}

// GetConnectionToFlashingDevice returns connection to the device that stores the
// firmware image and runs futility.
// Returns connection to ServoHost if fws.useServo, connection to DUT otherwise.
func (fws *FirmwareService) GetConnectionToFlashingDevice() common_utils.ServiceAdapterInterface {
	if fws.useServo {
		return fws.servoConnection
	}
	return fws.connection
}

// IsServoUsed returns whether servo is used during the provisioning.
func (fws *FirmwareService) IsServoUsed() bool {
	return fws.useServo
}

// IsForceUpdate returns whether an update is forced during the provisioning.
func (fws *FirmwareService) IsForceUpdate() bool {
	return fws.force
}

// GetMainRwPath returns the path of the read/write part of the AP firmware.
func (fws *FirmwareService) GetMainRwPath() string {
	return fws.mainRwPath.GetPath()
}

// GetMainRoPath returns the path of readonly part of the AP firmware.
func (fws *FirmwareService) GetMainRoPath() string {
	return fws.mainRoPath.GetPath()
}

// GetEcRoPath returns the path for the readonly portion of the EC firmware.
func (fws *FirmwareService) GetEcRoPath() string {
	return fws.ecRoPath.GetPath()
}

type configData struct {
	ChromeOS struct {
		Configs []struct {
			Firmware struct {
				BuildTargets struct {
					// AP image name, if missing fallback to ImageName
					Coreboot string `yaml:"coreboot"`
					EC       string `yaml:"ec"`
					ZephyrEC string `yaml:"zephyr-ec"`
				} `yaml:"build-targets"`
				ImageName string `yaml:"image-name"`
			} `yaml:"firmware"`
			Identity struct {
				PlatformName    string `yaml:"platform-ec"`
				SKUID           int    `yaml:"sku-id"`
				SmbiosNameMatch string `yaml:"smbios-name-match"`
			}
			Name string `yaml:"name"`
		}
	}
}

// DownloadAndProcess downloads the provided archive.
// If DetailedRequest is used, DownloadAndProcess also extracts the archive and
// enumerates its contents, storing the metadata in s.service.archiveFolders.
func (fws *FirmwareService) DownloadAndProcess(ctx context.Context, gsPath string) error {
	connection := fws.GetConnectionToFlashingDevice()
	if _, alreadyDownloaded := fws.imagesMetadata[gsPath]; !alreadyDownloaded {
		if err := stageFile(ctx, gsPath, fws.CacheServer); err != nil {
			return errors.Wrap(err, "failed to stage")
		}
		apImageName := ""
		ecImageName := ""
		config, err := extractFile(ctx, gsPath, "config.yaml", fws.CacheServer)
		if err != nil {
			log.Printf("No config.yaml in %q: %v", gsPath, err)
		} else {
			defer config.Close()
			configYaml := configData{}
			yamlDecoder := yaml.NewDecoder(config)
			err = yamlDecoder.Decode(&configYaml)
			if err != nil {
				return errors.Wrap(err, "failed to parse config.yaml")
			}
			model := fws.GetModel()
			for _, config := range configYaml.ChromeOS.Configs {
				if config.Name == model {
					thisAPName := config.Firmware.BuildTargets.Coreboot
					if thisAPName == "" {
						thisAPName = config.Firmware.ImageName
					}
					if thisAPName != "" {
						if apImageName != "" && apImageName != thisAPName {
							return errors.Errorf("ambiguous AP name for model %q, could be %q or %q", model, apImageName, thisAPName)
						}
						apImageName = thisAPName
					}
					thisECName := config.Firmware.BuildTargets.ZephyrEC
					if thisECName == "" {
						thisECName = config.Firmware.BuildTargets.EC
					}
					if thisECName != "" {
						if ecImageName != "" && ecImageName != thisECName {
							return errors.Errorf("ambiguous EC name for model %q, could be %q or %q", model, ecImageName, thisECName)
						}
						ecImageName = thisECName
					}
				}
			}
			log.Printf("config.yaml image names AP: %s EC: %s", apImageName, ecImageName)
		}

		// Infer names for the local files and folders from basename of gsPath.
		archiveFilename := filepath.Base(gsPath)

		// Try to get a descriptive name for the temporary folder.
		archiveSubfolder := archiveFilename
		if strings.HasPrefix(archiveSubfolder, "firmware_from_source") {
			splitGsPath := strings.Split(gsPath, "/")
			nameIdx := len(splitGsPath) - 2
			if nameIdx < 0 {
				nameIdx = 0
			}
			archiveSubfolder = splitGsPath[nameIdx]
		}

		// Use mktemp to safely create a unique temp directory.
		archiveDir, err := connection.RunCmd(ctx, "mktemp", []string{"-d", "--tmpdir", "cros-fw-provision.XXXXXXXXX." + archiveSubfolder})
		if err != nil {
			return errors.Errorf("remote mktemp failed: %w", err)
		}
		archiveDir = strings.Trim(archiveDir, "\n")

		fws.imagesMetadata[gsPath] = ImageArchiveMetadata{
			CorebootName: apImageName,
			ECName:       ecImageName,
			ArchiveDir:   archiveDir,
		}
	}
	return nil
}

// GetImageMetadata returns (ImageArchiveMetadata, IsImageMetadataPresent)
func (fws *FirmwareService) GetImageMetadata(gspath string) (ImageArchiveMetadata, bool) {
	metadata, ok := fws.imagesMetadata[gspath]
	return metadata, ok
}

// ProvisionWithFlashEC flashes EC image using flash_ec script.
func (fws *FirmwareService) ProvisionWithFlashEC(ctx context.Context, ecImage, flashECScriptPath string) error {
	customBitbangRate := ""
	if fws.ecChip == "stm32" {
		customBitbangRate = "--bitbang_rate=57600"
	}
	flashCmdArgs := fmt.Sprintf("--ro --chip=%s --board=%s --image=%s --port=%v %s --verify --verbose",
		fws.ecChip, fws.model, ecImage, fws.servoPort, customBitbangRate)
	output, err := fws.GetConnectionToFlashingDevice().RunCmd(ctx, flashECScriptPath, strings.Split(flashCmdArgs, " "))
	if err != nil {
		log.Println(output)
		return err
	}
	return nil
}
