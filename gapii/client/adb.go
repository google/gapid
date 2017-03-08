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
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapidapk"
	"github.com/pkg/errors"
)

const (
	// getPidRetries is the number of retries for getting the pid of the process
	// our newly-started activity runs in.
	getPidRetries = 7
)

// Start launches an activity on an android device with the GAPII interceptor
// enabled using the gapid.apk built for the ABI matching the specified action
// and device.
// GAPII will attempt to connect back on the returned host port to write the
// trace.
func Start(ctx context.Context, a *android.ActivityAction) (port adb.TCPPort, cleanup task.Task, err error) {
	p := a.Package
	ctx = log.Enter(ctx, "start")
	ctx = log.V{"activity": a.Activity, "on": p.Name}.Bind(ctx)
	d := p.Device.(adb.Device)

	abi := a.Package.ABI
	if abi.SameAs(device.UnknownABI) {
		abi = a.Package.Device.Instance().GetConfiguration().PreferredABI(nil)
	}
	ctx = log.V{"abi": abi.Name}.Bind(ctx)

	log.I(ctx, "Turning device screen on")
	if err := d.TurnScreenOn(ctx); err != nil {
		return 0, nil, log.Err(ctx, err, "Couldn't turn device screen on")
	}

	log.I(ctx, "Checking for lockscreen")
	locked, err := d.IsShowingLockscreen(ctx)
	if err != nil {
		log.W(ctx, "Couldn't determine lockscreen state: %v", err)
	}
	if locked {
		return 0, nil, log.Err(ctx, nil, "Cannot trace app on locked device")
	}

	port, err = adb.LocalFreeTCPPort()
	if err != nil {
		return 0, nil, log.Err(ctx, err, "Finding free port")
	}

	log.I(ctx, "Checking gapid.apk is installed")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return 0, nil, log.Err(ctx, err, "Installing gapid.apk")
	}

	ctx = log.V{"port": port}.Bind(ctx)

	log.I(ctx, "Forwarding")
	if err := d.Forward(ctx, adb.TCPPort(port), adb.NamedAbstractSocket("gapii")); err != nil {
		return 0, nil, log.Err(ctx, err, "Setting up port forwarding")
	}

	// FileDir may fail here. This happens if/when the app is non-debuggable.
	// Don't set up vulkan tracing here, since the loader will not try and load the layer
	// if we aren't debuggable regardless.
	if err := d.Command("shell", "setprop", "debug.vulkan.layers", "VkGraphicsSpy").Run(ctx); err != nil {
		d.RemoveForward(ctx, adb.TCPPort(port))
		return 0, nil, log.Err(ctx, err, "Setting up vulkan layer")
	}

	doCleanup := func(ctx context.Context) error {
		d.Command("shell", "setprop", "debug.vulkan.layers", "\"\"").Run(ctx)
		return d.RemoveForward(ctx, adb.TCPPort(port))
	}
	defer func() {
		if err != nil {
			doCleanup(ctx)
		}
	}()

	log.I(ctx, "Starting activity in debug mode")
	if err := d.StartActivityForDebug(ctx, *a); err != nil {
		return 0, nil, log.Err(ctx, err, "Starting activity in debug mode")
	}

	var pid int
	err = android.ErrProcessNotFound
	for attempt := 0; attempt <= getPidRetries && errors.Cause(err) == android.ErrProcessNotFound; attempt++ {
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		pid, err = p.Pid(ctx)
	}
	if err != nil {
		return 0, nil, log.Err(ctx, err, "Getting pid")
	}
	ctx = log.V{"pid": pid}.Bind(ctx)

	if err := loadLibrariesViaJDWP(ctx, apk, pid, d); err != nil {
		return 0, nil, err
	}

	return port, doCleanup, nil
}
