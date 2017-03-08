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
	"os"
	"path/filepath"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

const (
	// FatalSeverity Is the level at which logging causes panics.
	FatalSeverity = log.Fatal
)

func logDefaults() LogFlags {
	return LogFlags{
		Level:  log.Info,
		Style:  log.Normal,
		Stacks: true,
	}
}

func wrapHandler(to log.Handler) log.Handler {
	h := log.Channel(to, 0) // TODO: should this be a buffered channel?
	return fatalHandler{h}
}

type fatalHandler struct{ log.Handler }

func (h fatalHandler) Handle(m *log.Message) {
	h.Handler.Handle(m)
	if m.Severity >= FatalSeverity {
		h.Handler.Close()
		panic(FatalExit)
	}
}

func prepareContext(flags *LogFlags) (context.Context, func(), task.CancelFunc) {
	// now build the initial root context
	handler := wrapHandler(flags.Style.Handler(os.Stdout, os.Stderr))
	ctx := context.Background()
	ctx = log.PutFilter(ctx, log.SeverityFilter(flags.Level))
	ctx = log.PutHandler(ctx, handler)
	ctx, cancel := context.WithCancel(ctx)
	return ctx, handler.Close, task.CancelFunc(cancel)
}

func updateContext(ctx context.Context, flags *LogFlags, closeLogs func()) (context.Context, func()) {
	ctx = log.PutFilter(ctx, log.SeverityFilter(flags.Level))
	if flags.Stacks {
		ctx = log.PutStacktracer(ctx, log.SeverityStacktracer(log.Error))
		// TODO
		// ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{
		// 	Condition: stacktrace.OnError,
		// 	Source: stacktrace.TrimTop(stacktrace.MatchPackage("runtime"),
		// 		stacktrace.TrimBottom(stacktrace.MatchPackage("github.com/google/gapid/core/context/jot"),
		// 			stacktrace.Capture)),
		// })
	}
	if flags.File != "" {
		// Create the server logfile.
		os.MkdirAll(filepath.Dir(flags.File), 0755)
		file, err := os.Create(flags.File)
		if err != nil {
			panic(err)
		}
		log.I(ctx, "Logging to: %v", flags.File)
		// Build the logging context
		handler := wrapHandler(flags.Style.Handler(file, file))
		ctx = log.PutHandler(ctx, handler)
		closeLogs()
		closeLogs = func() {
			handler.Close()
			file.Close()
		}
	} else {
		handler := wrapHandler(flags.Style.Handler(os.Stdout, os.Stderr))
		ctx = log.PutHandler(ctx, handler)
		closeLogs()
		closeLogs = handler.Close
	}
	return ctx, closeLogs
}
