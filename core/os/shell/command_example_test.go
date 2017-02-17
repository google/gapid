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

// +build linux darwin windows

package shell_test

import (
	"bytes"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

// This example shows how to run a command, wait for it to complete and check if it succeeded.
func ExampleSilent() {
	ctx := log.Background().PreFilter(log.NoLimit).Filter(log.Pass).Handler(log.Stdout(log.Normal))
	ctx = ctx.Enter("Example")
	err := shell.Command("echo", "Hello from the shell").Run(ctx)
	if err != nil {
		jot.Fail(ctx, err, "Unable to say hello")
	}
	ctx.Print("Done")
	// Output:
	//Info:Example:Done
}

// This example shows how to run a command, with it's standard out and error being logged.
func ExampleVerbose() {
	ctx := log.Background().PreFilter(log.NoLimit).Filter(log.Pass).Handler(log.Stdout(log.Normal))
	ctx = ctx.Enter("Example")
	err := shell.Command("echo", "Hello", "from the shell").Verbose().Run(ctx)
	if err != nil {
		jot.Fail(ctx, err, "Unable to say hello")
	}
	// Output:
	//Info:Example:Exec:Command=echo Hello "from the shell"
	//stdout:Hello from the shell
}

// This example shows how to run a command and capture it's output.
func ExampleCapture() {
	ctx := log.Background().PreFilter(log.NoLimit).Filter(log.Pass).Handler(log.Stdout(log.Normal))
	ctx = ctx.Enter("Example")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	echo := shell.Command("echo").Verbose().Capture(stdout, stderr)
	echo.With("First echo").Run(ctx)
	echo.With("Second echo").Run(ctx)
	ctx.Printf("<{<%s}>", stdout.String())
	ctx.Warning().Logf("<!{<%s}!>", stderr.String())
	// Output:
	//Info:Example:Exec:Command=echo "First echo"
	//stdout:First echo
	//Info:Example:Exec:Command=echo "Second echo"
	//stdout:Second echo
	//Info:Example:<{<First echo
	//Second echo
	//}>
	//Warning:Example:<!{<}!>
}
