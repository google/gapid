// Copyright (C) 2019 Google Inc.
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

package android

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"

	perfetto_pb "protos/perfetto/config"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapis/service"
)

const (
	gpuRenderStagesDataSourceDescriptorName = "gpu.renderstages"

	// perfettoTraceFile is the location on the device where we'll ask Perfetto
	// to store the trace data while tracing.
	perfettoTraceFile                       = "/data/misc/perfetto-traces/gapis-trace"
	prereleaseDriverProperty                = "ro.gfx.driver.1"
	prereleaseDriverOverrideSettingVariable = "gapid.driver_package_override"
	prereleaseDriverSettingVariable         = "game_driver_prerelease_opt_in_apps"
	renderStageVulkanLayerName              = "VkRenderStagesProducer"
)

// Process represents a running Perfetto capture.
type Process struct {
	device   adb.Device
	config   *perfetto_pb.TraceConfig
	deferred bool
}

func hasRenderStageEnabled(perfettoConfig *perfetto_pb.TraceConfig) bool {
	for _, dataSource := range perfettoConfig.GetDataSources() {
		if dataSource.Config.GetName() == gpuRenderStagesDataSourceDescriptorName {
			return true
		}
	}
	return false
}

func setupVulkanLayers(ctx context.Context, d adb.Device, packageName string, abi *device.ABI, layers []string) (app.Cleanup, error) {
	packages := []string{gapidapk.PackageName(abi)}

	cleanup, err := android.SetupLayers(ctx, d, packageName, packages, layers, true)
	if err != nil {
		return cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to setup gpu.renderstages environment.")
	}
	return cleanup, nil
}

func getDriverPackageName(ctx context.Context, d adb.Device) (string, error) {
	driverPackageName, err := d.SystemProperty(ctx, prereleaseDriverProperty)
	if err != nil {
		return "", err
	}
	if driverPackageOverride, err := d.SystemSetting(ctx, "global", prereleaseDriverOverrideSettingVariable); driverPackageOverride != "" && driverPackageOverride != "null" && err == nil {
		driverPackageName = driverPackageOverride
	}
	if driverPackageName == "" {
		return "", nil
	}
	if res, err := d.Shell("pm", "path", driverPackageName).Call(ctx); err != nil || res == "" {
		return "", log.Err(ctx, err, "No driver package found.")
	}
	return driverPackageName, nil
}

// SetupProfileLayersSource configures the device to allow packages to be used as layer sources for profiling
func SetupProfileLayersSource(ctx context.Context, d adb.Device, packageName string, abi *device.ABI) (app.Cleanup, error) {
	driverPackageName, err := getDriverPackageName(ctx, d)
	if err != nil {
		return nil, err
	}
	packages := []string{driverPackageName}
	if abi != nil {
		packages = append(packages, gapidapk.PackageName(abi))
	}

	cleanup, err := android.SetupLayers(ctx, d, packageName, packages, []string{}, true)
	if err != nil {
		return cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to setup gpu.renderstages layer packages.")
	}
	return cleanup, nil
}

// SetupProfileLayers configures the device to allow the app being traced to load the layers required for render stage profiling
func SetupProfileLayers(ctx context.Context, d adb.Device, packageName string, hasRenderStages bool, abi *device.ABI, layers []string) (app.Cleanup, error) {
	if !hasRenderStages {
		if abi != nil {
			return setupVulkanLayers(ctx, d, packageName, abi, layers)
		}
		return nil, nil
	}
	driverPackageName, err := getDriverPackageName(ctx, d)
	if err != nil {
		return nil, err
	}
	packages := []string{driverPackageName}
	enabledLayers := []string{renderStageVulkanLayerName}
	if abi != nil {
		packages = append(packages, gapidapk.PackageName(abi))
		enabledLayers = append(enabledLayers, layers...)
	}

	cleanup, err := android.SetupLayers(ctx, d, packageName, packages, enabledLayers, true)
	if err != nil {
		return cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to setup gpu.renderstages environment.")
	}
	return cleanup, nil
}

