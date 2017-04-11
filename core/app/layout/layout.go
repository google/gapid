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
	"runtime"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/file"
)

const (
	ErrCannotFindPackageFiles = fault.Const("Cannot find package files")
	ErrUnknownABI             = fault.Const("Unknown device ABI: %+v")
)

// LibraryType enumerates the possible GAPID dynamic libraries.
type LibraryType int

const (
	LibGraphicsSpy LibraryType = iota
	LibVirtualSwapChain
)

// FileLayout provides a unified way of accessing various Gapid binaries.
type FileLayout interface {
	// Strings returns the path to the binary string table.
	Strings(ctx context.Context) (file.Path, error)
	// Gapit returns the path to the gapit binary in this layout.
	Gapit(ctx context.Context) (file.Path, error)
	// Gapis returns the path to the gapis binary in this layout.
	Gapis(ctx context.Context) (file.Path, error)
	// Gapir returns the path to the gapir binary in this layout.
	Gapir(ctx context.Context) (file.Path, error)
	// GapidApk returns the path to gapid.apk in this layout.
	GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error)
	// Library returns the path to the requested library.
	Library(ctx context.Context, lib LibraryType) (file.Path, error)
	// Json returns the path to the Vulkan layer JSON definition for the given library.
	Json(ctx context.Context, lib LibraryType) (file.Path, error)
}

var libTypeToName = map[LibraryType]string{
	LibGraphicsSpy:      withPlatformSuffix("libgapii"),
	LibVirtualSwapChain: withPlatformSuffix("libVkLayer_VirtualSwapchain"),
}

var libTypeToJson = map[LibraryType]string{
	LibGraphicsSpy:      "GraphicsSpyLayer.json",
	LibVirtualSwapChain: "VirtualSwapchainLayer.json",
}

func withPlatformSuffix(lib string) string {
	switch runtime.GOOS {
	case "windows":
		return lib + ".dll"
	default:
		return lib + ".so"
	}
}

var abiToApk = map[device.Architecture]string{
	device.ARMv7a: "gapid-armeabi.apk",
	device.ARMv8a: "gapid-aarch64.apk",
	device.X86:    "gapid-x86.apk",
}

// pkgLayout is the file layout used when running executables from a packaged
// build.
type pkgLayout struct {
	root file.Path
}

func (l pkgLayout) Strings(ctx context.Context) (file.Path, error) {
	return l.root.Join("strings", "en-us.stb"), nil
}

func (l pkgLayout) Gapit(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapit"), nil
}

func (l pkgLayout) Gapir(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapir"), nil
}

func (l pkgLayout) Gapis(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapis"), nil
}

func (l pkgLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return l.root.Join(abiToApk[abi.Architecture]), nil
}

func (l pkgLayout) Library(ctx context.Context, lib LibraryType) (file.Path, error) {
	return l.root.Join("lib", libTypeToName[lib]), nil
}

func (l pkgLayout) Json(ctx context.Context, lib LibraryType) (file.Path, error) {
	return l.root.Join("lib", libTypeToJson[lib]), nil
}

var binABIToDir = map[string]string{
	"armeabi":     "android-armv7a",
	"armeabi-v7a": "android-armv7a",
	"arm64-v8a":   "android-armv8a",
	"x86":         "android-x86",
}

// binLayout is the file layout used when running executables from the build's
// bin directory.
type binLayout struct {
	root file.Path
}

func abiDirectory(ctx context.Context, abi *device.ABI) (string, error) {
	dir, ok := binABIToDir[abi.Name]
	if !ok {
		return "", log.Errf(ctx, ErrUnknownABI, "ABI: %v", abi)
	}
	return dir, nil
}

func (l binLayout) file(ctx context.Context, abi *device.ABI, name string) (file.Path, error) {
	if abi.OS == host.Instance(ctx).Configuration.OS.Kind {
		return l.root.Join(name), nil
	}
	dir, err := abiDirectory(ctx, abi)
	if err != nil {
		return file.Path{}, err
	}
	return l.root.Join(dir, name), nil
}

func (l binLayout) Strings(ctx context.Context) (file.Path, error) {
	return l.root.Join("strings", "en-us.stb"), nil
}

func (l binLayout) Gapit(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapit"), nil
}

func (l binLayout) Gapir(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapir"), nil
}

func (l binLayout) Gapis(ctx context.Context) (file.Path, error) {
	return l.root.Join("gapis"), nil
}

func (l binLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return l.file(ctx, abi, abiToApk[abi.Architecture])
}

func (l binLayout) Library(ctx context.Context, lib LibraryType) (file.Path, error) {
	return l.root.Join(libTypeToName[lib]), nil
}

func (l binLayout) Json(ctx context.Context, lib LibraryType) (file.Path, error) {
	return l.root.Join(libTypeToJson[lib]), nil
}

// unknownLayout is the file layout used when no other layout can be discovered.
// All methods will return an error.
type unknownLayout struct{}

func (l unknownLayout) Strings(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapit(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapis(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapir(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Library(ctx context.Context, lib LibraryType) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Json(ctx context.Context, lib LibraryType) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

// BinLayout returns a binLayout implementation of FileLayout rooted in the given directory.
func BinLayout(root file.Path) FileLayout {
	return binLayout{root}
}
