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

	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
)

// Device extends the android.Device interface with adb specific features.
type Device interface {
	android.Device
	// Command is a helper that builds a shell.Cmd with the device as its target.
	Command(name string, args ...string) shell.Cmd
	// Root restarts adb as root. If the device is running a production build then
	// Root will return ErrDeviceNotRooted.
	Root(ctx context.Context) error
	// IsDebuggableBuild returns true if the device runs a debuggable Android build.
	IsDebuggableBuild(ctx context.Context) (bool, error)
	// Forward will forward the specified device Port to the specified local Port.
	Forward(ctx context.Context, local, device Port) error
	// RemoveForward removes a port forward made by Forward.
	RemoveForward(ctx context.Context, local Port) error
	// GraphicsDriver queries and returns info about the prerelease graphics driver.
	GraphicsDriver(ctx context.Context) (Driver, error)
	// HasGpuProfilingSupportInSystemImage returns whether system image has GPU profiling support.
	HasGpuProfilingSupportInSystemImage(ctx context.Context) (bool, error)
	// GetGpuProfilingLayerPackageName queries and returns the package name of the apk that contains
	// the GPU profiling Vulkan layer.
	GetGpuProfilingLayerPackageName(ctx context.Context) (string, error)
}

// Driver contains the information about a graphics driver.
type Driver struct {
	Package string
	Path    string
}

// DeviceList is a list of devices.
type DeviceList []Device

// FindBySerial returns the device with the matching serial, or nil if the
// device cannot be found.
func (l DeviceList) FindBySerial(serial string) Device {
	for _, d := range l {
		if d.Instance().Serial == serial {
			return d
		}
	}
	return nil
}

// binding represents an attached Android device.
type binding struct {
	bind.Simple
}

// verify that binding implements Device
var _ Device = (*binding)(nil)
