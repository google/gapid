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
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/os/device"
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
	LibCPUTiming
	LibDebugMarker
	LibMemoryTracker
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
	Gapir(ctx context.Context, abi *device.ABI) (file.Path, error)
	// GapidApk returns the path to gapid.apk in this layout.
	GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error)
	// PerfettoCmd returns the path to the perfetto command-line tool
	PerfettoCmd(ctx context.Context, abi *device.ABI) (file.Path, error)
	// Library returns the path to the requested library.
	Library(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error)
	// Json returns the path to the Vulkan layer JSON definition for the given library.
	Json(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error)
	// GoArgs returns additional arguments to pass to go binaries.
	GoArgs(ctx context.Context) []string
	// DeviceInfo returns the device info executable for the given ABI.
	DeviceInfo(ctx context.Context, os device.OSKind) (file.Path, error)
}

func withExecutablePlatformSuffix(exe string, os device.OSKind) string {
	if os == device.Windows {
		return exe + ".exe"
	}
	return exe
}

var libTypeToName = map[LibraryType]string{
	LibGraphicsSpy:      "libgapii",
	LibVirtualSwapChain: "libVkLayer_VirtualSwapchain",
	LibCPUTiming:        "libVkLayer_CPUTiming",
	LibDebugMarker:      "libVkLayer_DebugMarker",
	LibMemoryTracker:    "libVkLayer_MemoryTracker",
}

var layerNameToLibType = map[string]LibraryType{
	"VirtualSwapchain": LibVirtualSwapChain,
	"CPUTiming":        LibCPUTiming,
	"DebugMarker":      LibDebugMarker,
	"MemoryTracker":    LibMemoryTracker,
}

var dataSourceNameToLayerName = map[string]string{
	"VirtualSwapchain":    "VirtualSwapchain",
	"VulkanCPUTiming":     "CPUTiming",
	"VulkanMemoryTracker": "MemoryTracker",
	"VulkanAPI":           "DebugMarker",
}

var libTypeToJson = map[LibraryType]string{
	LibGraphicsSpy:      "GraphicsSpyLayer.json",
	LibVirtualSwapChain: "VirtualSwapchainLayer.json",
	LibCPUTiming:        "CPUTimingLayer.json",
	LibMemoryTracker:    "MemoryTrackerLayer.json",
	LibDebugMarker:      "DebugMarker.json",
}

func withLibraryPlatformSuffix(lib string, os device.OSKind) string {
	switch os {
	case device.Windows:
		return lib + ".dll"
	case device.OSX:
		return lib + ".dylib"
	default:
		return lib + ".so"
	}
}

// LibraryName returns the filename of the given Library.
func LibraryName(lib LibraryType, abi *device.ABI) string {
	return withLibraryPlatformSuffix(libTypeToName[lib], abi.OS)
}

// Returns a layer name from a data source name
func LayerFromDataSource(name string) (string, error) {
	if v, ok := dataSourceNameToLayerName[name]; ok {
		return v, nil
	}
	return "", fmt.Errorf("Invalid Datasource Name %s", name)
}

// Returns a library from a given LayerName
func LibraryFromLayerName(name string) (LibraryType, error) {
	if v, ok := layerNameToLibType[name]; ok {
		return v, nil
	}
	return -1, fmt.Errorf("Invalid Layer Name %s", name)
}

func AllLayers() []string {
	s := []string{}
	for k := range layerNameToLibType {
		s = append(s, k)
	}
	return s
}

var abiToApk = map[device.Architecture]string{
	device.ARMv7a: "gapid-armeabi-v7a.apk",
	device.ARMv8a: "gapid-arm64-v8a.apk",
	device.X86:    "gapid-x86.apk",
}

func hostOS(ctx context.Context) device.OSKind {
	var dev device.OSKind
	switch runtime.GOOS {
	case "windows":
		dev = device.Windows
	case "linux":
		dev = device.Linux
	case "darwin":
		dev = device.OSX
	}
	return dev
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
	return l.root.Join(withExecutablePlatformSuffix("gapit", hostOS(ctx))), nil
}

