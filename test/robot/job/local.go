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
	"reflect"
	"sync"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

type local struct {
	mu       sync.Mutex
	devices  devices
	entries  []*Worker
	onChange event.Broadcast
}

// NewLocal builds a new job manager that persists it's data in the
// supplied library.
func NewLocal(ctx context.Context, library record.Library) (Manager, error) {
	m := &local{}
	if err := m.devices.init(ctx, library); err != nil {
		return nil, err
	}
	return m, nil
}

// SearchDevices implements Manager.SearchDevicess
// It searches the set of persisted devices, and supports monitoring of new devices as they are added.
func (m *local) SearchDevices(ctx context.Context, query *search.Query, handler DeviceHandler) error {
	return m.devices.search(ctx, query, handler)
}

// SearchWorkers implements Manager.SearchWorkers
// It searches the set of persisted workers, and supports monitoring of workers as they are registered.
func (m *local) SearchWorkers(ctx context.Context, query *search.Query, handler WorkerHandler) error {
	filter := eval.Filter(ctx, query, reflect.TypeOf(&Worker{}), event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, m.entries)
	if query.Monitor {
		return event.Monitor(ctx, &m.mu, m.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

// GetWorker implements Manager.GetWorker
// This attempts to find a worker on a device that matches the supplied host
// controlling a device that matches the supplied target to perform the given operation.
// If none is found, a new worker will be created.
func (m *local) GetWorker(ctx context.Context, host *device.Instance, target *device.Instance, op Operation) (*Worker, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, err := m.devices.get(ctx, host)
	if err != nil {
		return nil, err
	}
	t, err := m.devices.get(ctx, target)
	if err != nil {
		return nil, err
	}
	for _, entry := range m.entries {
		if entry.Host != h.Id {
			continue
		}
		if entry.Target != t.Id {
			continue
		}
		if !entry.Supports(op) {
			entry.Operation = append(entry.Operation, op)
			m.onChange.Send(ctx, entry)
			return entry, nil
		}
	}
	// Not found, add a new worker
	entry := &Worker{
		Host:      h.Id,
		Target:    t.Id,
		Operation: []Operation{op},
	}
	m.entries = append(m.entries, entry)
	m.onChange.Send(ctx, entry)
	return entry, nil
}

// Supports is used to test if a worker supports a given operation.
func (w *Worker) Supports(op Operation) bool {
	for _, e := range w.Operation {
		if e == op {
			return true
		}
	}
	return false
}
