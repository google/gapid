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

package bind

import (
	"context"
	"sync"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
)

type (
	registryKeyTy string

	propertyKey struct {
		d Device
		k interface{}
	}
)

const registryKey = registryKeyTy("registryKeyID")

// PutRegistry attaches a registry to a Context.
func PutRegistry(ctx context.Context, m *Registry) context.Context {
	return keys.WithValue(ctx, registryKey, m)
}

// GetRegistry retrieves the registry from a context previously annotated by
// PutRegistry.
func GetRegistry(ctx context.Context) *Registry {
	val := ctx.Value(registryKey)
	if val == nil {
		panic(string(registryKey + " not present"))
	}
	return val.(*Registry)
}

// Registry is holds a list of registered devices. It provides methods for
// listening for devices that are added to and removed from the device.
type Registry struct {
	sync.Mutex
	devices    []Device
	properties map[propertyKey]interface{}
	listeners  map[DeviceListener]struct{}
}

// NewRegistry returns a newly constructed Registry.
func NewRegistry() *Registry {
	return &Registry{
		listeners:  make(map[DeviceListener]struct{}),
		properties: make(map[propertyKey]interface{}),
	}
}

// DeviceListener is the interface implemented by types that respond to devices
// being added to and removed from the registry.
type DeviceListener interface {
	OnDeviceAdded(context.Context, Device)
	OnDeviceRemoved(context.Context, Device)
}

// NewDeviceListener returns a DeviceListener that delegates calls on to
// onDeviceAdded and onDeviceRemoved.
func NewDeviceListener(onDeviceAdded, onDeviceRemoved func(context.Context, Device)) DeviceListener {
	return &funcDeviceListener{onDeviceAdded, onDeviceRemoved}
}

// funcDeviceListener is an implementatation of DeviceListener that delegates
// calls on to the field functions.
type funcDeviceListener struct {
	onAdded   func(context.Context, Device)
	onRemoved func(context.Context, Device)
}

func (l funcDeviceListener) OnDeviceAdded(ctx context.Context, d Device) {
	if f := l.onAdded; f != nil {
		f(ctx, d)
	}
}

func (l funcDeviceListener) OnDeviceRemoved(ctx context.Context, d Device) {
	if f := l.onRemoved; f != nil {
		f(ctx, d)
	}
}

// Listen registers l to be called whenever a device is added to or removed from
// the registry. l will be unregistered when the returned function is called.
func (r *Registry) Listen(l DeviceListener) (unregister func()) {
	r.Lock()
	r.listeners[l] = struct{}{}
	r.Unlock()
	return func() {
		r.Lock()
		delete(r.listeners, l)
		r.Unlock()
	}
}

// Device looks up the device with the specified identifier.
// If no device with the specified identifier was registered with the Registry
// then nil is returner.
func (r *Registry) Device(id id.ID) Device {
	r.Lock()
	defer r.Unlock()
	for _, d := range r.devices {
		if d.Instance().ID.ID() == id {
			return d
		}
	}
	return nil
}

// Devices returns the list of devices registered with the Registry.
func (r *Registry) Devices() []Device {
	r.Lock()
	defer r.Unlock()

	out := make([]Device, len(r.devices))
	copy(out, r.devices)
	return out
}

// DefaultDevice returns the first device registered with the Registry.
func (r *Registry) DefaultDevice() Device {
	r.Lock()
	defer r.Unlock()
	if len(r.devices) == 0 {
		return nil
	}
	return r.devices[0]
}

// AddDevice registers the device dev with the Registry.
func (r *Registry) AddDevice(ctx context.Context, d Device) {
	if d != nil {
		r.Lock()
		defer r.Unlock()
		for _, t := range r.devices {
			if t == d {
				return // already added
			}
		}
		ctx := log.V{"device": d}.Bind(ctx)
		log.I(ctx, "Adding new device")
		r.devices = append(r.devices, d)
		for l := range r.listeners {
			l.OnDeviceAdded(ctx, d)
		}
	}
}

// RemoveDevice unregisters the device d with the Registry.
func (r *Registry) RemoveDevice(ctx context.Context, d Device) {
	if d != nil {
		r.Lock()
		defer r.Unlock()
		for i, t := range r.devices {
			if t == d {
				ctx := log.V{"device": d}.Bind(ctx)
				log.I(ctx, "Removing existing device")
				copy(r.devices[i:], r.devices[i+1:])
				r.devices = r.devices[:len(r.devices)-1]
				for l := range r.listeners {
					l.OnDeviceRemoved(ctx, d)
				}
			}
		}
	}
}

// DeviceProperty returns the property with the key k for the device d,
// previously set with SetDeviceProperty. If the property for the device does
// not exist then nil is returned.
func (r *Registry) DeviceProperty(ctx context.Context, d Device, k interface{}) interface{} {
	r.Lock()
	defer r.Unlock()
	return r.properties[propertyKey{d, k}]
}

// SetDeviceProperty sets the property with the key k to the value v for the
// device d. This property can be retrieved with DeviceProperty.
// Properties will persist in the registry even when the device has not been
// added or has been removed.
func (r *Registry) SetDeviceProperty(ctx context.Context, d Device, k, v interface{}) {
	r.Lock()
	defer r.Unlock()
	r.properties[propertyKey{d, k}] = v
}
