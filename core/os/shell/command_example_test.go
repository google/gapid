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

//go:build (linux && !android) || (darwin && !ios)

package shell_test

import (
	"bytes"
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

func newCtx() context.Context {
	ctx := context.Background()
	ctx = log.PutClock(ctx, log.NoClock)
	ctx = log.PutHandler(ctx, log.Normal.Handler(log.Stdout()))
	ctx = log.Enter(ctx, "Example")
	return ctx
}

// This example shows how to run a command, wait for it to complete and check if it succeeded.
func ExampleSilent() {
	ctx := newCtx()
	err := shell.Command("echo", "Hello from the shell").Run(ctx)
	if err != nil {
		log.F(ctx, true, "Unable to say hello: %v", err)
	}
	log.I(ctx, "Done")
	// Output:
	//I: [Example] Done
}

// This example shows how to run a command, with it's standard out and error being logged.
func ExampleVerbose() {
	ctx := newCtx()
	err := shell.Command("echo", "Hello", "from the shell").Verbose().Run(ctx)
	if err != nil {
		log.F(ctx, true, "Unable to say hello: %v", err)
	}
	// Output:
	//I: [Example] Exec: echo Hello "from the shell"
	//I: [Example] <echo> Hello from the shell
}

// This example shows how to run a command and capture it's output.
func ExampleCapture() {
	ctx := newCtx()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	echo := shell.Command("echo").Verbose().Capture(stdout, stderr)
	echo.With("First echo").Run(ctx)
	echo.With("Second echo").Run(ctx)
	log.I(ctx, "<{<%s}>", stdout.String())
	log.W(ctx, "<!{<%s}!>", stderr.String())
	// Output:
	//I: [Example] Exec: echo "First echo"
	//I: [Example] <echo> First echo
	//I: [Example] Exec: echo "Second echo"
	//I: [Example] <echo> Second echo
	//I: [Example] <{<First echo
	//Second echo
	//}>
	//W: [Example] <!{<}!>
}
