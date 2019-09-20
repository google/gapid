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
	"strings"
	"time"

	perfetto_pb "perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell"
)

const (
	// ErrDeviceNotRooted is returned by Device.Root when the device is running a
	// production build as is not 'rooted'.
	ErrDeviceNotRooted = fault.Const("Device is not rooted")
	ErrRootFailed      = fault.Const("Device failed to switch to root")

	maxRootAttempts                            = 5
	gpuRenderingStagesDataSourceDescriptorName = "gpu.renderingstages"
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

func (b *binding) QueryPerfettoServiceState(ctx context.Context) (*device.PerfettoCapability, error) {
	if b.Instance().GetConfiguration().GetOS().GetAPIVersion() < 29 {
		return &device.PerfettoCapability{}, log.Errf(ctx, nil, "Querying perfetto capability requires Android API >= 29")
	}
	res, err := b.Shell("perfetto", "--query-raw", "|", "base64").Call(ctx)
	if err != nil {
		return &device.PerfettoCapability{}, log.Errf(ctx, err, "adb shell perfetto returned error: %s", res)
	}
	decoded, _ := base64.StdEncoding.DecodeString(res)
	tracingServiceState := &perfetto_pb.TracingServiceState{}
	if err = proto.Unmarshal(decoded, tracingServiceState); err != nil {
		return &device.PerfettoCapability{}, log.Errf(ctx, err, "Unmarshal returned error")
	}
	perfettoCapability := &device.PerfettoCapability{
		GpuProfiling: &device.GPUProfiling{},
	}
	dataSources := tracingServiceState.GetDataSources()
	for _, dataSource := range dataSources {
		dataSourceDescriptor := dataSource.GetDsDescriptor()
		if dataSourceDescriptor.GetName() == gpuRenderingStagesDataSourceDescriptorName {
			perfettoCapability.GpuProfiling.HasRenderStage = true
			continue
		}
		gpuCounterDescriptor := dataSourceDescriptor.GetGpuCounterDescriptor()
		specs := gpuCounterDescriptor.GetSpecs()
		if len(specs) == 0 || perfettoCapability.GpuProfiling.GpuCounterDescriptor != nil {
			continue
		}

		// We mirror the Perfetto GpuCounterDescriptor proto into GAPID, hence
		// they are binary format compatible.
		data, err := proto.Marshal(gpuCounterDescriptor)
		if err != nil {
			continue
		}
		deviceGpuCounterDescriptor := &device.GpuCounterDescriptor{}
		if err = proto.Unmarshal(data, deviceGpuCounterDescriptor); err != nil {
			continue
		}
		perfettoCapability.GpuProfiling.GpuCounterDescriptor = deviceGpuCounterDescriptor
	}

	return perfettoCapability, nil
}

func extrasFlags(extras []android.ActionExtra) []string {
	flags := []string{}
	for _, e := range extras {
		flags = append(flags, e.Flags()...)
	}
	return flags
}
