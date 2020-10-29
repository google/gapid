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

package adb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/perfetto"

	common_pb "protos/perfetto/common"
)

const (
	// ErrDeviceNotRooted is returned by Device.Root when the device is running a
	// production build as is not 'rooted'.
	ErrDeviceNotRooted = fault.Const("Device is not a userdebug build")
	ErrRootFailed      = fault.Const("Device failed to switch to root")

	maxRootAttempts                         = 5
	gpuRenderStagesDataSourceDescriptorName = "gpu.renderstages"
	gpuMemTotalDataSourceDescriptorName     = "android.gpu.memory"

	perfettoPort = NamedFileSystemSocket("/dev/socket/traced_consumer")

	driverProperty = "ro.gfx.driver.1"

	systemImageGpuProfilerSupportProperty = "graphics.gpu.profiler.support"
	gpuProfilerVulkanLayerApkProperty     = "graphics.gpu.profiler.vulkan_layer_apk"
)

func isRootSuccessful(line string) bool {
	for _, expected := range []string{
		"adbd is already running as root",
		"* daemon started successfully *",
	} {
		if line == expected {
			return true
		}
	}
	return false
}

// Root restarts adb as root. If the device is running a production build then
// Root will return ErrDeviceNotRooted.
func (b *binding) Root(ctx context.Context) error {
	buf := bytes.Buffer{}
	buf.WriteString("adb root gave output:")
retry:
	for attempt := 0; attempt < maxRootAttempts; attempt++ {
		output, err := b.Command("root").Call(ctx)
		if err != nil {
			return err
		}
		if len(output) == 0 {
			return nil // Assume no output is success
		}
		output = strings.Replace(output, "\r\n", "\n", -1) // Not expected, but let's be safe.
		buf.WriteString(fmt.Sprintf("\n#%d: %v", attempt, output))
		lines := strings.Split(output, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			if isRootSuccessful(line) {
				return nil // Success
			}
			switch line {
			case "adbd cannot run as root in production builds":
				return ErrDeviceNotRooted
			case "restarting adbd as root":
				time.Sleep(time.Millisecond * 100)
				continue retry
			default:
				// Some output we weren't expecting.
			}
		}
	}
	return log.Err(ctx, ErrRootFailed, buf.String())
}

// IsDebuggableBuild returns true if the device runs a debuggable Android build.
func (b *binding) IsDebuggableBuild(ctx context.Context) (bool, error) {
	output, err := b.Command("shell", "getprop", "ro.debuggable").Call(ctx)
	if err != nil {
		return false, err
	}
	return output == "1", nil
}

// InstallAPK installs the specified APK to the device. If reinstall is true
// and the package is already installed on the device then it will be replaced.
func (b *binding) InstallAPK(ctx context.Context, path string, reinstall bool, grantPermissions bool) error {
	args := []string{}
	if reinstall {
		args = append(args, "-r")
	}
	if grantPermissions && b.Instance().GetConfiguration().GetOS().GetAPIVersion() >= 23 {
		// Starting with API 23, permissions are not granted by default
		// during installation. Before API 23, the flag did not exist.
		args = append(args, "-g")
	}
	args = append(args, path)
	return b.Command("install", args...).Run(ctx)
}

// SELinuxEnforcing returns true if the device is currently in a
// SELinux enforcing mode, or false if the device is currently in a SELinux
// permissive mode.
func (b *binding) SELinuxEnforcing(ctx context.Context) (bool, error) {
	res, err := b.Shell("getenforce").Call(ctx)
	return strings.Contains(strings.ToLower(res), "enforcing"), err
}

// SetSELinuxEnforcing changes the SELinux-enforcing mode.
func (b *binding) SetSELinuxEnforcing(ctx context.Context, enforce bool) error {
	if enforce {
		return b.Shell("setenforce", "1").Run(ctx)
	}
	return b.Shell("setenforce", "0").Run(ctx)
}

// StartActivity launches the specified activity action.
func (b *binding) StartActivity(ctx context.Context, a android.ActivityAction, extras ...android.ActionExtra) error {
	args := append([]string{
		"start",
		"-S", // Force-stop the target app before starting the activity
		"-W", // Wait until the launch finishes
		"-a", a.Name,
		"-n", a.Component(),
	}, extrasFlags(extras)...)
	return b.Shell("am", args...).Run(ctx)
}

// StartActivityForDebug launches the specified activity in debug mode.
func (b *binding) StartActivityForDebug(ctx context.Context, a android.ActivityAction, extras ...android.ActionExtra) error {
	args := append([]string{
		"start",
		"-S", // Force-stop the target app before starting the activity
		"-W", // Wait until the launch finishes
		"-D", // Debug mode
		"-a", a.Name,
		"-n", a.Component(),
	}, extrasFlags(extras)...)
	return b.Shell("am", args...).Run(ctx)
}

