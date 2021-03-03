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

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service/path"
)

// Exporter stores the input replays and export them as gapir instruction.
type Exporter interface {
	Manager
	// Export wait for waitRequests replay requests to be sent,
	// it then compiles the instructions for replay and triggers
	// all postback with builder.ErrReplayNotExecuted .
	Export(ctx context.Context, waitRequests int) (*gapir.Payload, error)
}

// NewExporter creates a new Exporter.
func NewExporter() Exporter {
	return &exportManager{
		requests: make(chan RequestAndResult),
	}
}

type exportManager struct {
	key      *batchKey
	requests chan RequestAndResult
}

func (m *exportManager) Export(ctx context.Context, waitRequests int) (*gapir.Payload, error) {
	var requests []RequestAndResult
	for i := 0; i < waitRequests; i++ {
		requests = append(requests, <-m.requests)
	}

	ctx = PutDevice(ctx, path.NewDevice(m.key.device))
	d := bind.GetRegistry(ctx).Device(m.key.device)

	capturePath := path.NewCapture(m.key.capture)
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to load capture")
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = log.V{
		"capture": m.key.capture,
		"device":  d.Instance().GetName(),
	}.Bind(ctx)

	intent := Intent{path.NewDevice(m.key.device), capturePath}

	cml := c.Header.ABI.MemoryLayout
	ctx = log.V{"capture memory layout": cml}.Bind(ctx)

	deviceABIs := d.Instance().GetConfiguration().GetABIs()
	if len(deviceABIs) == 0 {
		return nil, log.Err(ctx, nil, "Replay device doesn't list any ABIs")
	}

	replayABI := findABI(cml, deviceABIs)
	if replayABI == nil {
		log.I(ctx, "Replay device does not have a memory layout matching device used to trace")
		replayABI = deviceABIs[0]
	}
	ctx = log.V{"replay target ABI": replayABI}.Bind(ctx)

	b := builder.New(replayABI.MemoryLayout, nil)

	_, ranges, err := initialcmds.InitialCommands(ctx, capturePath)

	generatorReplayTimer.Time(func() {
		ctx := status.Start(ctx, "Generate")
		defer status.Finish(ctx)

		err = m.key.generator.Replay(
			ctx,
			intent,
			m.key.config,
			"",
			requests,
			d.Instance(),
			c,
			&builderWriter{
				state:   c.NewUninitializedState(ctx).ReserveMemory(ranges),
				builder: b,
			})
	})

	if err != nil {
		return nil, log.Err(ctx, err, "Replay returned error")
	}

	if config.DebugReplay {
		log.I(ctx, "Building payload...")
	}

	var payload gapir.Payload
	builderBuildTimer.Time(func() { payload, err = b.Export(ctx) })
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to build replay payload")
	}
	return &payload, nil
}

func (m *exportManager) Replay(
	ctx context.Context,
	intent Intent,
	cfg Config,
	req Request,
	generator Generator,
	hints *path.UsageHints,
	forceNonSplitReplay bool) (val interface{}, err error) {

	key := &batchKey{
		capture:             intent.Capture.ID.ID(),
		device:              intent.Device.ID.ID(),
		config:              cfg,
		generator:           generator,
		forceNonSplitReplay: forceNonSplitReplay,
	}

	if m.key == nil {
		m.key = key
	}

	if *key != *m.key {
		return nil, log.Errf(ctx, nil, "Can not export trace with incompatible requests")
	}

	type res struct {
		val interface{}
		err error
	}
	out := make(chan res, 1)
	m.requests <- RequestAndResult{
		Request: req,
		Result:  Result(func(val interface{}, err error) { out <- res{val, err} }),
	}
	r := <-out
	return r.val, r.err
}
