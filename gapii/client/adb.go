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
	"time"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/cause"
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
func Start(ctx log.Context, a *android.ActivityAction) (port adb.TCPPort, cleanup task.Task, err error) {
	p := a.Package
	ctx = ctx.Enter("start").S("activity", a.Activity).S("on", p.Name)
	d := p.Device.(adb.Device)

	abi := a.Package.ABI
	if abi.SameAs(device.UnknownABI) {
		abi = a.Package.Device.Instance().GetConfiguration().PreferredABI(nil)
	}
	ctx = ctx.V("abi", abi.Name)

	ctx.Print("Turning device screen on")
	if err := d.TurnScreenOn(ctx); err != nil {
		return 0, nil, cause.Explain(ctx, err, "Couldn't turn device screen on")
	}

	ctx.Print("Checking for lockscreen")
	locked, err := d.IsShowingLockscreen(ctx)
	if err != nil {
		jot.Warning(ctx).Cause(err).Print("Couldn't determine lockscreen state")
	}
	if locked {
		return 0, nil, cause.Explain(ctx, nil, "Cannot trace app on locked device")
	}

	port, err = adb.LocalFreeTCPPort()
	if err != nil {
		return 0, nil, cause.Explain(ctx, err, "Finding free port")
	}

	ctx.Print("Checking gapid.apk is installed")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return 0, nil, cause.Explain(ctx, err, "Installing gapid.apk")
	}

	ctx = ctx.I("port", int(port))

	ctx.Print("Forwarding")
	if err := d.Forward(ctx, adb.TCPPort(port), adb.NamedAbstractSocket("gapii")); err != nil {
		return 0, nil, cause.Explain(ctx, err, "Setting up port forwarding")
	}

	// FileDir may fail here. This happens if/when the app is non-debuggable.
	// Don't set up vulkan tracing here, since the loader will not try and load the layer
	// if we aren't debuggable regardless.
	if err := d.Command("shell", "setprop", "debug.vulkan.layers", "VkGraphicsSpy").Run(ctx); err != nil {
		d.RemoveForward(ctx, adb.TCPPort(port))
		return 0, nil, cause.Explain(ctx, err, "Setting up vulkan layer")
	}

	doCleanup := func(ctx log.Context) error {
		d.Command("shell", "setprop", "debug.vulkan.layers", "\"\"").Run(ctx)
		return d.RemoveForward(ctx, adb.TCPPort(port))
	}
	defer func() {
		if err != nil {
			doCleanup(ctx)
		}
	}()

	ctx.Print("Starting activity in debug mode")
	if err := d.StartActivityForDebug(ctx, *a); err != nil {
		return 0, nil, cause.Explain(ctx, err, "Starting activity in debug mode")
	}

	var pid int
	err = android.ErrProcessNotFound
	for attempt := 0; attempt <= getPidRetries && errors.Cause(err) == android.ErrProcessNotFound; attempt++ {
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		pid, err = p.Pid(ctx)
	}
	if err != nil {
		return 0, nil, cause.Explain(ctx, err, "Getting pid")
	}
	ctx = ctx.I("pid", pid)

	if err := loadLibrariesViaJDWP(ctx, apk, pid, d); err != nil {
		return 0, nil, err
	}

	return port, doCleanup, nil
}
