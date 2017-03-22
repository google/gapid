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
	"fmt"
	"sync"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	gapir "github.com/google/gapid/gapir/client"
)

// Manager is used discover replay devices and to send replay requests to those
// discovered devices.
type Manager struct {
	gapir    *gapir.Client
	batchers map[batcherContext]*batcher
	mutex    sync.Mutex // guards batchers
}

func (m *Manager) getBatchStream(ctx context.Context, bContext batcherContext) (chan<- job, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// TODO: This accumulates batchers with running go-routines forever.
	// Rework to free the batcher after execution.
	b, found := m.batchers[bContext]
	if !found {
		log.I(ctx, "New batch stream for capture: %v", bContext.Capture)
		device := bind.GetRegistry(ctx).Device(bContext.Device)
		if device == nil {
			return nil, fmt.Errorf("Unknown device %v", bContext.Device)
		}
		b = &batcher{
			feed:    make(chan job, 64),
			context: bContext,
			device:  device,
			gapir:   m.gapir,
		}
		m.batchers[bContext] = b
		go b.run(ctx)
	}
	return b.feed, nil
}

// New returns a new Manager instance using the database db.
func New(ctx context.Context) *Manager {
	return &Manager{
		gapir:    gapir.New(ctx),
		batchers: make(map[batcherContext]*batcher),
	}
}

// Replay requests that req is to be performed on the device described by intent,
// using the capture described by intent. Replay requests made with configs that
// have equality (==) will likely be batched into the same replay pass.
func (m *Manager) Replay(ctx context.Context, intent Intent, cfg Config, req Request, generator Generator) error {
	log.I(ctx, "Replay request")
	batch, err := m.getBatchStream(ctx, batcherContext{
		Device:    intent.Device.Id.ID(),
		Capture:   intent.Capture.Id.ID(),
		Generator: generator,
		Config:    cfg,
	})
	if err != nil {
		return err
	}
	res := make(chan error, 1)
	batch <- job{request: req, result: res}
	return <-res
}
