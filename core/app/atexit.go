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
	"os/signal"
	"time"

	"github.com/google/gapid/core/fault/stacktrace"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
)

// ExitCode is the type for named return values from the application main entry point.
type ExitCode int

const (
	// SuccessExit is the exit code for succesful exit.
	SuccessExit ExitCode = iota
	// FatalExit is the exit code if something logs at a fatal severity (critical or higher by default)
	FatalExit
	// UsageExit is the exit code if the usage function was invoked
	UsageExit
)

var (
	// CleanupTimeout is the time to wait for all cleanup signals to fire when shutting down.
	CleanupTimeout = time.Second * 10

	events = task.Events{}
)

// AddCleanup calls f when the context is cancelled.
// Application will wait (for a maximum of CleanupTimeout) for f to complete
// before terminiating the application.
func AddCleanup(ctx context.Context, f func()) {
	signal, done := task.NewSignal()
	crash.Go(func() {
		defer done(ctx)
		<-task.ShouldStop(ctx)
		f()
	})
	AddCleanupSignal(signal)
}

// AddCleanupSignal adds a signal the app should wait on when shutting down.
// The signal will automatically be dropped when it is fired, no need unregister it.
func AddCleanupSignal(s ...task.Event) {
	events.Add(s...)
}

// WaitForCleanup waits for all the cleanup signals to fire, or the cleanup timeout to expire,
// whichever comes first.
func WaitForCleanup(ctx context.Context) bool {
	return events.TryWait(ctx, CleanupTimeout)
}

func handleAbortSignals(cancel task.CancelFunc) {
	// register a signal handler for exits
	sigchan := make(chan os.Signal)
	// Enable signal interception, no-op if already enabled.
	// Note: for Unix, these signals translate to SIGINT and SIGKILL.
	signal.Notify(sigchan, os.Interrupt, os.Kill)
	// Run a goroutine that calls the cancel func if the signal is received
	crash.Go(func() {
		<-sigchan
		cancel()
	})
}

func handleCrashSignals(cancel task.CancelFunc) {
	crash.Register(func(interface{}, stacktrace.Callstack) {
		cancel()
	})
}
