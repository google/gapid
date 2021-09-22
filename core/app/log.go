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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

const logChanBufferSize = 100

// LogHandler is the primary application logger target.
// It is assigned to the main context on startup and is closed on shutdown.
var LogHandler log.Indirect

func logDefaults() LogFlags {
	return LogFlags{
		Level:  log.Info,
		Style:  log.Normal,
		Stacks: true,
	}
}

func wrapHandler(to log.Handler) log.Handler {
	to = log.Channel(to, logChanBufferSize)
	return log.NewHandler(func(m *log.Message) {
		to.Handle(m)
		if m.StopProcess {
			to.Close()
			panic(FatalExit)
		}
	}, to.Close)
}

func prepareContext(flags *LogFlags) context.Context {
	// now build the initial root context
	process := file.Abs(os.Args[0]).NoExt().Basename()
	LogHandler.SetTarget(wrapHandler(flags.Style.Handler(log.Std())), true)
	ctx := context.Background()
	ctx = log.PutProcess(ctx, process)
	ctx = log.PutFilter(ctx, log.SeverityFilter(flags.Level))
	ctx = log.PutHandler(ctx, &LogHandler)
	return ctx
}

func updateContext(ctx context.Context, flags *LogFlags) context.Context {
	ctx = log.PutFilter(ctx, log.SeverityFilter(flags.Level))
	if flags.Stacks {
		ctx = log.PutStacktracer(ctx, log.SeverityStacktracer(log.Error))
	}

	if flags.File != "" {
		if file := createLogFile(ctx, flags); file != nil {
			// Build the file logging context.
			handler := flags.Style.Handler(func(s string, _ log.Severity) {
				file.WriteString(s)
				file.WriteString("\n")
			})
			handler = log.OnClosed(handler, func() { file.Close() })
			handler = wrapHandler(handler)
			if old, _ := LogHandler.SetTarget(handler, false); old != nil {
				old.Close()
			}
		}
	}
	return ctx
}

func createLogFile(ctx context.Context, flags *LogFlags) *os.File {
	path := file.Abs(flags.File)
	dir, name, ext := path.Smash()

	if name == "" {
		name = "gapis"
		ext = ".log"
		path = dir.Join(name + ext)
	}

	os.MkdirAll(dir.System(), 0755)
	for i := 0; i < 10; i++ {
		file, err := os.Create(path.System())
		if err == nil {
			log.I(ctx, "Logging to: %v", path.System())
			return file
		}

		// Try a different path next.
		path = dir.Join(fmt.Sprintf("%s-%d%s", name, i, ext))
	}

	log.E(ctx, "Failed to create log file "+flags.File)
	return nil
}
