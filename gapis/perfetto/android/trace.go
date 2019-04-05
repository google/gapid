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

package android

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"perfetto_pb"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/service"
)

const (
	// perfettoTraceFile is the location on the device where we'll ask Perfetto
	// to store the trace data while tracing.
	perfettoTraceFile = "/data/misc/perfetto-traces/gapis-trace"
)

// Process represents a running Perfetto capture.
type Process struct {
	device   adb.Device
	config   *perfetto_pb.TraceConfig
	deferred bool
}

// Start optional starts an app and sets up a Perfetto trace
func Start(ctx context.Context, d adb.Device, a *android.ActivityAction, opts *service.TraceOptions) (*Process, error) {
	ctx = log.Enter(ctx, "start")
	if a != nil {
		ctx = log.V{
			"package":  a.Package.Name,
			"activity": a.Activity,
		}.Bind(ctx)
	}

	log.I(ctx, "Turning device screen on")
	if err := d.TurnScreenOn(ctx); err != nil {
		return nil, log.Err(ctx, err, "Couldn't turn device screen on")
	}

	log.I(ctx, "Checking for lockscreen")
	locked, err := d.IsShowingLockscreen(ctx)
	if err != nil {
		log.W(ctx, "Couldn't determine lockscreen state: %v", err)
	}
	if locked {
		return nil, log.Err(ctx, nil, "Cannot trace app on locked device")
	}

	if a != nil {
		if err := d.StartActivity(ctx, *a); err != nil {
			return nil, log.Err(ctx, err, "Starting the activity")
		}
	}

	return &Process{
		device:   d,
		config:   opts.PerfettoConfig,
		deferred: opts.DeferStart,
	}, nil
}

// Capture starts the perfetto capture.
func (p *Process) Capture(ctx context.Context, start task.Signal, stop task.Signal, w io.Writer, written *int64) (int64, error) {
	tmp, err := file.Temp()
	if err != nil {
		return 0, log.Err(ctx, err, "Failed to create a temp file")
	}

	// Signal that we are ready to start.
	atomic.StoreInt64(written, 1)

	if p.deferred && !start.Wait(ctx) {
		return 0, log.Err(ctx, nil, "Cancelled")
	}

	if err := p.device.StartPerfettoTrace(ctx, p.config, perfettoTraceFile, stop); err != nil {
		return 0, err
	}

	if err := p.device.Pull(ctx, perfettoTraceFile, tmp.System()); err != nil {
		return 0, err
	}

	size := tmp.Info().Size()
	atomic.StoreInt64(written, size)
	fh, err := os.Open(tmp.System())
	if err != nil {
		return 0, log.Err(ctx, err, fmt.Sprintf("Failed to open %s", tmp))
	}
	return io.Copy(w, fh)
}
