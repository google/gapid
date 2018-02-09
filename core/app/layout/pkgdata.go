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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
)

// resolvedLayout is the data file layout discovered by layout()
// Call layout() instead of using this directly.
var resolvedLayout FileLayout

var (
	runfiles = flag.String("runfiles", "", "Location of the runfiles manifest")
)

func layout(ctx context.Context) (out FileLayout) {
	if resolvedLayout != nil {
		return resolvedLayout
	}
	defer func() { resolvedLayout = out }()
	for _, dir := range []file.Path{
		file.ExecutablePath().Parent(),
		file.Abs("."),
		// This is needed for running integration tests.
		file.ExecutablePath().Parent().Parent().Join("bin"),
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
		if dir.Join("build.properties").Exists() {
			return pkgLayout{dir}
		}
		// Check bin layout from executable's directory.
		// bin
		//  ├─── android-armv7a
		//  │     └─── gapid-armv7a.apk
		//  ├─── android-armv8a
		//  │     └─── gapid-armv8a.apk
		//  ├─── android-x86
		//  │     └─── gapid-x86.apk
		//  ├─── strings
		//  │     └─── en-us.stb
		//  ├─── gapir
		//  ├─── gapis
		//  ↓
		for _, abiDirName := range binABIToDir {
			if dir.Join(abiDirName).Exists() {
				return binLayout{dir}
			}
		}
	}

	if *runfiles != "" {
		if layout, err := RunfilesLayout(file.Abs(*runfiles)); err == nil {
			return layout
		}
	}

	if path := file.Abs(file.ExecutablePath().System() + ".runfiles_manifest"); path.Exists() {
		if layout, err := RunfilesLayout(path); err == nil {
			return layout
		}
	}

	// We're possibly dealing with a sparse-build, as used by robot.
	// gapis always has to exist.
	// TODO: this is kind of dumb, since gapis will always find itself.
	dir := file.ExecutablePath().Parent()
	if _, err := file.FindExecutable(dir.Join("gapis").System()); err == nil {
		return binLayout{dir}
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
func Gapir(ctx context.Context) (file.Path, error) {
	return layout(ctx).Gapir(ctx)
}

// Gapit returns the path to the gapir binary.
func Gapit(ctx context.Context) (file.Path, error) {
	return layout(ctx).Gapit(ctx)
}

// Library returns the path to the requested library.
func Library(ctx context.Context, lib LibraryType) (file.Path, error) {
	return layout(ctx).Library(ctx, lib)
}

// Json returns the path to the Vulkan layer JSON definition for the given library.
func Json(ctx context.Context, lib LibraryType) (file.Path, error) {
	return layout(ctx).Json(ctx, lib)
}

// GoArgs returns additional arguments to pass to go binaries.
func GoArgs(ctx context.Context) []string {
	return layout(ctx).GoArgs(ctx)
}