// StartService launches the specified service action.
func (b *binding) StartService(ctx context.Context, a android.ServiceAction, extras ...android.ActionExtra) error {
	cmd := "start-foreground-service"
	if b.Instance().GetConfiguration().GetOS().GetAPIVersion() < 26 {
		// "am start-foreground-service" was added in API 26.
		cmd = "startservice"
	}
	args := append([]string{
		cmd,
		"-a", a.Name,
		"-n", a.Component(),
	}, extrasFlags(extras)...)
	return b.Shell("am", args...).Run(ctx)
}

// ForceStop stops everything associated with the given package.
func (b *binding) ForceStop(ctx context.Context, pkg string) error {
	return b.Shell("am", "force-stop", pkg).Run(ctx)
}

// SystemProperty returns the system property in string
func (b *binding) SystemProperty(ctx context.Context, name string) (string, error) {
	res, err := b.Shell("getprop", name).Call(ctx)
	if err != nil {
		return "", log.Errf(ctx, err, "getprop returned error: \n%s", err.Error())
	}
	return res, nil
}

// SetSystemProperty sets the system property with the given string value
func (b *binding) SetSystemProperty(ctx context.Context, name, value string) error {
	if len(value) == 0 {
		value = `""`
	}
	res, err := b.Shell("setprop", name, value).Call(ctx)
	if res != "" {
		return log.Errf(ctx, nil, "setprop returned error: \n%s", res)
	}
	if err != nil {
		return err
	}
	return nil
}

// SystemSetting returns the system setting with the given namespaced key.
func (b *binding) SystemSetting(ctx context.Context, namespace, key string) (string, error) {
	res, err := b.Shell("settings", "get", namespace, key).Call(ctx)
	if err != nil {
		return "", log.Errf(ctx, err, "settings get returned error: \n%s", err.Error())
	}
	return res, nil
}

// SetSystemSetting sets the system setting with with the given namespaced key
// to value.
func (b *binding) SetSystemSetting(ctx context.Context, namespace, key, value string) error {
	res, err := b.Shell("settings", "put", namespace, key, value).Call(ctx)
	if err != nil {
		return log.Errf(ctx, nil, "settings put returned error: \n%s", res)
	}
	return nil
}

// DeleteSystemSetting removes the system setting with with the given namespaced key.
func (b *binding) DeleteSystemSetting(ctx context.Context, namespace, key string) error {
	res, err := b.Shell("settings", "delete", namespace, key).Call(ctx)
	if err != nil {
		return log.Errf(ctx, nil, "settings delete returned error: \n%s", res)
	}
	return nil
}

// TempFile creates a temporary file on the given Device. It returns the
// path to the file, and a function that can be called to clean it up.
func (b *binding) TempFile(ctx context.Context) (string, func(ctx context.Context), error) {
	res, err := b.Shell("mktemp").Call(ctx)
	if err != nil {
		return "", nil, err
	}
	return res, func(ctx context.Context) {
		b.Shell("rm", "-f", res).Call(ctx)
	}, nil
}

// FileContents returns the contents of a given file on the Device.
func (b *binding) FileContents(ctx context.Context, path string) (string, error) {
	return b.Shell("cat", path).Call(ctx)
}

// RemoveFile removes the given file from the device
func (b *binding) RemoveFile(ctx context.Context, path string) error {
	_, err := b.Shell("rm", "-f", path).Call(ctx)
	return err
}

// GetEnv returns the default environment for the Device.
func (b *binding) GetEnv(ctx context.Context) (*shell.Env, error) {
	env, err := b.Shell("env").Call(ctx)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(env))
	e := &shell.Env{}
	for scanner.Scan() {
		e.Add(scanner.Text())
	}
	return e, nil
}

func (b *binding) SupportsPerfetto(ctx context.Context) bool {
	os := b.Instance().GetConfiguration().GetOS()
	return os.GetAPIVersion() >= 28
}

func (b *binding) ConnectPerfetto(ctx context.Context) (*perfetto.Client, error) {
	if !b.SupportsPerfetto(ctx) {
		return nil, fmt.Errorf("Perfetto is not supported on this device")
	}

	localPort, err := LocalFreeTCPPort()
	if err != nil {
		return nil, err
	}

	if err := b.Forward(ctx, localPort, perfettoPort); err != nil {
		return nil, err
	}
	cleanup := app.Cleanup(func(ctx context.Context) {
		b.RemoveForward(ctx, localPort)
	})

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%v", localPort))
	if err != nil {
		cleanup.Invoke(ctx)
		return nil, err
	}
	return perfetto.NewClient(ctx, conn, cleanup)
}

