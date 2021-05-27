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

package process

import (
	"context"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
)

var portPattern = regexp.MustCompile(`^Bound on port '(\d+)'$`)

// NewPortWatcher creates and returns an initialized port watcher
func NewPortWatcher(portChan chan<- string, opts *StartOptions) *PortWatcher {
	return &PortWatcher{
		portChan: portChan,
		stdout:   opts.Stdout,
		portFile: opts.PortFile,
		device:   opts.Device,
	}
}

// PortWatcher will wait for the given port to become available
// on the given device. It will look at the given stdout,
// for the lines Bound on port <portnum>, and in the given
// portFile for the same lines. If either is found, the port
// will be written to portChan. Otherwise an error will be
// written to the channel.
type PortWatcher struct {
	portChan chan<- string
	stdout   io.Writer
	fragment string
	done     bool
	portFile string
	device   bind.Device
}

func (w *PortWatcher) getPortFromFile(ctx context.Context) (string, bool) {
	if contents, err := w.device.FileContents(ctx, w.portFile); err == nil {
		s := string(contents)
		match := portPattern.FindStringSubmatch(s)
		if match != nil {
			w.device.RemoveFile(ctx, w.portFile)
			return match[1], true
		}
	}
	return "", false
}

// Waits until the port file is found, or the context
// is cancelled
func (w *PortWatcher) WaitForFile(ctx context.Context) {
	if w.portFile == "" {
		return
	}

	hb := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hb.C:
			if port, ok := w.getPortFromFile(ctx); ok {
				w.portChan <- port
				close(w.portChan)
				w.done = true
				return
			}
		}
	}
}

func (w *PortWatcher) Write(b []byte) (n int, err error) {
	if stdout := w.stdout; stdout != nil {
		stdout.Write(b)
	}
	if w.done {
		return len(b), nil
	}
	s := w.fragment + string(b)
	start := 0
	for i, c := range s {
		if c == '\n' || c == '\r' {
			line := s[start:i]
			start = i + 1
			match := portPattern.FindStringSubmatch(line)
			if match != nil {
				w.portChan <- match[1]
				close(w.portChan)
				w.done = true
				return len(b), nil
			}
		}
	}
	w.fragment = s[start:]
	return len(b), nil
}

// StartOptions holds the options that can be passed to Start.
type StartOptions struct {
	// Command line arguments for starting the process.
	Args []string

	// Environment variables for starting the process.
	Env *shell.Env

	// Standard output pipe for the new process.
	Stdout io.Writer

	// Standard error pipe for the new process.
	Stderr io.Writer

	// Should all stderr and and stdout also be logged to the logger?
	Verbose bool

	// PortFile, if not "", is a file that can be search if we can
	// not find the output on stdout
	PortFile string

	// If not "", the working directory for this command
	WorkingDir string

	// IgnorePort, if true, then will not wait for the port to become
	// available
	IgnorePort bool

	// Device, which device should this be started on
	Device bind.DeviceWithShell
}

// StartOnDevice runs the application on the given remote device,
// with the given path and options, waits for the "Bound on port {port}" string
// to be printed to stdout, and then returns the port number.
func StartOnDevice(ctx context.Context, name string, opts StartOptions) (int, error) {
	// Append extra environment variable values
	errChan := make(chan error, 1)
	portChan := make(chan string, 1)
	c, cancel := task.WithCancel(ctx)
	defer cancel()

	stdout := opts.Stdout
	if !opts.IgnorePort {
		portWatcher := NewPortWatcher(portChan, &opts)
		stdout = portWatcher
		crash.Go(func() {
			portWatcher.WaitForFile(c)
		})
	}

	crash.Go(func() {
		cmd := opts.Device.Shell(name, opts.Args...).
			In(opts.WorkingDir).
			Env(opts.Env).
			Capture(stdout, opts.Stderr)
		if opts.Verbose {
			cmd = cmd.Verbose()
		}
		errChan <- cmd.Run(ctx)
	})
	if !opts.IgnorePort {
		select {
		case port := <-portChan:
			p, err := strconv.Atoi(port)
			if err != nil {
				return 0, err
			}
			return opts.Device.SetupLocalPort(ctx, p)
		case err := <-errChan:
			return 0, err
		}
	}
	return 0, nil
}
