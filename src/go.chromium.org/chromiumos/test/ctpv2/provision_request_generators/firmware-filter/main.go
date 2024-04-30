// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"

	"go.chromium.org/chromiumos/config/go/test/api"
	server "go.chromium.org/chromiumos/test/ctpv2/common/server_template"
)

// FirmwareSpecs contains the flags necessary for the
// filter to know how to build out the provision request
// and populate the lookup table.
type FirmwareSpecs struct {
	Ro string
	Rw string
}

func (specs *FirmwareSpecs) executor(req *api.InternalTestplan, log *log.Logger) (*api.InternalTestplan, error) {
	log.Println("Executing firmware filter")

	if err := GenerateDynamicInfo(req, specs); err != nil {
		log.Printf("Error while generating dynamic info, %s", err)
		return req, err
	}
	log.Println("Finished generating dynamic info")

	return req, nil
}

func main() {
	firmwareSpecs := &FirmwareSpecs{}
	fs := flag.NewFlagSet("Run firmware-provision-filter", flag.ExitOnError)
	fs.StringVar(&firmwareSpecs.Ro, "ro", "", "Spec for firmare RO")
	fs.StringVar(&firmwareSpecs.Rw, "rw", "", "Spec for firmare RW")

	err := server.ServerWithFlagSet(fs, firmwareSpecs.executor, "fw_filter")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
