// Copyright (C) 2018 Google Inc.
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

package trace

import (
	"context"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/trace/android"
	"github.com/google/gapid/gapis/trace/desktop"
	"github.com/google/gapid/gapis/trace/tracer"
)

// Manager is used discover trace devices and to send trace requests
// to those discovered devices.
type Manager struct {
	mutex sync.Mutex // guards schedulers

	tracers map[id.ID]tracer.Tracer
}

// New returns a new Manager instance using the database db.
func New(ctx context.Context) *Manager {
	out := &Manager{
		sync.Mutex{},
		make(map[id.ID]tracer.Tracer),
	}
	bind.GetRegistry(ctx).Listen(bind.NewDeviceListener(out.createTracer, out.destroyTracer))
	return out
}

func (m *Manager) createTracer(ctx context.Context, dev bind.Device) {
	if !dev.CanTrace() {
		return
	}
	deviceID := dev.Instance().ID.ID()
	log.I(ctx, "New trace scheduler for device: %v %v", deviceID, dev.Instance().Name)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if dev.Instance().GetConfiguration().GetOS().GetKind() == device.Android {
		m.tracers[deviceID] = android.NewTracer(dev)
	} else {
		m.tracers[deviceID] = desktop.NewTracer(dev)
	}
}

func (m *Manager) destroyTracer(ctx context.Context, dev bind.Device) {
	if !dev.CanTrace() {
		return
	}
	deviceID := dev.Instance().ID.ID()
	log.I(ctx, "Destroying trace scheduler for device: %v", deviceID)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.tracers, deviceID)
}
