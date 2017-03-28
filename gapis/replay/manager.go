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

package replay

import (
	"context"
	"sync"

	"time"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/replay/scheduler"
	"github.com/google/gapid/gapis/service"
)

const (
	lowPriority       = 0
	defaultPriority   = 1
	highPriorty       = 2
	defaultBatchDelay = time.Millisecond * 100
)

// Manager is used discover replay devices and to send replay requests to those
// discovered devices.
type Manager struct {
	gapir      *gapir.Client
	schedulers map[id.ID]*scheduler.Scheduler
	mutex      sync.Mutex // guards schedulers
}

// batchKey is used as a key for the batch that's being formed.
type batchKey struct {
	// Do not be tempted to turn these IDs into path nodes - go equality will
	// break and no batches will be formed.
	capture   id.ID
	device    id.ID
	config    Config
	generator Generator
}

// New returns a new Manager instance using the database db.
func New(ctx context.Context) *Manager {
	return &Manager{
		gapir:      gapir.New(ctx),
		schedulers: make(map[id.ID]*scheduler.Scheduler),
	}
}

// Replay requests that req is to be performed on the device described by intent,
// using the capture described by intent. Replay requests made with configs that
// have equality (==) will likely be batched into the same replay pass.
func (m *Manager) Replay(
	ctx context.Context,
	intent Intent,
	cfg Config,
	req Request,
	generator Generator,
	hints *service.UsageHints) (val interface{}, err error) {

	log.I(ctx, "Replay request")
	s, err := m.scheduler(ctx, intent.Device.Id.ID())
	if err != nil {
		return nil, err
	}

	b := scheduler.Batch{
		Key: batchKey{
			capture:   intent.Capture.Id.ID(),
			device:    intent.Device.Id.ID(),
			config:    cfg,
			generator: generator,
		},
		Priority:     defaultPriority,
		Precondition: defaultBatchDelay,
	}
	if hints != nil {
		if hints.Preview {
			b.Priority = lowPriority
		}
		if hints.Primary {
			b.Priority = highPriorty
			b.Precondition = nil
		}
	}
	return s.Schedule(ctx, req, b)
}

func (m *Manager) scheduler(ctx context.Context, deviceID id.ID) (*scheduler.Scheduler, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	s, found := m.schedulers[deviceID]
	if !found {
		log.I(ctx, "New scheduler for device: %v", deviceID)
		s = scheduler.New(ctx, m.batch)
		m.schedulers[deviceID] = s
	}
	return s, nil
}
