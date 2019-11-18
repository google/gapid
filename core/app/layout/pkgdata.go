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

package layout

import (
	"context"
	"flag"
	"path/filepath"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
)

// resolvedLayout is the data file layout discovered by layout()
// Call layout() instead of using this directly.
var resolvedLayout FileLayout

var (
	runfiles = flag.String("runfiles", "", "_Location of the runfiles manifest")
)

func layout(ctx context.Context) (out FileLayout) {
	if resolvedLayout != nil {
		return resolvedLayout
	}
	defer func() { resolvedLayout = out }()
	for _, dir := range []file.Path{
		file.ExecutablePath().Parent(),
		file.Abs("."),
	} {
		log.D(ctx, "Looking for package in %v", dir)

		// Check the regular package layout first:
		// pkg
		//  ├─── build.properties
		//  ├─── strings
		//  │     └─── en-us.stb
		//  ├─── gapid-<abi>.apk
		//  ├─── gapir
		//  ├─── gapis
		//  ├─── gapit
		//  ↓
		if layout, err := NewPkgLayout(dir, false); err == nil {
			return layout
		}
	}

	if *runfiles != "" {
		if layout, err := RunfilesLayout(file.Abs(*runfiles)); err == nil {
			return layout
		}
	}

	exe := file.ExecutablePath().System()
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	log.D(ctx, "Looking for runfiles manifest for %v", exe)
	if path := file.Abs(exe + ".runfiles_manifest"); path.Exists() {
		if layout, err := RunfilesLayout(path); err == nil {
			return layout
		}
	}

	return unknownLayout{}
}

// Strings returns the path to the binary string table.
func Strings(ctx context.Context) (file.Path, error) {
	return layout(ctx).Strings(ctx)
}

// Gapis returns the path to the gapis binary.
func Gapis(ctx context.Context) (file.Path, error) {
	return layout(ctx).Gapis(ctx)
}

// GapidApk returns the path to the gapid.apk corresponding to the given abi.
func GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return layout(ctx).GapidApk(ctx, abi)
}

// Gapir returns the path to the gapir binary.
func Gapir(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return layout(ctx).Gapir(ctx, abi)
}

// Gapit returns the path to the gapir binary.
func Gapit(ctx context.Context) (file.Path, error) {
	return layout(ctx).Gapit(ctx)
}

// Library returns the path to the requested library.
func Library(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	return layout(ctx).Library(ctx, lib, abi)
}

// Json returns the path to the Vulkan layer JSON definition for the given library.
func Json(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	return layout(ctx).Json(ctx, lib, abi)
}

// GoArgs returns additional arguments to pass to go binaries.
func GoArgs(ctx context.Context) []string {
	return layout(ctx).GoArgs(ctx)
}

// DeviceInfo returns the device info executable for the given ABI.
func DeviceInfo(ctx context.Context, os device.OSKind) (file.Path, error) {
	return layout(ctx).DeviceInfo(ctx, os)
}

// PerfettoCmd returns the device info executable for the given ABI.
func PerfettoCmd(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return layout(ctx).PerfettoCmd(ctx, abi)
}