func osToDir(k device.OSKind) string {
	switch k {
	case device.Linux:
		return "linux"
	case device.OSX:
		return "macos"
	case device.Windows:
		return "windows"
	default:
		return ""
	}
}

func (l pkgLayout) Gapir(ctx context.Context, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.root.Join(withExecutablePlatformSuffix("gapir", hostOS(ctx))), nil
	}
	return l.root.Join(osToDir(abi.OS), withExecutablePlatformSuffix("gapir", abi.OS)), nil
}

func (l pkgLayout) Gapis(ctx context.Context) (file.Path, error) {
	return l.root.Join(withExecutablePlatformSuffix("gapis", hostOS(ctx))), nil
}

func (l pkgLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return l.root.Join(abiToApk[abi.Architecture]), nil
}

func (l pkgLayout) Library(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.root.Join("lib", withLibraryPlatformSuffix(libTypeToName[lib], hostOS(ctx))), nil
	}
	return l.root.Join(osToDir(abi.OS), "lib", withLibraryPlatformSuffix(libTypeToName[lib], abi.OS)), nil
}

func (l pkgLayout) PerfettoCmd(ctx context.Context, abi *device.ABI) (file.Path, error) {
	if hostOS(ctx) == abi.OS {
		return l.root.Join("perfetto", withExecutablePlatformSuffix("perfetto", abi.OS)), nil
	}
	return l.root.Join(osToDir(abi.OS), "perfetto", withExecutablePlatformSuffix("perfetto", abi.OS)), nil
}

func (l pkgLayout) Json(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.root.Join("lib", libTypeToJson[lib]), nil
	}
	return l.root.Join(osToDir(abi.OS), "lib", libTypeToJson[lib]), nil
}

func (l pkgLayout) GoArgs(ctx context.Context) []string {
	return nil
}

func (l pkgLayout) DeviceInfo(ctx context.Context, os device.OSKind) (file.Path, error) {
	if hostOS(ctx) == os {
		return l.root.Join(withExecutablePlatformSuffix("device-info", os)), nil
	}
	return l.root.Join(osToDir(os), withExecutablePlatformSuffix("device-info", os)), nil
}

// NewPkgLayout returns a FileLayout rooted at the given directory with the standard package layout.
// If create is true, the package layout is created if it doesn't exist, otherwise an error is returned.
func NewPkgLayout(dir file.Path, create bool) (FileLayout, error) {
	bp := dir.Join("build.properties")
	if !bp.Exists() {
		if !create {
			return nil, ErrCannotFindPackageFiles
		}
		if err := file.Mkfile(bp); err != nil {
			return nil, err
		}
	}
	return pkgLayout{dir}, nil
}

// runfilesLayout is a layout that uses the bazel generated runfiles manifest
// to find the various dependencies.
type runfilesLayout struct {
	manifest string
	mapping  map[string]string
}

var abiToApkPath = map[device.Architecture]string{
	device.ARMv7a: "armeabi-v7a.apk",
	device.ARMv8a: "arm64-v8a.apk",
	device.X86:    "x86.apk",
}

var libTypeToLibPath = map[LibraryType]string{
	LibGraphicsSpy:      "gapid/gapii/cc/libgapii",
	LibVirtualSwapChain: "gapid/core/vulkan/vk_virtual_swapchain/cc/libVkLayer_VirtualSwapchain",
	LibCPUTiming:        "gapid/core/vulkan/vk_api_timing_layer/cc/libVkLayer_CPUTiming",
	LibDebugMarker:      "gapid/core/vulkan/vk_api_timing_layer/cc/libVkLayer_DebugMarker",
	LibMemoryTracker:    "gapid/core/vulkan/vk_memory_tracker_layer/cc/libVkLayer_MemoryTracker",
}

