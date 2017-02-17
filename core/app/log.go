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

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/note"
)

const (
	// FatalSeverity Is the level at which logging causes panics.
	FatalSeverity = severity.Critical
)

func LogDefaults() LogFlags {
	return LogFlags{
		Level:  severity.Notice,
		Style:  note.DefaultStyle,
		Stacks: true,
	}
}

func wrapHandler(to note.Handler) (note.Handler, func()) {
	// TODO: should this be a buffered channel?
	out, closer := note.Channel(note.Sorter(to), 0)
	return func(page note.Page) error {
		err := out(page)
		if severity.FindLevel(page) <= FatalSeverity {
			closer()
			panic(FatalExit)
		}
		return err
	}, closer
}

func prepareContext(flags *LogFlags) (log.Context, func(), task.CancelFunc) {
	// now build the initial root context
	handler, closeLogs := wrapHandler(output.Std(flags.Style))
	output.Default = handler
	ctx := context.Background()
	ctx = severity.Filter(ctx, flags.Level)
	ctx, cancel := context.WithCancel(ctx)
	return log.Wrap(ctx), closeLogs, task.CancelFunc(cancel)
}

func updateContext(legacy log.Context, flags *LogFlags, closeLogs func()) (log.Context, func()) {
	ctx := legacy.Unwrap()
	ctx = severity.Filter(ctx, flags.Level)
	if flags.Stacks {
		ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{
			Condition: stacktrace.OnError,
			Source: stacktrace.TrimTop(stacktrace.MatchPackage("runtime"),
				stacktrace.TrimBottom(stacktrace.MatchPackage("github.com/google/gapid/core/context/jot"),
					stacktrace.Capture)),
		})
	}
	if flags.File != "" {
		// Create the server logfile.
		os.MkdirAll(filepath.Dir(flags.File), 0755)
		file, err := os.Create(flags.File)
		if err != nil {
			panic(err)
		}
		jot.To(ctx).With("File", flags.File).Print("Switching to log file")
		// Build the logging context
		handler, closer := wrapHandler(flags.Style.Scribe(file))
		closeLogs()
		closeLogs = func() {
			closer()
			file.Close()
		}
		ctx = output.NewContext(ctx, handler)
	} else if flags.Style.Name != note.DefaultStyle.Name {
		handler, closer := wrapHandler(output.Std(flags.Style))
		closeLogs()
		closeLogs = closer
		ctx = output.NewContext(ctx, handler)
	}
	return log.Wrap(ctx), closeLogs
}
