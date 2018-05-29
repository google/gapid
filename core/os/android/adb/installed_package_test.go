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
	"github.com/google/gapid/core/os/device"
)

func TestParsePackages(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "dumpsys_device")

	p0 := &android.InstalledPackage{
		Name:           "com.google.foo",
		ABI:            device.AndroidARMv7a,
		Device:         d,
		VersionCode:    902107,
		MinSDK:         14,
		TargetSdk:      15,
		ServiceActions: android.ServiceActions{},
	}
	p0.ActivityActions = android.ActivityActions{
		{
			Package:  p0,
			Name:     "android.intent.action.MAIN",
			Activity: "com.google.foo.FooActivity",
		}, {
			Package:  p0,
			Name:     "com.google.android.FOO",
			Activity: "com.google.foo.FooActivity",
		}, {
			Package:  p0,
			Name:     "android.intent.action.SEARCH",
			Activity: "com.google.foo.FooActivity",
		},
	}

	p1 := &android.InstalledPackage{
		Name:           "com.google.qux",
		ABI:            device.AndroidARMv7a,
		Device:         d,
		Debuggable:     true,
		VersionCode:    123456,
		MinSDK:         0,
		TargetSdk:      15,
		ServiceActions: android.ServiceActions{},
	}
	p1.ActivityActions = android.ActivityActions{
		{
			Package:  p1,
			Name:     "android.intent.action.MAIN",
			Activity: "com.google.qux.QuxActivity",
		},
	}

	expected := android.InstalledPackages{p0, p1}
	packages, err := d.InstalledPackages(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "pkgs").That(packages).DeepEquals(expected)
}

func TestPid(t_ *testing.T) {
	ctx := log.Testing(t_)

	get := func(dev string, pkg string) (int, error) {
		p := &android.InstalledPackage{Name: pkg, Device: mustConnect(ctx, dev)}
		return p.Pid(ctx)
	}

	_, err := get("no_pgrep_no_ps_device", "com.google.bar")
	assert.For(ctx, "err").ThatError(err).Failed()

	pid, err := get("no_pgrep_ok_ps_device", "com.google.bar")
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "pid").That(pid).Equals(2778)

	pid, err = get("ok_pgrep_no_ps_device", "com.google.bar")
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "pid").That(pid).Equals(2778)

	pid, err = get("ok_pgrep_ok_ps_device", "com.google.foo")
	assert.For(ctx, "err").ThatError(err).Equals(android.ErrProcessNotFound)

	pid, err = get("no_pgrep_ok_ps_device", "com.google.foo")
	assert.For(ctx, "err").ThatError(err).Equals(android.ErrProcessNotFound)
}
