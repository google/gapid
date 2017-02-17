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
	"fmt"
	"time"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/executor"
	"github.com/google/gapid/gapis/service/path"
)

const maxBatchDelay = 250 * time.Millisecond

// batcherContext is used as a key for the batch that's being formed.
type batcherContext struct {
	// Do not be tempted to turn these IDs into path nodes - go equality will
	// break and no batches will be formed.
	Device    id.ID
	Capture   id.ID
	Generator Generator
	Config    Config
}

type job struct {
	request Request
	result  chan<- error
}

type batcher struct {
	feed    chan job
	context batcherContext
	device  bind.Device
	gapir   *gapir.Client
}

var (
	generatorReplayCounter       = benchmark.GlobalCounters.Duration("batcher.send.generatorReplayTotalDuration")
	builderBuildCounter          = benchmark.GlobalCounters.Duration("batcher.send.builderBuildTotalDuration")
	executorExecuteCounter       = benchmark.GlobalCounters.Duration("batcher.send.executorExecuteTotalDuration")
	batcherSendInvocationCounter = benchmark.GlobalCounters.Integer("batcher.send.invocations")
)

func (b *batcher) run(ctx log.Context) {
	ctx = ctx.V("capture", b.context.Capture)

	// Gather all the batchEntries that are added to feed within maxBatchDelay.
	for j := range b.feed {
		jobs := []job{j}
		timeout := time.After(maxBatchDelay)
	inner:
		for {
			select {
			case j, ok := <-b.feed:
				if !ok {
					break inner
				}
				jobs = append(jobs, j)
			case <-timeout:
				break inner
			}
		}

		// Batch formed. Trigger the replay.
		requests := make([]Request, len(jobs))
		for i, job := range jobs {
			requests[i] = job.request
		}

		ctx.Info().Log("Replay batch")
		err := b.send(ctx, requests)
		for _, job := range jobs {
			job.result <- err
		}
	}
}

// captureMemoryLayout returns the device memory layout of the capture from the
// atoms. This function assumes there's an architecture atom at the beginning of
// the capture. TODO: Replace this with a proper capture header containing
// device and process information.
func captureMemoryLayout(ctx log.Context, list *atom.List) *device.MemoryLayout {
	s := capture.NewState(ctx)
	if len(list.Atoms) > 0 {
		list.Atoms[0].Mutate(ctx, s, nil)
	}
	return s.MemoryLayout
}

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

func (b *batcher) send(ctx log.Context, requests []Request) (err error) {
	batcherSendInvocationCounter.Increment()

	devPath := path.NewDevice(b.context.Device)
	capPath := path.NewCapture(b.context.Capture)
	intent := Intent{devPath, capPath}

	ctx = capture.Put(ctx, capPath)

	device := b.device.Instance()
	ctx = ctx.S("device", device.Name)

	ctx.Info().Logf("Replaying...")

	c, err := capture.ResolveFromPath(ctx, capPath)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to load capture")
	}

	list, err := c.Atoms(ctx)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to load atom stream")
	}

	cml := captureMemoryLayout(ctx, list)
	ctx = ctx.V("capture memory layout", cml)

	if len(device.Configuration.ABIs) == 0 {
		return cause.Explain(ctx, nil, "Replay device doesn't list any ABIs")
	}

	replayABI := findABI(cml, device.Configuration.ABIs)
	if replayABI == nil {
		ctx.Info().Log("Replay device does not have a memory layout matching device used to trace")
		replayABI = device.Configuration.ABIs[0]
	}
	ctx = ctx.V("replay target ABI", replayABI)

	builder := builder.New(replayABI.MemoryLayout)

	out := &adapter{
		state:   capture.NewState(ctx),
		builder: builder,
	}

	t0 := generatorReplayCounter.Start()
	if err := b.context.Generator.Replay(
		ctx,
		intent,
		b.context.Config,
		requests,
		device,
		c,
		out); err != nil {
		return cause.Explain(ctx, err, "Replay returned error")
	}
	generatorReplayCounter.Stop(t0)

	if config.DebugReplay {
		ctx.Print("Building payload...")
	}

	t0 = builderBuildCounter.Start()
	payload, decoder, err := builder.Build(ctx)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to build replay payload")
	}
	builderBuildCounter.Stop(t0)

	defer func() {
		caught := recover()
		if err == nil && caught != nil {
			err, _ = caught.(error)
			if err == nil {
				// If we are panicing, we always want an error to send.
				err = fmt.Errorf("%s", caught)
			}
		}
		if err != nil {
			// An error was returned or thrown after the replay postbacks were requested.
			// Inform each postback handler that they're not going to get data,
			// to avoid chans blocking forever.
			decoder(nil, err)
		}
		if caught != nil {
			panic(caught)
		}
	}()

	connection, err := b.gapir.Connect(ctx, b.device, replayABI)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to connect to device")
	}
	defer connection.Close()

	if config.DebugReplay {
		ctx.Info().Log("Sending payload")
	}

	if Events.OnReplay != nil {
		Events.OnReplay(b.device, intent, b.context.Config, requests)
	}

	t0 = executorExecuteCounter.Start()
	err = executor.Execute(
		ctx,
		payload,
		decoder,
		connection,
		replayABI.MemoryLayout,
	)
	executorExecuteCounter.Stop(t0)
	return err
}

// adapter conforms to the the atom Writer interface, performing replay writes
// on each atom.
type adapter struct {
	state   *gfxapi.State
	builder *builder.Builder
}

func (w *adapter) State() *gfxapi.State {
	return w.state
}

func (w *adapter) MutateAndWrite(ctx log.Context, i atom.ID, a atom.Atom) {
	w.builder.BeginAtom(uint64(i))
	if err := a.Mutate(ctx, w.state, w.builder); err == nil {
		w.builder.CommitAtom()
	} else {
		w.builder.RevertAtom(err)
		ctx.Warning().Logf("Failed to write atom %d (%T) for replay: %v", i, a, err)
	}
}
