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
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/flock"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapidapk"
	"github.com/pkg/errors"
)

const (
	// getPidRetries is the number of retries for getting the pid of the process
	// our newly-started activity runs in.
	getPidRetries = 7
	// vkImplicitLayersProp is the name of the system property that contains implicit
	// Vulkan layers to be loaded by Vulkan loader on Android
	vkImplicitLayersProp = "debug.vulkan.layers"
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
func Start(ctx context.Context, p *android.InstalledPackage, a *android.ActivityAction, o Options) (*Process, error) {
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

	// For NativeBridge emulated devices opt for the native ABI of the emulator.
	abi = d.NativeBridgeABI(ctx, abi)

	ctx = log.V{"abi": abi.Name}.Bind(ctx)

	log.I(ctx, "Turning device screen on")
	if err := d.TurnScreenOn(ctx); err != nil {
		return nil, log.Err(ctx, err, "Couldn't turn device screen on")
	}

	log.I(ctx, "Checking for lockscreen")
	locked, err := d.IsShowingLockscreen(ctx)
	if err != nil {
		log.W(ctx, "Couldn't determine lockscreen state: %v", err)
	}
	if locked {
		return nil, log.Err(ctx, nil, "Cannot trace app on locked device")
	}

	port, err := adb.LocalFreeTCPPort()
	if err != nil {
		return nil, log.Err(ctx, err, "Finding free port for gapii")
	}

	log.I(ctx, "Checking gapid.apk is installed")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return nil, log.Err(ctx, err, "Installing gapid.apk")
	}

	ctx = log.V{"port": port}.Bind(ctx)

	log.I(ctx, "Forwarding")
	pipe := "gapii"
	if o.PipeName != "" {
		pipe = o.PipeName
	}
	if err := d.Forward(ctx, adb.TCPPort(port), adb.NamedAbstractSocket(pipe)); err != nil {
		return nil, log.Err(ctx, err, "Setting up port forwarding for gapii")
	}

	// FileDir may fail here. This happens if/when the app is non-debuggable.
	// Don't set up vulkan tracing here, since the loader will not try and load the layer
	// if we aren't debuggable regardless.
	var m *flock.Mutex
	if o.APIs&VulkanAPI != uint32(0) {
		m, err = reserveVulkanDevice(ctx, d)
		if err != nil {
			d.RemoveForward(ctx, port)
			return nil, log.Err(ctx, err, "Setting up for tracing Vulkan")
		}
	}

	app.AddCleanup(ctx, func() {
		d.RemoveForward(ctx, port)
		releaseVulkanDevice(ctx, d, m)
	})

	var additionalArgs []android.ActionExtra
	if o.AdditionalFlags != "" {
		additionalArgs = append(additionalArgs, android.CustomExtras(text.Quote(text.SplitArgs(o.AdditionalFlags))))
	}

	if a != nil {
		log.I(ctx, "Starting activity in debug mode")
		if err := d.StartActivityForDebug(ctx, *a, additionalArgs...); err != nil {
			return nil, log.Err(ctx, err, "Starting activity in debug mode")
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
		return nil, log.Err(ctx, err, "Getting pid")
	}
	ctx = log.V{"pid": pid}.Bind(ctx)

	process := &Process{
		Port:    int(port),
		Device:  d,
		Options: o,
	}
	if err := process.loadAndConnectViaJDWP(ctx, apk, pid, d); err != nil {
		return nil, err
	}

	return process, nil
}

// reserveVulkanDevice reserves the given device for starting Vulkan trace and
// set the implicit Vulkan layers property to let the Vulkan loader loads
// VkGraphicsSpy layer. It returns the mutex which reserves the device and error.
func reserveVulkanDevice(ctx context.Context, d adb.Device) (*flock.Mutex, error) {
	m := flock.Lock(d.Instance().GetSerial())
	if err := d.SetSystemProperty(ctx, vkImplicitLayersProp, "VkGraphicsSpy"); err != nil {
		return nil, log.Err(ctx, err, "Setting up vulkan layer")
	}
	return m, nil
}

// releaseVulkanDevice checks if the given mutex is nil, and if not, unsets the
// implicit Vulkan layers property on the given Android device and release the
// lock in the given mutex.
func releaseVulkanDevice(ctx context.Context, d adb.Device, m *flock.Mutex) error {
	if m != nil {
		ctx = keys.Clone(context.Background(), ctx)
		if err := d.SetSystemProperty(ctx, vkImplicitLayersProp, ""); err != nil {
			return err
		}
		m.Unlock()
	}
	return nil
}
