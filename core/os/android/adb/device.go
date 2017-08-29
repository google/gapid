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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
)

const (
	// ErrNoDeviceList May be returned if the adb fails to return a device list when asked.
	ErrNoDeviceList = fault.Const("Device list not returned")
	// ErrInvalidDeviceList May be returned if the device list could not be parsed.
	ErrInvalidDeviceList = fault.Const("Could not parse device list")
	// ErrInvalidStatus May be returned if the status string is not a known status.
	ErrInvalidStatus = fault.Const("Invalid status string")
)

var (
	// Each of the devInfoProviders are called each time a new device is found.
	// External packages can use this to add additional information to the
	// device.
	devInfoProviders      []DeviceInfoProvider
	devInfoProvidersMutex sync.Mutex

	// cache is a map of device serials to fully resolved bindings.
	cache      = map[string]*binding{}
	cacheMutex sync.Mutex // Guards cache.

	// Registry of all the discovered devices.
	registry = bind.NewRegistry()
)

// DeviceInfoProvider is a function that adds additional information to a
// Device.
type DeviceInfoProvider func(ctx context.Context, d Device) error

// RegisterDeviceInfoProvider registers f to be called to add additional
// information to a newly discovered Android device.
func RegisterDeviceInfoProvider(f DeviceInfoProvider) {
	devInfoProvidersMutex.Lock()
	defer devInfoProvidersMutex.Unlock()
	devInfoProviders = append(devInfoProviders, f)
}

// Monitor updates the registry with devices that are added and removed at the
// specified interval. Monitor returns once the context is cancelled.
func Monitor(ctx context.Context, r *bind.Registry, interval time.Duration) error {
	unlisten := registry.Listen(bind.NewDeviceListener(r.AddDevice, r.RemoveDevice))
	defer unlisten()

	for _, d := range registry.Devices() {
		r.AddDevice(ctx, d)
	}

	for {
		if err := scanDevices(ctx); err != nil {
			return log.Err(ctx, err, "Couldn't scan devices")
		}
		select {
		case <-task.ShouldStop(ctx):
			return nil
		case <-time.After(interval):
		}
	}
}

// Devices returns the list of attached Android devices.
func Devices(ctx context.Context) (DeviceList, error) {
	if err := scanDevices(ctx); err != nil {
		return nil, err
	}
	devs := registry.Devices()
	out := make(DeviceList, len(devs))
	for i, d := range devs {
		out[i] = d.(Device)
	}
	return out, nil
}

func newDevice(ctx context.Context, serial string, status bind.Status) (*binding, error) {
	d := &binding{
		Simple: bind.Simple{
			To: &device.Instance{
				Serial:        serial,
				Configuration: &device.Configuration{},
			},
			LastStatus: status,
		},
	}

	// Lookup the basic hardware information
	if res, err := d.Shell("getprop", "ro.build.product").Call(ctx); err == nil {
		d.To.Configuration.Hardware = &device.Hardware{
			Name: strings.TrimSpace(res),
		}
	}

	// Collect the operating system version
	if version, err := d.Shell("getprop", "ro.build.version.release").Call(ctx); err == nil {
		var major, minor, point int32
		fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &point)
		d.To.Configuration.OS = device.AndroidOS(major, minor, point)
	}

	if description, err := d.Shell("getprop", "ro.build.description").Call(ctx); err == nil {
		d.To.Configuration.OS.Build = strings.TrimSpace(description)
	}

	// Check which abis the device says it supports
	d.To.Configuration.ABIs = d.To.Configuration.ABIs[:0]
	seen := map[string]bool{}
	for _, prop := range []string{
		"ro.product.cpu.abilist",
		"ro.product.cpu.abi",
		"ro.product.cpu.abi2",
	} {
		abis, _ := d.Shell("getprop", prop).Call(ctx)
		if strings.TrimSpace(abis) == "" {
			continue
		}
		for _, abi := range strings.Split(abis, ",") {
			if seen[abi] {
				continue
			}
			d.To.Configuration.ABIs = append(d.To.Configuration.ABIs, device.ABIByName(abi))
			seen[abi] = true
		}
	}

	devInfoProvidersMutex.Lock()
	defer devInfoProvidersMutex.Unlock()
	for _, f := range devInfoProviders {
		if err := f(ctx, d); err != nil {
			return nil, err
		}
	}

	if i := d.Instance(); i.Id == nil || allZero(i.Id.Data) {
		// Generate an identifier for the device based on it's details.
		i.GenID()
	}

	return d, nil
}

