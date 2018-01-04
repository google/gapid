// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
)

const (
	ErrMissingPackage   = fault.Const("Missing package name")
	ErrNoDevices        = fault.Const("No devices found")
	ErrNoMatchingDevice = fault.Const("No matching device")
)

var (
	output  = flag.String("out", "", "The output file path")
	serial  = flag.String("device", "", "The serial of the device to pull from")
	skipOBB = flag.Bool("skip-obb", false, "Set this flag to skip trying to pull a matching OBB file from the device")
)

func main() {
	app.ShortHelp = "pullapk pulls an APK from an Android device."
	app.Run(run)
}

func run(ctx context.Context) error {
	pkg := flag.Arg(0)
	if pkg == "" {
		return log.Err(ctx, ErrMissingPackage, "")
	}
	devices, err := adb.Devices(ctx)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return log.Err(ctx, ErrNoDevices, "")
	}

	device := devices[0]
	if *serial != "" {
		device = devices.FindBySerial(*serial)
		if device == nil {
			return log.Errf(ctx, ErrNoMatchingDevice, "serial: %v", *serial)
		}
	}

	pkgs, err := device.InstalledPackages(ctx)
	if err != nil {
		return err
	}

	found, err := pkgs.FindSingleByPartialName(pkg)
	if err != nil {
		return err
	}

	out := *output
	if out == "" { // No output directory specified? Use CWD.
		out, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	if stat, err := os.Stat(out); err == nil {
		if stat.IsDir() { // Output is a directory? Append apk name.
			out = filepath.Join(out, found.Name+".apk")
		}
	}

	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}

	if !*skipOBB && found.OBBExists(ctx) {
		obbOut := filepath.Join(filepath.Dir(out), fmt.Sprintf("main.%d.%s.obb", found.VersionCode, found.Name))
		err := found.PullOBB(ctx, obbOut)
		if err != nil {
			return err
		}
	}

	return found.Pull(ctx, out)
}