// EnsurePerfettoPersistent ensures that Perfetto daemons, traced and
// traced_probes, are running. Note that there is a delay between setting the
// system property and daemons finish starting, hence this function needs to be
// called as early as possible.
func (b *binding) EnsurePerfettoPersistent(ctx context.Context) error {
	if !b.SupportsPerfetto(ctx) {
		return nil
	}
	if err := b.SetSystemProperty(ctx, "persist.traced.enable", "1"); err != nil {
		return err
	}
	return nil
}

// QueryPerfettoServiceState queries all existing perfetto data sources
// regardless of which it is registered from.
func (b *binding) QueryPerfettoServiceState(ctx context.Context) (*device.PerfettoCapability, error) {
	result := b.To.Configuration.PerfettoCapability
	if result == nil {
		result = &device.PerfettoCapability{
			GpuProfiling: &device.GPUProfiling{},
		}
	}

	if !b.SupportsPerfetto(ctx) {
		return result, fmt.Errorf("Perfetto is not supported on this device")
	}

	gpu, err := b.QueryPerfettoGpuProfilingDataSources(ctx)
	if err != nil {
		return result, log.Errf(ctx, err, "Failed to query perfetto GPU profiling data sources.")
	}
	result.GpuProfiling = gpu

	// SurfaceFlinger frame lifecycle perfetto producer is mandated by Android 11 CTS, hence it will
	// always exist.
	if b.Instance().GetConfiguration().GetOS().GetAPIVersion() >= 30 {
		result.HasFrameLifecycle = true
	}
	return result, nil
}

// QueryPerfettoGpuProfilingDataSources queries and returns the data sources
// that support the GPU profiling functionailities, it includes:
//     1) gpu.counters
//     2) gpu.renderstages
//     3) android.gpu.memory
func (b *binding) QueryPerfettoGpuProfilingDataSources(ctx context.Context) (*device.GPUProfiling, error) {
	gpu := &device.GPUProfiling{}
	encoded, err := b.Shell("perfetto", "--query-raw", "|", "base64").Call(ctx)
	if err != nil {
		return gpu, log.Errf(ctx, err, "adb shell perfetto returned error: %s", encoded)
	}
	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	state := &common_pb.TracingServiceState{}
	if err = proto.Unmarshal(decoded, state); err != nil {
		return gpu, log.Errf(ctx, err, "Unmarshal returned error")
	}

	for _, ds := range state.GetDataSources() {
		desc := ds.GetDsDescriptor()
		if desc.GetName() == gpuRenderStagesDataSourceDescriptorName {
			gpu.HasRenderStage = true
			continue
		}
		if desc.GetName() == gpuMemTotalDataSourceDescriptorName {
			gpu.HasGpuMemTotal = true
			continue
		}
		counters := desc.GetGpuCounterDescriptor().GetSpecs()
		if len(counters) != 0 {
			if gpu.GpuCounterDescriptor == nil {
				gpu.GpuCounterDescriptor = &device.GpuCounterDescriptor{}
			}
			// We mirror the Perfetto GpuCounterDescriptor proto into AGI, hence
			// they are binary format compatible.
			data, err := proto.Marshal(desc.GetGpuCounterDescriptor())
			if err != nil {
				continue
			}
			proto.UnmarshalMerge(data, gpu.GpuCounterDescriptor)
		}
	}
	return gpu, nil
}

// Currently using Android Q/10, API 29 as ANGLE support cut-off
func (b *binding) SupportsAngle(ctx context.Context) bool {
	os := b.Instance().GetConfiguration().GetOS()
	return os.GetAPIVersion() >= 29
}

func (b *binding) QueryAnglePackageName(ctx context.Context) (string, error) {
	if !b.SupportsAngle(ctx) {
		return "", fmt.Errorf("ANGLE not supported on this device")
	}
	// ANGLE supported, so check for installed ANGLE package
	// Favor custom installed package first, followed by default system package
	custom := "org.chromium.angle"
	system := "com.google.android.angle"
	// Check installed packages for ANGLE package
	packages, _ := b.InstalledPackages(ctx)
	if pkg := packages.FindByName(custom); pkg != nil {
		return custom, nil
	}
	if pkg := packages.FindByName(system); pkg != nil {
		return system, nil
	}
	return "", fmt.Errorf("No ANGLE packages installed on this device")
}

