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

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/executor"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/scheduler"
	"github.com/google/gapid/gapis/service/path"
)

var (
	generatorReplayTimer = benchmark.Duration("replay.executor.generatorReplayTotalDuration")
	builderBuildTimer    = benchmark.Duration("replay.executor.builderBuildTotalDuration")
	executeTimer         = benchmark.Duration("replay.executor.executeTotalDuration")
	executeCounter       = benchmark.Integer("replay.executor.invocations")
)

// findABI looks for the ABI with the matching memory layout, retuning it if an
// exact match is found. If no matching ABI is found then nil is returned.
func findABI(ml *device.MemoryLayout, abis []*device.ABI) *device.ABI {
	for _, abi := range abis {
		if abi.MemoryLayout.SameAs(ml) {
			return abi
		}
	}
	return nil
}

func (m *Manager) batch(ctx context.Context, e []scheduler.Executable, b scheduler.Batch) {
	ctx = log.V{
		"priority": b.Priority,
		"delay":    b.Precondition,
	}.Bind(ctx)
	log.I(ctx, "Replay for %d requests", len(e))

	requests := make([]RequestAndResult, len(e))
	for i, e := range e {
		requests[i] = RequestAndResult{
			Request: e.Task,
			Result:  Result(e.Result),
		}
	}
	batch := b.Key.(batchKey)
	err := m.execute(ctx, batch.device, batch.capture, batch.config, batch.generator, requests)
	if err != nil {
		for _, e := range requests {
			e.Result(nil, err)
		}
	}
}

func (m *Manager) execute(
	ctx context.Context,
	deviceID, captureID id.ID,
	cfg Config,
	generator Generator,
	requests []RequestAndResult) error {

	executeCounter.Increment()

	devicePath := path.NewDevice(deviceID)
	d := bind.GetRegistry(ctx).Device(deviceID)
	if d == nil {
		return log.Errf(ctx, nil, "Unknown device %v", deviceID)
	}

	capturePath := path.NewCapture(captureID)
	c, err := capture.ResolveFromPath(ctx, capturePath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load capture")
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = log.V{
		"capture": captureID,
		"device":  d.Instance().GetName(),
	}.Bind(ctx)

	intent := Intent{devicePath, capturePath}

	cml := c.Header.Abi.MemoryLayout
	ctx = log.V{"capture memory layout": cml}.Bind(ctx)

	deviceABIs := d.Instance().GetConfiguration().GetABIs()
	if len(deviceABIs) == 0 {
		return log.Err(ctx, nil, "Replay device doesn't list any ABIs")
	}

	replayABI := findABI(cml, deviceABIs)
	if replayABI == nil {
		log.I(ctx, "Replay device does not have a memory layout matching device used to trace")
		replayABI = deviceABIs[0]
	}
	ctx = log.V{"replay target ABI": replayABI}.Bind(ctx)

	b := builder.New(replayABI.MemoryLayout)

	out := &adapter{
		state:   c.NewState(),
		builder: b,
	}

	generatorReplayTimer.Time(func() {
		err = generator.Replay(
			ctx,
			intent,
			cfg,
			requests,
			d.Instance(),
			c,
			out)
	})
	if err != nil {
		return log.Err(ctx, err, "Replay returned error")
	}

	if config.DebugReplay {
		log.I(ctx, "Building payload...")
	}

	var payload protocol.Payload
	var decoder builder.ResponseDecoder
	builderBuildTimer.Time(func() { payload, decoder, err = b.Build(ctx) })
	if err != nil {
		return log.Err(ctx, err, "Failed to build replay payload")
	}

	connection, err := m.gapir.Connect(ctx, d, replayABI)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to device")
	}
	defer connection.Close()

	if config.DebugReplay {
		log.I(ctx, "Sending payload")
	}

	if Events.OnReplay != nil {
		Events.OnReplay(d, intent, cfg)
	}

	executeTimer.Time(func() {
		err = executor.Execute(
			ctx,
			payload,
			decoder,
			connection,
			replayABI.MemoryLayout,
		)
	})
	return err
}

// adapter conforms to the the transformer.Writer interface, performing replay
// writes on each command.
type adapter struct {
	state   *api.GlobalState
	builder *builder.Builder
}

func (w *adapter) State() *api.GlobalState {
	return w.state
}

func (w *adapter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	w.builder.BeginCommand(uint64(id), cmd.Thread())
	if err := cmd.Mutate(ctx, id, w.state, w.builder); err == nil {
		w.builder.CommitCommand()
	} else {
		w.builder.RevertCommand(err)
		log.W(ctx, "Failed to write command %v %v for replay: %v", id, cmd, err)
	}
}