var libTypeToJsonPath = map[LibraryType]string{
	LibGraphicsSpy:      "gapid/gapii/vulkan/vk_graphics_spy/cc/GraphicsSpyLayer.json",
	LibVirtualSwapChain: "gapid/core/vulkan/vk_virtual_swapchain/cc/VirtualSwapchainLayer.json",
	LibCPUTiming:        "gapid/core/vulkan/vk_api_timing_layer/cc/CPUTimingLayer.json",
	LibDebugMarker:      "gapid/core/vulkan/vk_api_timing_layer/cc/DebugMarkerLayer.json",
	LibMemoryTracker:    "gapid/core/vulkan/vk_memory_tracker_layer/cc/MemoryTrackerLayer.json",
}

// RunfilesLayout creates a new layout based on the given runfiles manifest.
func RunfilesLayout(manifest file.Path) (FileLayout, error) {
	file, err := os.Open(manifest.System())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	r := runfilesLayout{
		manifest: manifest.System(),
		mapping:  make(map[string]string),
	}
	for scanner.Scan() {
		line := scanner.Text()
		if p := strings.IndexRune(line, ' '); p > 0 {
			key, value := line[:p], line[p+1:]
			r.mapping[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &r, nil
}

func (l *runfilesLayout) find(key string) (file.Path, error) {
	if r, ok := l.mapping[key]; ok {
		return file.Abs(r), nil
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l *runfilesLayout) Strings(ctx context.Context) (file.Path, error) {
	return l.find("gapid/gapis/messages/en-us.stb")
}

func (l *runfilesLayout) Gapit(ctx context.Context) (file.Path, error) {
	return l.find(withExecutablePlatformSuffix("gapid/cmd/gapit/gapit", hostOS(ctx)))
}

func (l *runfilesLayout) Gapis(ctx context.Context) (file.Path, error) {
	return l.find(withExecutablePlatformSuffix("gapid/cmd/gapis/gapis", hostOS(ctx)))
}

func (l *runfilesLayout) Gapir(ctx context.Context, abi *device.ABI) (file.Path, error) {
	if hostOS(ctx) == abi.OS {
		return l.find(withExecutablePlatformSuffix("gapid/cmd/gapir/cc/gapir", hostOS(ctx)))
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l *runfilesLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return l.find("gapid/gapidapk/android/apk/" + abiToApkPath[abi.Architecture])
}

func (l *runfilesLayout) Library(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.find(withLibraryPlatformSuffix(libTypeToLibPath[lib], hostOS(ctx)))
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l *runfilesLayout) Json(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.find(libTypeToJsonPath[lib])
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l *runfilesLayout) GoArgs(ctx context.Context) []string {
	return []string{"-runfiles", l.manifest}
}

func (l *runfilesLayout) DeviceInfo(ctx context.Context, os device.OSKind) (file.Path, error) {
	if hostOS(ctx) == os {
		return l.find("core/os/device/deviceinfo/exe/" + withExecutablePlatformSuffix("device-info", os))
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l *runfilesLayout) PerfettoCmd(ctx context.Context, abi *device.ABI) (file.Path, error) {
	if abi == nil || hostOS(ctx) == abi.OS {
		return l.find("external/perfetto/" + withExecutablePlatformSuffix("perfetto_cmd", hostOS(ctx)))
	}
	return file.Path{}, ErrCannotFindPackageFiles
}

// unknownLayout is the file layout used when no other layout can be discovered.
// All methods will return an error.
type unknownLayout struct{}

func (l unknownLayout) Strings(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Gapit(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Gapis(ctx context.Context) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Gapir(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) GapidApk(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Library(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) Json(ctx context.Context, lib LibraryType, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) GoArgs(ctx context.Context) []string {
	return nil
}

func (l unknownLayout) DeviceInfo(ctx context.Context, os device.OSKind) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

func (l unknownLayout) PerfettoCmd(ctx context.Context, abi *device.ABI) (file.Path, error) {
	return file.Path{}, ErrCannotFindPackageFiles
}

// ZipLayout is a FileLayout view over a ZIP file.
type ZipLayout struct {
	f     *zip.Reader
	files map[string]*zip.File
	os    device.OSKind
}

// NewZipLayout returns a new ZipLayout using the given ZIP file.
func NewZipLayout(f *zip.Reader, os device.OSKind) *ZipLayout {
	r := &ZipLayout{
		f:     f,
		files: make(map[string]*zip.File, len(f.File)),
		os:    os,
	}

	for _, file := range f.File {
		r.files[strings.TrimPrefix(file.Name, "gapid/")] = file
	}

	return r
}

// file returns the zip.File of the given path or an error if it's not found.
func (l *ZipLayout) file(path string) (*zip.File, error) {
	if f, ok := l.files[path]; ok {
		return f, nil
	}
	return nil, ErrCannotFindPackageFiles
}

// Strings returns the path to the binary string table.
func (l *ZipLayout) Strings(ctx context.Context) (*zip.File, error) {
	return l.file("strings/en-us.stb")
}

// Gapit returns the path to the gapit binary in this layout.
func (l *ZipLayout) Gapit(ctx context.Context) (*zip.File, error) {
	return l.file(withExecutablePlatformSuffix("gapit", l.os))
}

// Gapir returns the path to the gapir binary in this layout.
func (l *ZipLayout) Gapir(ctx context.Context, abi *device.ABI) (*zip.File, error) {
	if abi == nil || l.os == abi.OS {
		return l.file(withExecutablePlatformSuffix("gapir", l.os))
	}
	return l.file(osToDir(abi.OS) + "/" + withExecutablePlatformSuffix("gapir", abi.OS))
}

// Gapis returns the path to the gapis binary in this layout.
func (l *ZipLayout) Gapis(ctx context.Context) (*zip.File, error) {
	return l.file(withExecutablePlatformSuffix("gapis", l.os))
}

// GapidApk returns the path to gapid.apk in this layout.
func (l *ZipLayout) GapidApk(ctx context.Context, abi *device.ABI) (*zip.File, error) {
	return l.file(abiToApk[abi.Architecture])
}

// Library returns the path to the requested library.
func (l *ZipLayout) Library(ctx context.Context, lib LibraryType, abi *device.ABI) (*zip.File, error) {
	if abi == nil || l.os == abi.OS {
		return l.file("lib/" + withLibraryPlatformSuffix(libTypeToName[lib], l.os))
	}
	return l.file(osToDir(abi.OS) + "lib/" + withLibraryPlatformSuffix(libTypeToName[lib], abi.OS))
}

// Json returns the path to the Vulkan layer JSON definition for the given library.
func (l *ZipLayout) Json(ctx context.Context, lib LibraryType, abi *device.ABI) (*zip.File, error) {
	if abi == nil || l.os == abi.OS {
		return l.file("lib/" + libTypeToJson[lib])
	}
	return l.file(osToDir(abi.OS) + "lib/" + libTypeToJson[lib])
}

// DeviceInfo returns the device info executable for the given ABI.
func (l *ZipLayout) DeviceInfo(ctx context.Context, os device.OSKind) (*zip.File, error) {
	if l.os == os {
		return l.file(withExecutablePlatformSuffix("device-info", os))
	}
	return l.file(osToDir(os) + "/" + withExecutablePlatformSuffix("device-info", os))
}

// PerfettoCmd returns the device info executable for the given ABI.
func (l *ZipLayout) PerfettoCmd(ctx context.Context, abi *device.ABI) (*zip.File, error) {
	if l.os == abi.OS {
		return l.file("perfetto/" + withExecutablePlatformSuffix("perfetto", abi.OS))
	}
	return l.file(osToDir(abi.OS) + "/perfetto/" + withExecutablePlatformSuffix("perfetto", abi.OS))
}