func setupPrereleaseDriver(ctx context.Context, d adb.Device, p *android.InstalledPackage) (app.Cleanup, error) {
	driverPackageName, err := d.SystemProperty(ctx, prereleaseDriverProperty)
	if err != nil {
		return nil, err
	}
	if driverPackageName == "" {
		return nil, nil
	}
	oldOptinApps, err := d.SystemSetting(ctx, "global", prereleaseDriverSettingVariable)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to get prerelease driver opt in apps.")
	}
	if strings.Contains(oldOptinApps, p.Name) {
		return nil, nil
	}
	newOptinApps := oldOptinApps + "," + p.Name
	// TODO(b/145893290) Check whether application has developer driver enabled once b/145893290 is fixed.
	if err := d.SetSystemSetting(ctx, "global", prereleaseDriverSettingVariable, newOptinApps); err != nil {
		return nil, log.Errf(ctx, err, "Failed to set up prerelease driver for app: %v.", p.Name)
	}
	return func(ctx context.Context) {
		d.SetSystemSetting(ctx, "global", prereleaseDriverSettingVariable, oldOptinApps)
	}, nil
}

// Start optional starts an app and sets up a Perfetto trace
func Start(ctx context.Context, d adb.Device, a *android.ActivityAction, opts *service.TraceOptions, abi *device.ABI, layers []string) (*Process, app.Cleanup, error) {
	ctx = log.Enter(ctx, "start")

	var cleanup app.Cleanup
	if abi != nil {
		_, err := gapidapk.EnsureInstalled(ctx, d, abi)
		if err != nil {
			return nil, cleanup, err
		}
	}
	if a != nil {
		ctx = log.V{
			"package":  a.Package.Name,
			"activity": a.Activity,
		}.Bind(ctx)
		// Before start the activity, attempt to setup prerelease driver.
		nextCleanup, err := setupPrereleaseDriver(ctx, d, a.Package)
		cleanup = cleanup.Then(nextCleanup)
		if err != nil {
			return nil, cleanup.Invoke(ctx), err
		}
		// Before we start the activity, attempt to setup environment if gpu.renderstages is selected.
		hasRenderStages := hasRenderStageEnabled(opts.PerfettoConfig)
		nextCleanup, err = SetupProfileLayers(ctx, d, a.Package.Name, hasRenderStages, abi, layers)
		cleanup = cleanup.Then(nextCleanup)
		if err != nil {
			return nil, cleanup.Invoke(ctx), err
		}
	}

	log.I(ctx, "Unlocking device screen")
	unlocked, err := d.UnlockScreen(ctx)
	if err != nil {
		log.W(ctx, "Failed to determine lock state: %s", err)
	} else if !unlocked {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, nil, "Please unlock your device screen: GAPID can automatically unlock the screen only when no PIN/password/pattern is needed")
	}

	if a != nil {
		if err := d.StartActivity(ctx, *a); err != nil {
			return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Starting the activity")
		}
	}

	return &Process{
		device:   d,
		config:   opts.PerfettoConfig,
		deferred: opts.DeferStart,
	}, cleanup, nil
}

// Capture starts the perfetto capture.
func (p *Process) Capture(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64) (int64, error) {
	tmp, err := file.Temp()
	if err != nil {
		return 0, log.Err(ctx, err, "Failed to create a temp file")
	}

	// Signal that we are ready to start.
	atomic.StoreInt64(written, 1)

	if p.deferred && !start.Wait(ctx) {
		return 0, log.Err(ctx, nil, "Cancelled")
	}

	if err := p.device.StartPerfettoTrace(ctx, p.config, perfettoTraceFile, stop, ready); err != nil {
		return 0, err
	}

	if err := p.device.Pull(ctx, perfettoTraceFile, tmp.System()); err != nil {
		return 0, err
	}

	if err := p.device.RemoveFile(ctx, perfettoTraceFile); err != nil {
		log.E(ctx, "Failed to delete perfetto trace file %v", err)
	}

	size := tmp.Info().Size()
	atomic.StoreInt64(written, size)
	fh, err := os.Open(tmp.System())
	if err != nil {
		return 0, log.Err(ctx, err, fmt.Sprintf("Failed to open %s", tmp))
	}
	return io.Copy(w, fh)
}
