// Copyright (C) 2021 Google Inc.
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

package ffx

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/fuchsia"
	"github.com/google/gapid/core/os/shell"
)

const (
	// Frequency at which to print scan errors.
	printScanErrorsEveryNSeconds = 120
	// ErrNoDeviceList May be returned if the ffx fails to return a device list when asked.
	ErrNoDeviceList = fault.Const("Device list not returned")
	// ErrInvalidDeviceList May be returned if the device list could not be parsed.
	ErrInvalidDeviceList = fault.Const("Could not parse device list")
	// Tracing file couldn't be written to.
	ErrTraceFilePermissions = fault.Const("Tracing file permissions")
	// Start tracing command failed.
	ErrStartTrace = fault.Const("Start trace failed")
	// Trace providers command failed.
	ErrTraceProviders = fault.Const("Trace providers failed")
	// Trace providers format.
	ErrTraceProvidersFormat = fault.Const("Trace providers format")
)

var (
	// cache is a map of device serials to fully resolved bindings.
	cache      = map[string]*binding{}
	cacheMutex sync.Mutex // Guards cache.

	// Registry of all the discovered devices.
	registry = bind.NewRegistry()
)

// DeviceList is a list of devices.
type DeviceList []fuchsia.Device

// Devices returns the list of attached Fuchsia devices.
func Devices(ctx context.Context) (DeviceList, error) {
	if err := scanDevices(ctx); err != nil {
		return nil, err
	}
	devs := registry.Devices()
	deviceList := make(DeviceList, len(devs))
	for i, d := range devs {
		deviceList[i] = d.(fuchsia.Device)
	}
	return deviceList, nil
}

// Monitor updates the registry with devices that are added and removed at the
// specified interval. Monitor returns once the context is cancelled.
func Monitor(ctx context.Context, r *bind.Registry, interval time.Duration) error {
	unlisten := registry.Listen(bind.NewDeviceListener(r.AddDevice, r.RemoveDevice))
	defer unlisten()

	for _, d := range registry.Devices() {
		r.AddDevice(ctx, d)
	}

	lastErrorPrinted := time.Now()
	for {
		err := scanDevices(ctx)
		if err != nil {
			if time.Since(lastErrorPrinted).Seconds() > printScanErrorsEveryNSeconds {
				log.E(ctx, "Couldn't scan devices: %v", err)
				lastErrorPrinted = time.Now()
			}
		} else {
			lastErrorPrinted = time.Time{}
		}

		select {
		case <-task.ShouldStop(ctx):
			return nil
		case <-time.After(interval):
		}
	}
}

func newDevice(ctx context.Context, serial string) (*binding, error) {
	log.I(ctx, "Fuchsia Serial: %v", serial)
	d := &binding{
		Simple: bind.Simple{
			To: &device.Instance{
				Serial: serial,
				// TODO: change "Fuchsia" to a device-specific name.
				Name:          "Fuchsia",
				Configuration: &device.Configuration{OS: &device.OS{Kind: device.Fuchsia}},
			},
			LastStatus: bind.Online,
		},
	}

	// TODO: fill in d.To.Configuration defined in device.proto

	d.Instance().GenID()

	return d, nil
}

// scanDevices returns the list of attached Fuchsia devices.
func scanDevices(ctx context.Context) error {
	exe, err := ffx()
	if err != nil {
		return log.Err(ctx, err, "")
	}
	stdout, err := shell.Command(exe.System(), "target", "list", "-f", "simple").Call(ctx)
	if err != nil {
		return err
	}
	if strings.Contains(stdout, "No devices found") {
		return ErrNoDeviceList
	}
	parsed, err := ParseDevices(ctx, stdout)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	for serial, _ := range parsed {
		if _, ok := cache[serial]; !ok {
			device, err := newDevice(ctx, serial)
			if err != nil {
				return err
			}
			cache[serial] = device
			registry.AddDevice(ctx, device)
		}
	}

	// Remove cached results for removed devices.
	for serial, cached := range cache {
		if _, found := parsed[serial]; !found {
			delete(cache, serial)
			registry.RemoveDevice(ctx, cached)
		}
	}

	return nil
}

func ParseDevices(ctx context.Context, out string) (map[string]bind.Status, error) {
	lines := strings.Split(out, "\n")
	devices := make(map[string]bind.Status, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		switch len(fields) {
		case 0:
			continue
		case 2:
			_, serial := fields[0], fields[1]
			devices[serial] = bind.Online
		case 3:
			if strings.ToUpper(line) != "NO DEVICES FOUND." {
				return nil, ErrInvalidDeviceList
			}
			return devices, nil
		default:
			return nil, ErrInvalidDeviceList
		}
	}
	return devices, nil
}
