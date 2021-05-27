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

package desktop

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"sync/atomic"

	perfetto_pb "protos/perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/remotessh"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/service"
)

// Process represents a running Perfetto capture.
type Process struct {
	device      bind.DeviceWithShell
	config      *perfetto_pb.TraceConfig
	deferred    bool
	tracefile   string
	perfettoCmd string
	setup       perfettoDeviceSetup
	cleanup     app.Cleanup

	perf shell.Process
	err  chan error
}

// perfettoDeviceSetup handles getting files from/to the right location
// for a particular Device
type perfettoDeviceSetup interface {
	// makeTempDir returns a path to a created temporary. The returned function
	// can be called to clean up the temporary directory.
	makeTempDir(ctx context.Context) (string, app.Cleanup, error)

	// initializePerfetto takes the perfetto executable, and if necessary copies it
	// into the given temporary directory. It returns the executable
	// location.
	initializePerfetto(ctx context.Context, tempdir string) (string, error)

	// pullTrace moves the trace from src to dst if necessary.
	// If not necessary, it leaves the file at source. Returns the
	// location of the file
	pullTrace(ctx context.Context, source string, dest string) (string, error)
}

type localSetup struct {
	device bind.Device
	abi    *device.ABI
}

type remoteSetup struct {
	device remotessh.Device
	abi    *device.ABI
}

func (*localSetup) makeTempDir(ctx context.Context) (string, app.Cleanup, error) {
	tempdir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return "", nil, err
	}
	return tempdir, func(ctx context.Context) {
		os.RemoveAll(tempdir)
	}, nil
}

func (r *remoteSetup) makeTempDir(ctx context.Context) (string, app.Cleanup, error) {
	return r.device.TempDir(ctx)
}

func (l *localSetup) initializePerfetto(ctx context.Context, tempdir string) (string, error) {
	lib, err := layout.PerfettoCmd(ctx, l.abi)
	if err != nil {
		return "", err
	}
	return lib.System(), nil
}

func (r *remoteSetup) initializePerfetto(ctx context.Context, tempdir string) (string, error) {
	lib, err := layout.PerfettoCmd(ctx, r.abi)
	if err != nil {
		return "", err
	}
	if err := r.device.PushFile(ctx, lib.System(), tempdir+"/perfetto"); err != nil {
		return "", err
	}
	return tempdir + "/perfetto", nil
}

func (l *localSetup) pullTrace(ctx context.Context, source string, dest string) (string, error) {
	file, err := l.device.IsFile(ctx, source)
	if err != nil {
		return "", err
	}
	if !file {
		return "", fmt.Errorf("Could not open file %s", source)
	}
	return source, nil
}

func (l *remoteSetup) pullTrace(ctx context.Context, source string, dest string) (string, error) {
	file, err := l.device.IsFile(ctx, source)
	if err != nil {
		return "", err
	}
	if !file {
		return "", fmt.Errorf("Could not open file %s", source)
	}
	if err = l.device.PullFile(ctx, source, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func startPerfettoTrace(ctx context.Context, perfettocmd string, b bind.DeviceWithShell, ready task.Task, opts *perfetto_pb.TraceConfig, fileLoc string) (shell.Process, chan error, error) {
	reader, stdout := io.Pipe()
	data, err := proto.Marshal(opts)
	readyOnce := task.Once(ready)

	if err != nil {
		return nil, nil, err
	}

	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
			readyOnce(ctx)
			switch e {
			default:
				log.E(ctx, "[perfetto] Read error %v", e)
				fail <- e
			case io.EOF:
				fail <- nil
				return
			case nil:
				log.I(ctx, "[perfetto] %s", strings.TrimSuffix(line, "\n"))
			}
		}
	})
	log.E(ctx, "Starting perfetto trace %v", fileLoc)

	proc, err := b.Shell(perfettocmd, "-c", "-", "-o", fileLoc).
		Read(bytes.NewReader(data)).
		Capture(stdout, stdout).
		Verbose().
		Start(ctx)
	if err != nil {
		stdout.Close()
		return nil, nil, err
	}

	return proc, fail, nil
}

// Start sets up a Perfetto trace
func Start(ctx context.Context, d bind.DeviceWithShell, abi *device.ABI, opts *service.TraceOptions) (*Process, error) {
	ctx = log.Enter(ctx, "start")

	var setup perfettoDeviceSetup
	if dev, ok := d.(remotessh.Device); ok {
		setup = &remoteSetup{dev, abi}
	} else {
		setup = &localSetup{d, abi}
	}

	tempdir, cleanup, err := setup.makeTempDir(ctx)
	if err != nil {
		return nil, err
	}

	perfettoTraceFile := tempdir + "/gapis-trace"
	readyFunc := task.Noop()

	var fail chan error
	var cmd shell.Process
	c, err := setup.initializePerfetto(ctx, tempdir)
	if err != nil {
		return nil, err
	}

	if !opts.DeferStart {
		cmd, fail, err = startPerfettoTrace(ctx, c, d, readyFunc, opts.PerfettoConfig, perfettoTraceFile)
		if err != nil {
			return nil, err
		}
	}

	return &Process{
		device:      d,
		config:      opts.PerfettoConfig,
		deferred:    opts.DeferStart,
		tracefile:   perfettoTraceFile,
		perfettoCmd: c,
		setup:       setup,
		cleanup:     cleanup,
		err:         fail,
		perf:        cmd,
	}, nil
}

// Capture starts the perfetto capture.
func (p *Process) Capture(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64) (int64, error) {
	tmp, err := file.Temp()
	if err != nil {
		return 0, log.Err(ctx, err, "Failed to create a temp file")
	}

	// Signal that we are ready to start.
	atomic.StoreInt64(written, 1)

	if p.deferred {
		if !start.Wait(ctx) {
			return 0, log.Err(ctx, nil, "Cancelled")
		}
		cmd, fail, err := startPerfettoTrace(ctx, p.perfettoCmd, p.device, ready, p.config, p.tracefile)
		if err != nil {
			return 0, err
		}
		p.perf = cmd
		p.err = fail
	}

	wait := make(chan error, 1)
	crash.Go(func() {
		wait <- p.perf.Wait(ctx)
	})

	select {
	case err = <-p.err:
		return 0, err
	case err = <-wait:
		// Do nothing
	case <-stop:
		log.E(ctx, "Stopping %v", p.tracefile)
		if err = p.device.Shell("killall", "-2", "perfetto").Run(ctx); err != nil {
			break
		}
		err = <-wait
	}

	if err != nil {
		return 0, err
	}
	traceLoc, err := p.setup.pullTrace(ctx, p.tracefile, tmp.System())
	if err != nil {
		return 0, err
	}
	defer p.cleanup.Invoke(ctx)

	f := file.Abs(traceLoc)
	size := f.Info().Size()
	atomic.StoreInt64(written, size)
	fh, err := os.Open(f.System())
	if err != nil {
		return 0, log.Err(ctx, err, fmt.Sprintf("Failed to open %s", traceLoc))
	}
	return io.Copy(w, fh)
}
