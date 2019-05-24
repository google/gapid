// Copyright (C) 2019 Google Inc.
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

package resolve

import (
	"context"
	"errors"
	"time"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace"
)

func Perfetto(ctx context.Context, p *path.Perfetto, r *path.ResolveConfig) (*path.Capture, error) {
	obj, err := database.Build(ctx, &PerfettoResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*path.Capture), nil
}

func (r *PerfettoResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, r.Path.Capture, r.Config)

	c, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}
	d := bind.GetRegistry(ctx).Device(r.Config.ReplayDevice.ID.ID())
	if d == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrUnknownDevice()}
	}

	if !d.SupportsPerfetto(ctx) {
		return nil, errors.New("The replay device does not support Perfetto")
	}

	defer analytics.SendTiming("resolve", "perfetto")(analytics.Size(len(c.Commands)))

	intent := replay.Intent{
		Capture: r.Path.Capture,
		Device:  r.Config.ReplayDevice,
	}
	hints := &service.UsageHints{Background: true}
	mgr := replay.GetManager(ctx)
	out, err := file.Temp()
	if err != nil {
		return nil, err
	}

	for _, a := range c.APIs {
		if pf, ok := a.(replay.Profiler); ok {
			h := startTrace(ctx, options(r.Path, r.Config.ReplayDevice, out.System()))

			for h.written == 0 {
				time.Sleep(5 * time.Millisecond)
			}

			err := pf.Profile(ctx, intent, mgr, hints, r.Path.Overrides)
			h.stopFunc(ctx)

			if err != nil {
				return nil, err
			}

			h.doneSignal.Wait(ctx)
			if h.err != nil {
				return nil, h.err
			}

			src := &capture.File{Path: out.System()}
			r, err := capture.Import(ctx, c.Name()+"_perfetto", out.System(), src)
			if err != nil {
				return nil, err
			}
			// Ensure the capture can be read by resolving it now.
			if _, err = capture.ResolveFromPath(ctx, r); err != nil {
				return nil, err
			}
			return r, nil
		}
	}

	return nil, errors.New("The capture does not support profiling")
}

type handler struct {
	stopFunc   task.Task
	doneSignal task.Signal
	written    int64
	err        error
}

func startTrace(ctx context.Context, opts *service.TraceOptions) *handler {
	startSignal, _ := task.NewSignal()
	stopSignal, stopFunc := task.NewSignal()
	doneSignal, doneFunc := task.NewSignal()

	r := &handler{
		stopFunc:   stopFunc,
		doneSignal: doneSignal,
	}
	go func() {
		r.err = trace.Trace(ctx, opts.Device, startSignal, stopSignal, opts, &r.written)
		doneFunc(ctx)
	}()
	return r
}

func options(p *path.Perfetto, d *path.Device, out string) *service.TraceOptions {
	return &service.TraceOptions{
		Device:              d,
		Type:                service.TraceType_Perfetto,
		PerfettoConfig:      p.Config,
		ServerLocalSavePath: out,
	}
}
