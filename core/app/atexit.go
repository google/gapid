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
	"syscall"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
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

	interruptHandlers = make(map[int]func())
	lastInterrupt     = int(0)
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

// AddInterruptHandler adds a function that will be called on the next interrupt.
// It returns a function that can be called to remove the value from the list
func AddInterruptHandler(f func()) func() {
	li := lastInterrupt
	lastInterrupt++
	interruptHandlers[li] = f
	return func() {
		delete(interruptHandlers, li)
	}
}

func handleAbortSignals(ctx context.Context, cancel task.CancelFunc) {
	// register a signal handler for exits
	sigchan := make(chan os.Signal)
	// Enable signal interception, no-op if already enabled.
	signal.Notify(sigchan, os.Interrupt, os.Kill, syscall.SIGTERM)
	// Run a goroutine that calls the cancel func if the signal is received
	crash.Go(func() {
		s := <-sigchan
		log.D(ctx, "Handling signal %v", s)

		handled := false
		for _, v := range interruptHandlers {
			handled = true
			v()
		}
		interruptHandlers = make(map[int]func())
		if !handled {
			cancel()
		}
	})
}

func handleCrashSignals(ctx context.Context, cancel task.CancelFunc) {
	crash.Register(func(err interface{}, callstack stacktrace.Callstack) {
		log.F(ctx, false, "Crash: %v (%T)\n%v", err, err, callstack)
	})
}
