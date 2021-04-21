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

package client

import (
	"context"
	"net"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapidapk"
	"github.com/pkg/errors"
)

const (
	// getPidRetries is the number of retries for getting the pid of the process
	// our newly-started activity runs in.
	getPidRetries = 7

	// captureProcessNameProperty is the Android system property holding the
	// name of the process to capture. Mirrored in gapii/cc/spy.cpp.
	captureProcessNameProperty = "debug.agi.procname"
)

// Process represents a running process to capture.
type Process struct {
	// The local host port used to connect to GAPII.
	Port int

	// Information about the target device.
	Device bind.Device

	// The options used for the capture.
	Options Options

	// The connection
	conn net.Conn
}

// Start launches an activity on an android device with the GAPII interceptor
// enabled using the gapid.apk built for the ABI matching the specified action and device.
// GAPII will attempt to connect back on the returned host port to write the trace.
func Start(ctx context.Context, p *android.InstalledPackage, a *android.ActivityAction, o Options) (*Process, app.Cleanup, error) {
	ctx = log.Enter(ctx, "start")
	if a != nil {
		ctx = log.V{"activity": a.Activity}.Bind(ctx)
	}
	ctx = log.V{"on": p.Name}.Bind(ctx)
	d := p.Device.(adb.Device)

	abi := p.ABI
	if abi.SameAs(device.UnknownABI) {
		abi = p.Device.Instance().GetConfiguration().PreferredABI(nil)
	}

	if a == nil || a.Package == nil {
		return nil, nil, log.Err(ctx, nil, "Invalid activity information.")
	}

	_, _, cleanup, err := d.PrepareGpuProfiling(ctx, a.Package)
	if err != nil {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to prepare GPU profiling")
	}

	// TODO: Need to clean this up and get it working
	// If Angle was selected, then save current angle settings to restore and then set Angle for selected app
	log.I(ctx, "Checking if ANGLE requested")
	if o.EnableAngle {
		log.I(ctx, "ANGLE is enabled")
		if anglePackage := p.Device.Instance().GetConfiguration().GetAngle().GetPackage(); anglePackage != "" {
			log.I(ctx, "Found ANGLE package %s, enabling it for %s", anglePackage, p.Name)
			nextCleanup, err := adb.SetupAngle(ctx, d, a.Package)
			cleanup = cleanup.Then(nextCleanup)
			if err != nil {
				return nil, cleanup.Invoke(ctx), err
			}
		} else {
			// We shouldn't be able to get here. EnableAngle flag should only be set
			// when there's an Angle package on the device to enable.
			return nil, cleanup.Invoke(ctx), log.Err(ctx, nil, "ANGLE requested but no ANGLE package found")
		}
	}

	// For NativeBridge emulated devices opt for the native ABI of the emulator.
	abi = d.NativeBridgeABI(ctx, abi)

	ctx = log.V{"abi": abi.Name}.Bind(ctx)

	log.I(ctx, "Unlocking device screen")
	unlocked, err := d.UnlockScreen(ctx)
	if err != nil {
		log.W(ctx, "Failed to determine lock state: %s", err)
	} else if !unlocked {
		return nil, nil, log.Err(ctx, nil, "Please unlock your device screen: GAPID can automatically unlock the screen only when no PIN/password/pattern is needed")
	}

	port, err := adb.LocalFreeTCPPort()
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Finding free port for gapii")
	}

	log.I(ctx, "Checking gapid.apk is installed")
	_, err = gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Installing gapid.apk")
	}

	ctx = log.V{"port": port}.Bind(ctx)

	log.I(ctx, "Forwarding")
	pipe := "gapii"
	if o.PipeName != "" {
		pipe = o.PipeName
	}
	if err := d.Forward(ctx, adb.TCPPort(port), adb.NamedAbstractSocket(pipe)); err != nil {
		return nil, nil, log.Err(ctx, err, "Setting up port forwarding for gapii")
	}
	cleanup = cleanup.Then(func(ctx context.Context) {
		d.RemoveForward(ctx, port)
	})

	if !android.SupportsVulkanLayersViaSystemSettings(d) {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, nil, "Cannot trace without layers support")
	}

	log.I(ctx, "Setting up Layer")
	cu, err := android.SetupLayers(ctx, d, p.Name, []string{gapidapk.PackageName(abi)}, []string{gapidapk.GraphicsSpyLayerName})
	if err != nil {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Failed when setting up layers")
	}
	cleanup = cleanup.Then(cu)

	var additionalArgs []android.ActionExtra
	if o.AdditionalFlags != "" {
		additionalArgs = append(additionalArgs, android.CustomExtras(text.Quote(text.SplitArgs(o.AdditionalFlags))))
	}

	log.I(ctx, "Setting capture process name")
	procNameBackup, err := d.SystemProperty(ctx, captureProcessNameProperty)
	if err != nil {
		return nil, cleanup.Invoke(ctx), err
	}
	cleanup = cleanup.Then(func(ctx context.Context) {
		d.SetSystemProperty(ctx, captureProcessNameProperty, procNameBackup)
	})
	if err := d.SetSystemProperty(ctx, captureProcessNameProperty, o.ProcessName); err != nil {
		return nil, cleanup.Invoke(ctx), err
	}

	if a != nil {
		log.I(ctx, "Starting activity")
		if err := d.StartActivity(ctx, *a, additionalArgs...); err != nil {
			return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Starting activity")
		}
	} else {
		log.I(ctx, "No start activity selected - trying to attach...")
	}

	var pid int
	err = android.ErrProcessNotFound
	for attempt := 0; attempt <= getPidRetries && errors.Cause(err) == android.ErrProcessNotFound; attempt++ {
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		pid, err = p.Pid(ctx)
	}
	if err != nil {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Getting pid")
	}
	ctx = log.V{"pid": pid}.Bind(ctx)

	process := &Process{
		Port:    int(port),
		Device:  d,
		Options: o,
	}

	return process, cleanup, nil
}

// Connect connects to an app that is already setup to trace. This is similar to
// Start(...), except that it skips some steps as it is assumed that the loading
// of libgapii is done manually and the app is waiting for a connection from the
// host.
func Connect(ctx context.Context, d adb.Device, abi *device.ABI, pipe string, o Options) (*Process, error) {
	port, err := adb.LocalFreeTCPPort()
	if err != nil {
		return nil, log.Err(ctx, err, "Finding free port for gapii")
	}
	ctx = log.V{"port": port}.Bind(ctx)

	log.I(ctx, "Checking gapid.apk is installed")
	_, err = gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return nil, log.Err(ctx, err, "Installing gapid.apk")
	}

	log.I(ctx, "Forwarding")
	if err := d.Forward(ctx, adb.TCPPort(port), adb.NamedAbstractSocket(pipe)); err != nil {
		return nil, log.Err(ctx, err, "Setting up port forwarding for gapii")
	}

	process := &Process{
		Port:    int(port),
		Device:  d,
		Options: o,
	}
	if err := process.connect(ctx); err != nil {
		return nil, err
	}
	return process, nil
}