func SetupAngle(ctx context.Context, d Device, p *android.InstalledPackage) (app.Cleanup, error) {
	// Restore ANGLE settings during app cleanup
	angleDriverValues, err := d.SystemSetting(ctx, "global", "angle_gl_driver_selection_values")
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to get original ANGLE driver selection name.")
	}
	angleDriverPkgs, err := d.SystemSetting(ctx, "global", "angle_gl_driver_selection_pkgs")
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to get original ANGLE enabled packages.")
	}
	anglePackage, err := d.SystemSetting(ctx, "global", "angle_debug_package")
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to get original ANGLE package name.")
	}
	log.I(ctx, "Saved original ANGLE pkg %s for app %s to restore during cleanup.", anglePackage, angleDriverPkgs)
	// We successfully saved old settings, now set new ANGLE values for tracing
	d.SetSystemSetting(ctx, "global", "angle_gl_driver_selection_values", "angle")
	d.SetSystemSetting(ctx, "global", "angle_debug_package", d.Instance().GetConfiguration().GetAnglePackage())
	d.SetSystemSetting(ctx, "global", "angle_gl_driver_selection_pkgs", p.Name)
	// Enable ANGLE debug markers.
	d.SetSystemProperty(ctx, "debug.angle.markers", "1")
	// Return cleanup function to restore original ANGLE settings
	return func(ctx context.Context) {
		d.SetSystemSetting(ctx, "global", "angle_gl_driver_selection_values", angleDriverValues)
		d.SetSystemSetting(ctx, "global", "angle_gl_driver_selection_pkgs", angleDriverPkgs)
		d.SetSystemSetting(ctx, "global", "angle_debug_package", anglePackage)
		d.SetSystemProperty(ctx, "debug.angle.markers", "")
	}, nil
}

func extrasFlags(extras []android.ActionExtra) []string {
	flags := []string{}
	for _, e := range extras {
		flags = append(flags, e.Flags()...)
	}
	return flags
}

func (b *binding) resolveDriverPath(ctx context.Context, driver string) (Driver, error) {
	path, err := b.Shell("pm", "path", driver).Call(ctx)
	if err != nil {
		return Driver{}, err
	}
	// Check the package path of the driver.
	if !strings.HasPrefix(path, "package:") {
		return Driver{}, nil
	}
	path = path[8:]
	// If the driver package path doesn't have the /data/app prefix, it means the returned
	// path is the preinstalled emtpy prerelease driver. Hence don't return it.
	if path == "" || !strings.HasPrefix(path, "/data/app") {
		return Driver{}, nil
	}

	return Driver{
		Package: driver,
		Path:    path,
	}, nil
}

// GraphicsDriver queries and returns info about the prerelease graphics driver.
func (b *binding) GraphicsDriver(ctx context.Context) (Driver, error) {
	driver, err := b.SystemProperty(ctx, driverProperty)
	if err != nil {
		return Driver{}, err
	}
	if driver == "" {
		// There is no prerelease driver.
		return Driver{}, nil
	}
	return b.resolveDriverPath(ctx, driver)
}

// Returns the package version code of the graphics driver
func (b *binding) DriverVersionCode(ctx context.Context) (int, error) {
	driver, err := b.SystemProperty(ctx, driverProperty)
	if err != nil {
		return 0, err
	}
	ip, err := b.InstalledPackage(ctx, driver)
	if err != nil {
		return 0, err
	}
	return ip.VersionCode, err
}

// PrepareGpuProfiling implements the adb.Device interface.
func (b *binding) PrepareGpuProfiling(ctx context.Context, installedPackage *android.InstalledPackage) (bool, string, app.Cleanup, error) {
	driver, err := b.GraphicsDriver(ctx)
	if err != nil {
		// If there's an error, keep going to attempt to use GPU profiling
		// libraries in system image.
		log.W(ctx, "Failed to query developer driver: %v, assuming no developer driver found.", err)
	}

	if driver.Package != "" {
		log.I(ctx, "Using GPU profiling libraries from developer driver package: %v.", driver.Package)

		// Set up device info service to use prerelease driver.
		cleanup, err := SetupPrereleaseDriver(ctx, b, installedPackage)
		if err != nil {
			return false, "", cleanup, err
		}
		return true, driver.Package, cleanup, nil
	}

	// For Android 11, GPU profiling could be part of the system driver. It can be implemented as
	// a vulkan layer. Hence check whether GPU profiling is supported as part of the system driver,
	// and append the vulkan profiling layer apk package name as a place to discover the vulkan
	// profiling library.
	log.I(ctx, "No developer driver found, attempting to use GPU profiling libraries in system image.")

	supported, err := b.SystemProperty(ctx, systemImageGpuProfilerSupportProperty)
	if err != nil {
		return false, "", nil, err
	}
	if supported != "true" {
		return false, "", nil, nil
	}

	packageName, err := b.SystemProperty(ctx, gpuProfilerVulkanLayerApkProperty)
	if err != nil {
		return false, "", nil, err
	}
	return true, packageName, nil, nil
}
