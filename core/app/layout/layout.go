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
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
)

const (
	ErrCannotFindPackageFiles = fault.Const("Cannot find package files")
	ErrUnknownABI             = fault.Const("Unknown device ABI: %+v")
)

// FileLayout provides a unified way of accessing various Gapid binaries.
type FileLayout interface {
	// File returns the path to the specified file for the given ABI.
	File(ctx log.Context, abi *device.ABI, name string) (file.Path, error)
	// Strings returns the path to the binary string table.
	Strings(ctx log.Context) (file.Path, error)
	// Gapit returns the path to the gapit binary in this layout.
	Gapit(ctx log.Context) (file.Path, error)
	// Gapis returns the path to the gapis binary in this layout.
	Gapis(ctx log.Context) (file.Path, error)
	// Gapir returns the path to the gapir binary in this layout.
	Gapir(ctx log.Context) (file.Path, error)
	// GapidApk returns the path to gapid.apk in this layout.
	GapidApk(ctx log.Context, abi *device.ABI) (file.Path, error)
}

var packageOSToDir = map[device.OSKind]string{
	device.UnknownOS: "unknown",
	device.Windows:   "win32",
	device.OSX:       "darwin",
	device.Linux:     "linux",
	device.Android:   "android",
}

var pkgABIToDir = map[device.Architecture]string{
	device.ARMv7a: "armeabi-v7a",
	device.ARMv8a: "arm64-v8a",
	device.X86:    "x86",
	device.X86_64: "x86_64",
}

// pkgLayout is the file layout used when running executables from a packaged
// build.
type pkgLayout struct {
	root file.Path
}

func (l pkgLayout) File(ctx log.Context, abi *device.ABI, name string) (file.Path, error) {
	return l.root.Join(packageOSToDir[abi.OS], pkgABIToDir[abi.Architecture], name), nil
}

func (l pkgLayout) Strings(ctx log.Context) (file.Path, error) {
	return l.root.Join("strings", "en-us.stb"), nil
}

func (l pkgLayout) Gapit(ctx log.Context) (file.Path, error) {
	return l.File(ctx, device.Host(ctx).Configuration.ABIs[0], "gapit")
}

func (l pkgLayout) Gapir(ctx log.Context) (file.Path, error) {
	return l.File(ctx, device.Host(ctx).Configuration.ABIs[0], "gapir")
}

func (l pkgLayout) Gapis(ctx log.Context) (file.Path, error) {
	return l.File(ctx, device.Host(ctx).Configuration.ABIs[0], "gapis")
}

func (l pkgLayout) GapidApk(ctx log.Context, abi *device.ABI) (file.Path, error) {
	return l.File(ctx, abi, "gapid.apk")
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

func abiDirectory(ctx log.Context, abi *device.ABI) (string, error) {
	dir, ok := binABIToDir[abi.Name]
	if !ok {
		return "", cause.Wrap(ctx, ErrUnknownABI).With("ABI", abi)
	}
	return dir, nil
}

func (l binLayout) File(ctx log.Context, abi *device.ABI, name string) (file.Path, error) {
	if abi.OS == device.Host(ctx).Configuration.OS.Kind {
		return l.root.Join(name), nil
	}
	dir, err := abiDirectory(ctx, abi)
	if err != nil {
		return file.Path{}, err
	}
	return l.root.Join(dir, name), nil
}

func (l binLayout) Strings(ctx log.Context) (file.Path, error) {
	return l.root.Join("strings", "en-us.stb"), nil
}

func (l binLayout) Gapit(ctx log.Context) (file.Path, error) {
	return l.root.Join("gapit"), nil
}

func (l binLayout) Gapir(ctx log.Context) (file.Path, error) {
	return l.root.Join("gapir"), nil
}

func (l binLayout) Gapis(ctx log.Context) (file.Path, error) {
	return l.root.Join("gapis"), nil
}

func (l binLayout) GapidApk(ctx log.Context, abi *device.ABI) (file.Path, error) {
	return l.File(ctx, abi, "gapid.apk")
}

// unknownLayout is the file layout used when no other layout can be discovered.
// All methods will return an error.
type unknownLayout struct{}

func (l unknownLayout) File(ctx log.Context, abi *device.ABI, name string) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Strings(ctx log.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapit(ctx log.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapis(ctx log.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) Gapir(ctx log.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (a unknownLayout) GapidApk(ctx log.Context, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

// BinLayout returns a binLayout implementation of FileLayout rooted in the given directory.
func BinLayout(root file.Path) FileLayout {
	return binLayout{root}
}
