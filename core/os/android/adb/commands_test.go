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

package adb_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
)

func TestRootProduction(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "production_device")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).Equals(adb.ErrDeviceNotRooted)
}

func TestRootDebug(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "debug_device")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
}

func TestRootDebug2(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "debug_device2")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
}

func TestRootRooted(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "rooted_device")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
}

func TestRootInvalid(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "invalid_device")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).HasMessage(`adb root gave output:
#0: not a normal response
#1: not a normal response
#2: not a normal response
#3: not a normal response
#4: not a normal response
   Cause: ` + adb.ErrRootFailed.Error())
}

func TestRootFailed(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "error_device")
	err := d.Root(ctx)
	assert.For(ctx, "err").ThatError(err).HasMessage(`Process returned error
   Cause: not a normal response`)
}

func TestInstallAPK(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "install_device")
	err := d.InstallAPK(ctx, "thing_to_install", false, false)
	expectedCommand(ctx, adbPath.System()+` -s install_device install thing_to_install`, err)
}

func TestInstallAPKWithPermissions(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "install_device")
	err := d.InstallAPK(ctx, "thing_to_install", false, true)
	expectedCommand(ctx, adbPath.System()+` -s install_device install -g thing_to_install`, err)
}

func TestReinstallAPK(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "install_device")
	err := d.InstallAPK(ctx, "thing_to_install", true, false)
	expectedCommand(ctx, adbPath.System()+` -s install_device install -r thing_to_install`, err)
}

func TestSELinuxEnforcing(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "production_device")
	got, err := d.SELinuxEnforcing(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "Device enforcing state").That(got).Equals(true)
}

func TestSELinuxNotEnforcing(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "debug_device")
	got, err := d.SELinuxEnforcing(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "Device enforcing state").That(got).Equals(false)
}

func TestSELinuxFailedEnforcing(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "error_device")
	_, err := d.SELinuxEnforcing(ctx)
	assert.For(ctx, "err").ThatError(err).HasMessage(`Process returned error
   Cause: not a normal response`)
}

func TestSetSELinuxEnforcing(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "install_device")
	err := d.SetSELinuxEnforcing(ctx, false)
	expectedCommand(ctx, adbPath.System()+` -s install_device shell setenforce 0`, err)
}

func TestClearSELinuxEnforcing(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "install_device")
	err := d.SetSELinuxEnforcing(ctx, true)
	expectedCommand(ctx, adbPath.System()+` -s install_device shell setenforce 1`, err)
}

func TestStartActivity(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "run_device")
	a := android.ActivityAction{
		Name: "android.intent.action.MAIN",
		Package: &android.InstalledPackage{
			Name: "com.google.test.AnApp",
		},
		Activity: "FooBarActivity",
	}
	err := d.StartActivity(ctx, a)
	expectedCommand(ctx, adbPath.System()+` -s run_device shell am start -S -W -a android.intent.action.MAIN -n com.google.test.AnApp/.FooBarActivity`, err)
}
