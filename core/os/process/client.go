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
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/shell"
)

var portPattern = regexp.MustCompile(`^Bound on port '(\d+)'$`)

type portWatcher struct {
	portChan chan<- string
	stdout   io.Writer
	fragment string
	done     bool
	portFile string
}

func (w *portWatcher) getPortFromFile() (string, bool) {
	if contents, err := ioutil.ReadFile(w.portFile); err == nil {
		s := string(contents)
		match := portPattern.FindStringSubmatch(s)
		if match != nil {
			os.Remove(w.portFile)
			return match[1], true
		}
	}
	return "", false
}

func (w *portWatcher) waitForFile(ctx context.Context) {
	if w.portFile == "" {
		return
	}

	os.Remove(w.portFile)
	hb := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hb.C:
			if port, ok := w.getPortFromFile(); ok {
				w.portChan <- port
				close(w.portChan)
				w.done = true
				return
			}
		}
	}
}

func (w *portWatcher) Write(b []byte) (n int, err error) {
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
}

// Start runs the application with the given path and options, waits for
// the "Bound on port {port}" string to be printed to stdout, and then returns
// the port number.
func Start(ctx context.Context, name string, opts StartOptions) (int, error) {
	// Append extra environment variable values
	errChan := make(chan error, 1)
	portChan := make(chan string, 1)
	stdout := &portWatcher{
		portChan: portChan,
		stdout:   opts.Stdout,
		portFile: opts.PortFile,
	}

	c, cancel := task.WithCancel(ctx)
	defer cancel()
	crash.Go(func() { stdout.waitForFile(c) })
	crash.Go(func() {
		cmd := shell.
			Command(name, opts.Args...).
			In(opts.WorkingDir).
			Env(opts.Env).
			Capture(stdout, opts.Stderr)
		if opts.Verbose {
			cmd = cmd.Verbose()
		}
		errChan <- cmd.Run(ctx)
	})

	select {
	case port := <-portChan:
		return strconv.Atoi(port)
	case err := <-errChan:
		return 0, err
	}
}

func Connect(port int, authToken auth.Token) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, err
	}
	if err := auth.Write(conn, authToken); err != nil {
		return nil, err
	}
	return conn, err
}
