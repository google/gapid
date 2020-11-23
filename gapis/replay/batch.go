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

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/scheduler"
	"github.com/google/gapid/gapis/resolve/initialcmds"
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

func (m *manager) batch(ctx context.Context, r *status.Replay, e []scheduler.Executable, b scheduler.Batch) {
	batch := b.Key.(batchKey)

	ctx = PutDevice(ctx, path.NewDevice(batch.device))

	d := bind.GetRegistry(ctx).Device(batch.device)

	requests := make([]RequestAndResult, len(e))
	for i, e := range e {
		requests[i] = RequestAndResult{
			Request: e.Task,
			Result:  Result(e.Result),
		}
	}

	err := func() error {
		if d == nil {
			return log.Errf(ctx, nil, "Unknown device %v", batch.device)
		}

		defer analytics.SendTiming("replay", "batch")(
			analytics.TargetDevice(d.Instance().GetConfiguration()),
		)

		ctx = log.V{
			"priority": b.Priority,
			"delay":    b.Precondition,
		}.Bind(ctx)
		log.I(ctx, "Replay for %d requests", len(e))

		return m.execute(ctx, d, r, batch.device, batch.capture, batch.config, batch.generator, batch.forceNonSplitReplay, requests)
	}()

	if err != nil {
		analytics.SendEvent("replay", "batch", "failure",
			analytics.TargetDevice(d.Instance().GetConfiguration()),
		)
		for _, e := range requests {
			e.Result(nil, err)
		}
	} else {
		analytics.SendEvent("replay", "batch", "success",
			analytics.TargetDevice(d.Instance().GetConfiguration()),
		)
	}
}

type InitialPayloadResult struct {
	prerunID   string
	cleanupID  string
	oldBuilder *builder.Builder
	oldState   *api.GlobalState
}

func (r *InitialPayloadResolvable) Resolve(
	ctx context.Context) (interface{}, error) {

	d := bind.GetRegistry(ctx).Device(r.DeviceID.ID())
	captureID := r.CaptureID.ID()
	generator := api.Find(api.ID(r.ApiID.ID())).(SplitGenerator)

	ctx = status.Start(ctx, "Initial Payload")
	defer status.Finish(ctx)

	capturePath := path.NewCapture(captureID)
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to load capture")
	}

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

	out := &adapter{
		state:   c.NewUninitializedState(ctx),
		builder: b,
	}
	oldBuilder := b

	generatorReplayTimer.Time(func() {
		nctx := status.Start(ctx, "Generate Initial")
		defer status.Finish(nctx)

		err = generator.GetInitialPayload(
			nctx,
			capturePath,
			d.Instance(),
			out)
	})
	if err != nil {
		return nil, log.Err(ctx, err, "Initial Payload returned error")
	}

	if config.DebugReplay {
		log.I(ctx, "Building Initial Payload...")
	}

	var payload gapir.Payload
	builderBuildTimer.Time(func() {
		log.D(ctx, "Initial Payload:")
		payload, _, _, _, err = b.Build(ctx)
	})
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to build initial payload")
	}

	generatedState := c.NewUninitializedState(ctx)
	generatedState.Memory = out.state.Memory.Clone()
	// Clone serialized state, and initialize it for use.
	for k, v := range out.state.APIs {
		s := v.Clone()
		s.SetupInitialState(ctx, out.state)
		generatedState.APIs[k] = s
	}

	b = builder.New(replayABI.MemoryLayout, oldBuilder)
	out = &adapter{
		state:   out.state,
		builder: b,
	}

	generatorReplayTimer.Time(func() {
		nctx := status.Start(ctx, "Generate Cleanup")
		defer status.Finish(nctx)

		err = generator.CleanupResources(
			nctx,
			d.Instance(),
			out)
	})
	if err != nil {
		return nil, log.Err(ctx, err, "Cleanup Payload returned error")
	}

	var cleanupPayload gapir.Payload
	builderBuildTimer.Time(func() {
		log.D(ctx, "Cleanup Payload:")
		cleanupPayload, _, _, _, err = b.Build(ctx)
	})

	id, err := database.Store(ctx, &payload)
	if err != nil {
		return nil, log.Errf(ctx, err, "Storing warmup payload")
	}

	cleanID, err := database.Store(ctx, &cleanupPayload)
	if err != nil {
		return nil, log.Errf(ctx, err, "Storing cleanup payload")
	}

	return InitialPayloadResult{
		prerunID:   id.String(),
		cleanupID:  cleanID.String(),
		oldBuilder: oldBuilder,
		oldState:   generatedState,
	}, nil
}

