// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package adb

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"go.chromium.org/luci/common/errors"
)

type execer struct {
	logger *log.Logger
}

// New creates new Execer.
func New(logger *log.Logger) (*execer, error) {
	if logger == nil {
		return nil, errors.Reason("logger is not provided").Err()
	}
	return &execer{
		logger: logger,
	}, nil
}

type result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int32
}

// Exec run blocking command by ADB.
// Do not use non-blocking command (like: logcat).
func (e *execer) Exec(ctx context.Context, args ...string) (*result, error) {
	if len(args) == 0 {
		return nil, errors.Reason("exec: command is not provided").Err()
	}
	fullCommand := fmt.Sprintf("adb %s", strings.Join(args, " "))
	e.print("Running command", fullCommand)
	cmd := exec.CommandContext(ctx, "adb", args...)
	var outB bytes.Buffer
	var errB bytes.Buffer
	cmd.Stdout = &outB
	cmd.Stderr = &errB
	// We always convert error to exit code.
	// Return an error only if issue in code.
	err := cmd.Run()
	res := &result{
		Stdout:   outB.Bytes(),
		Stderr:   errB.Bytes(),
		ExitCode: 0,
	}
	if err != nil {
		res.ExitCode = 1
		res.Stderr = []byte(err.Error())
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = int32(exitErr.ExitCode())
		}
	}
	e.print("stdout: ", string(res.Stdout))
	e.print("stderr: ", string(res.Stderr))
	e.print("exit-code: ", res.ExitCode)
	return res, nil
}

func (e *execer) print(v ...any) {
	if len(v) == 0 {
		return
	}
	if e.logger != nil {
		e.logger.Println(v...)
	} else {
		log.Println(v...)
	}
}
