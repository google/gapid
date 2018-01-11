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

package job

import (
	"context"
	"sync"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
)

var devReg *bind.Registry
var deviceLocks map[string]*sync.Mutex
var mu sync.Mutex

// BindRegistry creates a new device registry to monitor new Android devices
// and returns the context which is bound to the created registry.
func BindRegistry(ctx context.Context) context.Context {
	devReg = bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, devReg)
	crash.Go(func() {
		if err := adb.Monitor(ctx, devReg, 15*time.Second); err != nil {
			log.E(ctx, "adb.Monitor failed: %v", err)
		}
	})
	return ctx
}

type deviceListener struct {
	devices  map[string]bind.Device
	onAdded  func(ctx context.Context, host *device.Instance, target bind.Device)
	registry *bind.Registry
}

// OnDeviceAdded registers onAdded to be called whenever a new Android device is
// added.
func OnDeviceAdded(ctx context.Context,
	onAdded func(ctx context.Context, host *device.Instance, target bind.Device)) {
	dl := &deviceListener{
		devices:  map[string]bind.Device{},
		onAdded:  onAdded,
		registry: bind.GetRegistry(ctx),
	}
	dl.registry.Listen(dl)
}

// OnDeviceAdded implements bind.DeviceListener interface
func (l *deviceListener) OnDeviceAdded(ctx context.Context, d bind.Device) {
	mu.Lock()
	defer mu.Unlock()
	host := host.Instance(ctx)
	inst := d.Instance()
	serial := inst.GetSerial()
	if _, ok := l.devices[serial]; !ok {
		log.I(ctx, "Device, added: %s", inst.GetName())
		l.onAdded(ctx, host, d)
	} else {
		l.devices[inst.GetSerial()] = d
	}
	if deviceLocks == nil {
		deviceLocks = map[string]*sync.Mutex{}
	}
	if _, ok := deviceLocks[serial]; !ok {
		deviceLocks[serial] = &sync.Mutex{}
	}
}

// OnDeviceRemoved implements bind.DeviceListener interface
func (l *deviceListener) OnDeviceRemoved(ctx context.Context, d bind.Device) {
	log.I(ctx, "Device removed: %s", d.Instance().GetName())
	// TODO: find a more graceful way to handle this.
}

// LockDevice reserves the given device for the requester, prevents the device
// to be used by other clients.
func LockDevice(ctx context.Context, d bind.Device) error {
	if _, ok := deviceLocks[d.Instance().GetSerial()]; !ok {
		return log.Errf(ctx, nil, "Lock not found for device: %v", d.Instance().GetSerial())
	}
	deviceLocks[d.Instance().GetSerial()].Lock()
	return nil
}

// UnlockDevices release the given device so other clients can run tasks on it.
func UnlockDevice(ctx context.Context, d bind.Device) error {
	if _, ok := deviceLocks[d.Instance().GetSerial()]; !ok {
		return log.Errf(ctx, nil, "Lock not found for device: %v", d.Instance().GetSerial())
	}
	deviceLocks[d.Instance().GetSerial()].Unlock()
	return nil
}