func (m *manager) execute(
	ctx context.Context,
	d bind.Device,
	r *status.Replay,
	deviceID, captureID id.ID,
	cfg Config,
	generator Generator,
	forceNonSplitReplay bool,
	requests []RequestAndResult) error {

	capturePath := path.NewCapture(captureID)
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load capture")
	}

	ctx = status.Start(ctx, "Batch (%d x config: %T%+v)", len(requests), cfg, cfg)
	defer status.Finish(ctx)

	executeCounter.Increment()

	ctx = capture.Put(ctx, capturePath)
	ctx = log.V{
		"capture": captureID,
		"device":  d.Instance().GetName(),
	}.Bind(ctx)

	intent := Intent{path.NewDevice(deviceID), capturePath}

	cml := c.Header.ABI.MemoryLayout
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

	// We may or may not actually need the initial payload, but kick it off now
	// so we have the maximum chance of it being ready when we need it.
	ipl := InitialPayloadResolvable{
		CaptureID: NewID(captureID),
		ApiID:     NewID(id.ID(generator.(api.API).ID())),
		DeviceID:  NewID(d.Instance().ID.ID()),
	}

	newCtx := keys.Clone(context.Background(), ctx)
	go func() {
		// kick the initial payload build. we don't know if we actually
		// need it yet, but want to kick it off as early as possible,
		// even before we're connected.

		// errors here are OK -- we'll pick them up later when we force completion
		cctx := status.PutTask(newCtx, nil)
		cctx = status.StartBackground(cctx, "Precaching initial payload")
		defer status.Finish(cctx)
		_, _ = database.Build(cctx, &ipl)
	}()

	conn, err := m.connect(ctx, d, replayABI)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to device")
	}

	var depID string
	var depBuilder *builder.Builder
	var depState *api.GlobalState
	if !forceNonSplitReplay {
		if _, ok := generator.(SplitGenerator); ok {
			// Force initial payload to be finished building
			ipr, err := database.Build(ctx, &ipl)
			if err != nil {
				return err
			}
			i, ok := ipr.(InitialPayloadResult)
			if !ok {
				return log.Err(ctx, nil, "Invalid Initial Payload")
			}

			depID = i.prerunID
			depBuilder = i.oldBuilder
			depState = i.oldState

			err = m.PrewarmReplay(ctx, conn, i.prerunID, i.cleanupID)
			if err != nil {
				return log.Err(ctx, err, "Replay returned error")
			}
		}
	}

	initState := c.NewUninitializedState(ctx)
	if depState != nil {
		initState.Memory = depState.Memory.Clone()
		// Clone serialized state, and initialize it for use.
		for k, v := range depState.APIs {
			s := v.Clone()
			s.SetupInitialState(ctx, initState)
			initState.APIs[k] = s
		}
	}

	b := builder.New(replayABI.MemoryLayout, depBuilder)

	_, ranges, err := initialcmds.InitialCommands(ctx, capturePath)

	out := &adapter{
		state:   initState,
		builder: b,
	}
	out.state.ReserveMemory(ranges)
	generatorReplayTimer.Time(func() {
		nctx := status.Start(ctx, "Generate")
		defer status.Finish(nctx)

		err = generator.Replay(
			nctx,
			intent,
			cfg,
			depID,
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
	var payload gapir.Payload
	var handlePost builder.PostDataHandler
	var handleNotification builder.NotificationHandler
	var fenceReadyCallback builder.FenceReadyRequestCallback
	builderBuildTimer.Time(func() {
		log.D(ctx, "Main Payload:")
		payload, handlePost, handleNotification, fenceReadyCallback, err = b.Build(ctx)
	})
	if err != nil {
		return log.Err(ctx, err, "Failed to build replay payload")
	}

	err = b.RegisterReplayStatusReader(ctx, r)
	if err != nil {
		return log.Err(ctx, err, "Failed to register replay status notification reader.")
	}

	if Events.OnReplay != nil {
		Events.OnReplay(d, intent, cfg)
	}

	// TODO(pmuetschard): Make the "state reconstruction" an actual event status and have gapir let us
	// know how far along it is in executing it. Also, there is technically still some server work
	// happening after this (e.g. storing the payload in the db), but that is fast and passing things
	// down isn't worth it.
	// This transitions the status from building to executing.
	r.Progress(ctx, 0, 1, 0)

	executeTimer.Time(func() {
		err = Execute(
			ctx,
			depID,
			payload,
			handlePost,
			handleNotification,
			fenceReadyCallback,
			m,
			conn,
			replayABI.MemoryLayout,
			d.Instance().GetConfiguration().GetOS(),
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

func (w *adapter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	w.builder.BeginCommand(uint64(id), cmd.Thread())
	err := cmd.Mutate(ctx, id, w.state, w.builder, nil)
	if err == nil {
		w.builder.CommitCommand(ctx, true)
	} else {
		w.builder.RevertCommand(err)
		log.W(ctx, "Failed to write command %v %v for replay: %v", id, cmd, err)
	}
	return err
}
