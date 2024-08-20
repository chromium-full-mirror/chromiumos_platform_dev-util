// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package adb

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func KillServer(logger *log.Logger) {
	outStr, _ := AdbCmd([]string{"kill-server"}, logger)
	logger.Println(outStr)
	outStr, _ = AdbCmd([]string{"devices"}, logger)
	logger.Println(outStr)
}

func TeardownAdb(logger *log.Logger, addr string) error {
	if out, err := AdbCmd([]string{"-s", FmtAddr(addr), "disconnect"}, logger); err != nil {
		return fmt.Errorf("ADB disconnect failed. %s: %s", err, string(out))
	}
	return nil
}

func AdbConnect(logger *log.Logger, addr string) (err error) {
	// Assumes the ADB port on the CrOS device is visible directly.
	outStr, err := AdbCmd([]string{"connect", FmtAddr(addr)}, logger)
	if err != nil {
		return fmt.Errorf("ADB Start failed. %s: %s", err, outStr)
	}

	if !strings.Contains(outStr, fmt.Sprintf("connected to %s", FmtAddr(addr))) {
		return fmt.Errorf("ADB Start failed.")
	}

	logger.Println("adb connected!:", outStr)

	return err
}

func AdbCmd(args []string, log *log.Logger) (string, error) {
	// Add a 15 second ctx timeout to prevent cmds from hanging.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "adb", args...)
	log.Println("Running ADB SHELL CMD: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("CMD ABOVE DEADLINE EXCEEEDED")
	}
	outStr := string(out)
	log.Println("ADB SHELL OUT: ", outStr)

	return outStr, err
}

func RetrySetupAdb(log *log.Logger, addr string, timeout time.Duration) error {
	st := time.Now()
	for time.Since(st) < timeout {
		err := SetupAdb(log, addr)
		if err == nil {
			log.Println("Device online")

			return nil
		}
	}
	log.Println("Device offline")

	return fmt.Errorf("device not online within timeout")

}

func SetupAdb(logger *log.Logger, addr string) (err error) {
	err = AdbConnect(logger, addr)
	if err != nil {
		logger.Println("setupAdb failed. Trying again: ", err)
		// ADB can be in akward state requiring a retry often.
		// KillServer(logger)
		err = AdbConnect(logger, addr)
		if err != nil {
			logger.Println("setupAdb failed Attempt 2. Exiting: ", err)

			return fmt.Errorf("setupAdb failed")

		}
	}
	found, err := adbDeviceFound(addr, logger)
	if !found {
		return fmt.Errorf("ADB Start failed: %s", err)
	}

	return nil
}

func adbDeviceFound(addr string, logger *log.Logger) (bool, error) {
	outStr, err := AdbCmd([]string{"devices"}, logger)
	deviceRegex := regexp.MustCompile(fmt.Sprintf(`%s\sdevice`, FmtAddr(addr)))
	if !deviceRegex.MatchString(outStr) {

		return false, fmt.Errorf("ADB Devices CMd failed: %s, %s", err, outStr)
	}
	return true, nil
}

func FmtAddr(addr string) string {
	port := "5555"
	return fmt.Sprintf("%s:%s", addr, port)
}

func AdbShellCmd(cmdStr []string, addr string, log *log.Logger) (string, error) {
	args := []string{"-s", FmtAddr(addr), "shell"}
	args = append(args, cmdStr...)
	return AdbCmd(args, log)
}
