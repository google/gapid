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

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/replay/scheduler"
	"github.com/google/gapid/gapis/service/path"
)

const (
	lowestPriority       = 0
	lowPriority          = 1
	defaultPriority      = 2
	highPriorty          = 3
	backgroundBatchDelay = time.Millisecond * 500
	defaultBatchDelay    = time.Millisecond * 100
)

// Manager executes replay requests.
type Manager interface {
	// Replay requests that req is to be performed on the device described by
	// intent, using the capture described by intent.
	Replay(
		ctx context.Context,
		intent Intent,
		cfg Config,
		req Request,
		generator Generator,
		hints *path.UsageHints,
		forceNonSplitReplay bool) (val interface{}, err error)
}

// Manager is used discover replay devices and to send replay requests to those
// discovered devices.
type manager struct {
	gapir      *gapir.Client
	schedulers map[id.ID]*scheduler.Scheduler
	mutex      sync.Mutex // guards schedulers
}

// batchKey is used as a key for the batch that's being formed.
type batchKey struct {
	// Do not be tempted to turn these IDs into path nodes - go equality will
	// break and no batches will be formed.
	capture             id.ID
	device              id.ID
	config              Config
	generator           Generator
	forceNonSplitReplay bool
}

// New returns a new Manager instance using the database db.
func New(ctx context.Context) Manager {
	out := &manager{
		gapir:      gapir.New(ctx),
		schedulers: make(map[id.ID]*scheduler.Scheduler),
	}
	bind.GetRegistry(ctx).Listen(bind.NewDeviceListener(out.createScheduler, out.destroyScheduler))
	return out
}

// NewManagerForTest returns a new Manager for the test.
func NewManagerForTest(client *gapir.Client) Manager {
	return &manager{gapir: client}
}

// Replay requests that req is to be performed on the device described by intent,
// using the capture described by intent. Replay requests made with configs that
// have equality (==) will likely be batched into the same replay pass.
func (m *manager) Replay(
	ctx context.Context,
	intent Intent,
	cfg Config,
	req Request,
	generator Generator,
	hints *path.UsageHints,
	forceNonSplitReplay bool) (val interface{}, err error) {

	ctx = status.Start(ctx, "Replay Request")
	defer status.Finish(ctx)
	status.Block(ctx)
	defer status.Unblock(ctx)

	log.D(ctx, "Replay request")
	s, err := m.scheduler(ctx, intent.Device.ID.ID())
	if err != nil {
		return nil, err
	}

	b := scheduler.Batch{
		Key: batchKey{
			capture:             intent.Capture.ID.ID(),
			device:              intent.Device.ID.ID(),
			config:              cfg,
			generator:           generator,
			forceNonSplitReplay: forceNonSplitReplay,
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
		if hints.Background {
			b.Priority = lowestPriority
			b.Precondition = backgroundBatchDelay
		}
	}
	return s.Schedule(ctx, req, b)
}

func (m *manager) scheduler(ctx context.Context, deviceID id.ID) (*scheduler.Scheduler, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	s, found := m.schedulers[deviceID]
	if !found {
		return nil, log.Err(ctx, nil, "Device scheduler not found")
	}
	return s, nil
}

func (m *manager) createScheduler(ctx context.Context, device bind.Device) {
	deviceID := device.Instance().ID.ID()
	log.I(ctx, "New scheduler for device: %v", deviceID)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.schedulers[deviceID] = scheduler.New(ctx, deviceID, m.batch)
}

func (m *manager) destroyScheduler(ctx context.Context, device bind.Device) {
	deviceID := device.Instance().ID.ID()
	log.I(ctx, "Destroying scheduler for device: %v", deviceID)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.schedulers, deviceID)
}

func (m *manager) connect(ctx context.Context, device bind.Device, replayABI *device.ABI) (*gapir.ReplayerKey, error) {
	return m.gapir.Connect(ctx, device, replayABI)
}

func (m *manager) BeginReplay(ctx context.Context, key *gapir.ReplayerKey, payload string, dependent string) error {
	return m.gapir.BeginReplay(ctx, key, payload, dependent)
}

func (m *manager) SetReplayExecutor(ctx context.Context, key *gapir.ReplayerKey, executor gapir.ReplayExecutor) (func(), error) {
	return m.gapir.SetReplayExecutor(ctx, key, executor)
}

func (m *manager) PrewarmReplay(ctx context.Context, key *gapir.ReplayerKey, payload string, cleanup string) error {
	return m.gapir.PrewarmReplay(ctx, key, payload, cleanup)
}