func allZero(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return true
		}
	}
	return false
}

// scanDevices returns the list of attached Android devices.
func scanDevices(ctx context.Context) error {
	exe, err := adb()
	if err != nil {
		return log.Err(ctx, err, "")
	}
	stdout, err := shell.Command(exe.System(), "devices").Call(ctx)
	if err != nil {
		return err
	}
	parsed, err := parseDevices(ctx, stdout)
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	for serial, status := range parsed {
		device, ok := cache[serial]
		if !ok {
			device, err = newDevice(ctx, serial, status)
			if err != nil {
				return err
			}
			cache[serial] = device
			registry.AddDevice(ctx, device)
		}
	}

	// Remove cached results for removed devices.
	for serial, device := range cache {
		if _, found := parsed[serial]; !found {
			delete(cache, serial)
			registry.RemoveDevice(ctx, device)
		}
	}

	return nil
}

func parseDevices(ctx context.Context, out string) (map[string]bind.Status, error) {
	a := strings.SplitAfter(out, "List of devices attached")
	if len(a) != 2 {
		return nil, ErrNoDeviceList
	}
	lines := strings.Split(a[1], "\n")
	devices := make(map[string]bind.Status, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "adb server version") && strings.HasSuffix(line, "killing...") {
			continue // adb server version (36) doesn't match this client (35); killing...
		}
		if strings.HasPrefix(line, "*") {
			continue // For example, "* daemon started successfully *"
		}
		fields := strings.Fields(line)
		switch len(fields) {
		case 0:
			continue
		case 2:
			serial, status := fields[0], fields[1]
			switch status {
			case "unknown":
				devices[serial] = bind.Status_Unknown
			case "offline":
				devices[serial] = bind.Status_Offline
			case "device":
				devices[serial] = bind.Status_Online
			case "unauthorized":
				devices[serial] = bind.Status_Unauthorized
			default:
				return nil, log.Errf(ctx, ErrInvalidStatus, "value: %v", status)
			}
		default:
			return nil, ErrInvalidDeviceList
		}
	}
	return devices, nil
}

// NativeBridgeABI returns the native ABI for the given emulated ABI for the
// device by consulting the ro.dalvik.vm.isa.<emulated_isa>=<native_isa>
// system properties.
func (b *binding) NativeBridgeABI(ctx context.Context, emulated *device.ABI) *device.ABI {
	isa := abiToISA(emulated)
	if isa == "" {
		return emulated
	}
	isa, err := b.Shell("getprop", "ro.dalvik.vm.isa."+isa).Call(ctx)
	if err != nil {
		return emulated
	}
	native := isaToABI(isa)
	if native == nil {
		return emulated
	}
	return native
}

var abiToISAs = []struct {
	abi *device.ABI
	isa string
}{
	// {device.Architecture_ARMEABI, "arm"},
	{device.AndroidARMv7a, "arm"},
	{device.AndroidARM64v8a, "arm64"},
	{device.AndroidMIPS, "mips"},
	{device.AndroidMIPS64, "mips64"},
	{device.AndroidX86, "x86"},
	{device.AndroidX86_64, "x86_64"},
}

func abiToISA(abi *device.ABI) string {
	for _, e := range abiToISAs {
		if e.abi.Architecture == abi.Architecture {
			return e.isa
		}
	}
	return ""
}

func isaToABI(isa string) *device.ABI {
	for _, e := range abiToISAs {
		if e.isa == isa {
			return e.abi
		}
	}
	return nil
}
