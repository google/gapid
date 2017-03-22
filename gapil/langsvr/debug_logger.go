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

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

const maxLogHistory = 5

// debugLogger will write all stdin, stdout and log messages to log files next
// to the executable when enabled.
type debugLogger struct {
	enabled    bool
	stdin      io.Reader // The real stdin
	stdout     io.Writer // The real stdout
	stdinLog   io.ReadWriteCloser
	stdoutLog  io.ReadWriteCloser
	msgLog     io.ReadWriteCloser
	logHandler log.Handler
	stop       func()
}

func (d *debugLogger) Read(p []byte) (n int, err error) {
	n, err = d.stdin.Read(p)
	if d.stdinLog != nil {
		d.stdinLog.Write(p[:n])
	}
	return
}

func (d *debugLogger) Write(p []byte) (n int, err error) {
	if d.stdoutLog != nil {
		d.stdoutLog.Write(p)
	}
	return d.stdout.Write(p)
}

func (d *debugLogger) bind(ctx context.Context) context.Context {
	return log.PutHandler(ctx, debugLogHandler{d})
}

type debugLogHandler struct{ d *debugLogger }

func (h debugLogHandler) Handle(m *log.Message) {
	if h.d.logHandler != nil {
		h.d.logHandler.Handle(m)
	}
}

func (h debugLogHandler) Close() {
	if h.d.logHandler != nil {
		h.d.logHandler.Close()
	}
}

func (d *debugLogger) setEnabled(enabled bool) error {
	if enabled == d.enabled {
		return nil
	}
	d.enabled = enabled
	if enabled {
		d.stop()
		for name, io := range map[string]*io.ReadWriteCloser{
			"stdin.log":   &d.stdinLog,
			"stdout.log":  &d.stdoutLog,
			"message.log": &d.msgLog,
		} {
			path := file.ExecutablePath().Parent().Join(name)
			rotateLogs(path)
			f, err := os.Create(path.System())
			if err != nil {
				return err
			}
			*io = f
		}
		logHandler := log.Channel(log.Normal.Handler(func(s string, _ log.Severity) {
			d.msgLog.Write([]byte(s))
			d.msgLog.Write([]byte("\n"))
		}), 0)
		d.logHandler = logHandler
		d.stop = func() {
			logHandler.Close()
			d.stdinLog.Close()
			d.stdoutLog.Close()
			d.msgLog.Close()
			d.stop = func() {}
		}
	} else {
		d.stop()
	}
	return nil
}

// rotateLogs renames the log file at path by inserting '-1' between the file
// name and extension. If a file already exists with the new path, then that
// file is also renamed with the numeric part incremented. This renaming
// continues for maxLogHistory number of files. If there are more than
// maxLogHistory that need renaming then the last file is simply deleted.
func rotateLogs(path file.Path) error {
	pathNoExt, ext := path.SplitExt()

	ithPath := func(i int) file.Path {
		if i > 0 {
			return file.Abs(fmt.Sprintf("%v-%d%v", pathNoExt, i, ext))
		}
		return path
	}

	for i := maxLogHistory - 1; i >= 0; i-- {
		src := ithPath(i)
		if !src.Exists() {
			continue
		}
		if i < maxLogHistory-1 {
			if err := file.Move(ithPath(i+1), src); err != nil {
				return err
			}
		} else {
			if err := file.Remove(src); err != nil {
				return err
			}
		}
	}
	return nil
}
