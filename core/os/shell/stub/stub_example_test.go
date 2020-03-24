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

package stub_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/os/shell/stub"
)

// TestNothing exists because some tools don't like packages that don't have any tests in them.
func TestNothing(t *testing.T) {}

func newCtx() context.Context {
	ctx := context.Background()
	ctx = log.PutClock(ctx, log.NoClock)
	ctx = log.PutHandler(ctx, log.Normal.Handler(log.Stdout()))
	ctx = log.Enter(ctx, "Example")
	return ctx
}

// This example shows how to use a stub command target to fake an starting error.
func ExampleAlways() {
	ctx := newCtx()
	s, err := shell.Command("echo", "Hello from the shell").On(stub.Respond("Hello")).Call(ctx)
	if err != nil {
		log.F(ctx, true, "Unable to say hello. Error: ", err)
	}
	log.I(ctx, s)
	log.I(ctx, "Done")
	// Output:
	//I: [Example] Hello
	//I: [Example] Done
}

// This example shows how to use simple exact matches with a fallback to the command echo.
func ExampleHello() {
	ctx := newCtx()
	target := stub.OneOf(
		stub.RespondTo(`echo "Hello from the shell"`, "Nice to meet you"),
		stub.Echo{},
	)
	s, err := shell.Command("echo", "Hello from the shell").On(target).Call(ctx)
	if err != nil {
		log.F(ctx, true, "Unable to say hello. Error: ", err)
	}
	log.I(ctx, s)
	s, err = shell.Command("echo", "Goodbye now").On(target).Call(ctx)
	if err != nil {
		log.F(ctx, true, "Unable to say goodbye. Error: ", err)
	}
	log.I(ctx, s)
	log.I(ctx, "Done")
	// Output:
	//I: [Example] Nice to meet you
	//I: [Example] Goodbye now
	//I: [Example] Done
}

// This example shows how to use simple exact matches with a fallback to the command echo.
func ExampleRegexp() {
	ctx := newCtx()
	target := stub.OneOf(
		stub.Regex(`smalltalk`, stub.Respond("Nice to meet you")),
		stub.Echo{},
	)
	s, _ := shell.Command("echo", "Hello").On(target).Call(ctx)
	log.I(ctx, s)
	s, _ = shell.Command("echo", "Insert smalltalk here").On(target).Call(ctx)
	log.I(ctx, s)
	s, _ = shell.Command("echo", "Goodbye").On(target).Call(ctx)
	log.I(ctx, s)
	log.I(ctx, "Done")
	// Output:
	//I: [Example] Hello
	//I: [Example] Nice to meet you
	//I: [Example] Goodbye
	//I: [Example] Done
}

// This example shows how to use a stub command target to fake an starting error.
func ExampleStartError() {
	ctx := newCtx()
	target := stub.OneOf(
		stub.Match(`echo Goodbye`, &stub.Response{StartErr: log.Err(ctx, nil, "bad command")}),
		stub.Echo{},
	)
	output, err := shell.Command("echo", "Hello").On(target).Call(ctx)
	if err != nil {
		log.E(ctx, "Unable to say hello. Error: %v", err)
	}
	log.I(ctx, output)
	err = shell.Command("echo", "Goodbye").On(target).Run(ctx)
	if err != nil {
		log.E(ctx, "Unable to say goodbye. Error: %v", err)
	}
	log.I(ctx, "Done")
	// Output:
	// I: [Example] Hello
	// E: [Example] Unable to say goodbye. Error: Failed to start process
	//    Cause: bad command
	// I: [Example] Done
}

// This example shows how to use a stub command target to fake an blocking process, and cancelling it.
func ExampleBlockingCancel() {
	ctx := newCtx()
	ctx, cancel := task.WithCancel(ctx)
	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()
	response := &stub.Response{WaitErr: log.Err(ctx, nil, "Cancelled")}
	response.WaitSignal, response.KillTask = task.NewSignal()
	target := stub.OneOf(stub.Match(`echo "Hello from the shell"`, response))
	err := shell.Command("echo", "Hello from the shell").On(target).Run(ctx)
	if err != nil {
		log.E(ctx, "Unable to say hello. Error: %v", err)
	}
	log.I(ctx, "Done")
	// Output:
	// E: [Example] Unable to say hello. Error: Process returned error
	//    Cause: Cancelled
	// I: [Example] Done
}

// This example shows what happens if you attempt a command that has no response.
func ExampleUnmatchedCommand() {
	ctx := newCtx()
	target := stub.OneOf(
		stub.RespondTo(`echo "Hello from the shell"`, "Nice to meet you"),
	)
	err := shell.Command("echo", "Not hello from the shell").On(target).Run(ctx)
	if err != nil {
		log.E(ctx, "Unable to say hello. Error: %v", err)
	}
	log.I(ctx, "Done")
	// Output:
	// E: [Example] Unable to say hello. Error: Failed to start process
	//    Cause: unmatched:echo "Not hello from the shell"
	// I: [Example] Done
}
