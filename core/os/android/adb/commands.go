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
	"context"
	"strings"
	"time"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
)

const (
	// ErrDeviceNotRooted is returned by Device.Root when the device is running a
	// production build as is not 'rooted'.
	ErrDeviceNotRooted = fault.Const("Device is not rooted")
	ErrRootTimeout     = fault.Const("Timeout waiting for adbd root")
	ErrRootFailed      = fault.Const("Device failed to switch to root")

	timeoutWaitingForRoot = time.Second * 5
)

// Root restarts adb as root. If the device is running a production build then
// Root will return ErrDeviceNotRooted.
func (b *binding) Root(ctx context.Context) error {
	const expected = "adbd is already running as root"
	output, err := b.Command("root").Call(ctx)
	if err != nil {
		return err
	}
	switch output {
	case "adbd cannot run as root in production builds":
		return ErrDeviceNotRooted
	case "restarting adbd as root":
		start := time.Now()
		for time.Since(start) < timeoutWaitingForRoot {
			time.Sleep(time.Millisecond * 100)
			if output, _ := b.Command("root").Call(ctx); output == expected {
				return nil
			}
		}
		return log.Err(ctx, ErrRootTimeout, "")
	case expected:
		return nil
	default:
		return log.Err(ctx, ErrRootFailed, output)
	}
}

// InstallAPK installs the specified APK to the device. If reinstall is true
// and the package is already installed on the device then it will be replaced.
func (b *binding) InstallAPK(ctx context.Context, path string, reinstall bool, grantPermissions bool) error {
	args := []string{}
	if reinstall {
		args = append(args, "-r")
	}
	if grantPermissions && b.Instance().GetConfiguration().GetOS().GetMajor() >= 6 {
		// Starting with Android 6.0, permissions are not granted by default
		// during installation. Before Android 6.0, the flag did not exist.
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
		"-D", // Debug mode
		"-a", a.Name,
		"-n", a.Component(),
	}, extrasFlags(extras)...)
	return b.Shell("am", args...).Run(ctx)
}

// StartService launches the specified service action.
func (b *binding) StartService(ctx context.Context, a android.ServiceAction, extras ...android.ActionExtra) error {
	args := append([]string{
		"startservice",
		"-a", a.Name,
		"-n", a.Component(),
	}, extrasFlags(extras)...)
	return b.Shell("am", args...).Run(ctx)
}

func extrasFlags(extras []android.ActionExtra) []string {
	flags := []string{}
	for _, e := range extras {
		flags = append(flags, e.Flags()...)
	}
	return flags
}
