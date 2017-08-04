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

package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/log"
)

// Cmd holds the configuration to run an external command.
//
// A Cmd can be run any number of times, and new commands may be derived from existing ones.
type Cmd struct {
	// Name is the name of the command to run
	Name string
	// Args is the arguments handed to the command, it should not include the command itself.
	Args []string
	// Target is the target to execute the command on
	// If left as nil, this will default to LocalTarget.
	Target Target
	// Verbosity makes the command echo it's stdout and stderr to the supplied logging context.
	// It will also log the command itself as it starts.
	Verbosity bool
	// Dir sets the working directory for the command.
	Dir string
	// Stdout is the writer to which the command will write it's standard output if set.
	Stdout io.Writer
	// Stdout is the writer to which the command will write it's standard error if set.
	Stderr io.Writer
	// Stdin is the reader from which the command will read it's standard input if set.
	Stdin io.Reader
	// Environment is the processes environment, if set.
	Environment *Env
}

// Command returns a Cmd with the specified command and arguments set.
func Command(name string, args ...string) Cmd {
	return Cmd{Name: name, Args: args}
}

// On returns a copy of the Cmd with the Target set to target.
func (cmd Cmd) On(target Target) Cmd {
	cmd.Target = target
	return cmd
}

// Verbose returns a copy of the Cmd with the Verbosity flag set to true.
func (cmd Cmd) Verbose() Cmd {
	cmd.Verbosity = true
	return cmd
}

// In returns a copy of the Cmd with the Dir set to dir.
func (cmd Cmd) In(dir string) Cmd {
	cmd.Dir = dir
	return cmd
}

// Capture returns a copy of the Cmd with Stdout and Stderr set.
func (cmd Cmd) Capture(stdout, stderr io.Writer) Cmd {
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd
}

// Read returns a copy of the Cmd with Stdin set.
func (cmd Cmd) Read(stdin io.Reader) Cmd {
	cmd.Stdin = stdin
	return cmd
}

// Env returns a copy of the Cmd with the Environment set to env.
func (cmd Cmd) Env(env *Env) Cmd {
	cmd.Environment = env
	return cmd
}

// With returns a copy of the Cmd with the args added to the end of Args.
func (cmd Cmd) With(args ...string) Cmd {
	old := cmd.Args
	cmd.Args = make([]string, len(cmd.Args)+len(args))
	copy(cmd.Args, old)
	copy(cmd.Args[len(old):], args)
	return cmd
}

// Run executes the command, and blocks until it completes or the context is cancelled.
func (cmd Cmd) Run(ctx context.Context) error {
	// Deliberately a value receiver so the cmd object can be updated prior to execution
	if cmd.Target == nil {
		cmd.Target = LocalTarget
	} else if cmd.Target != LocalTarget {
		ctx = log.V{"On": cmd.Target}.Bind(ctx)
	}

	if cmd.Dir != "" {
		ctx = log.V{"Dir": cmd.Dir}.Bind(ctx)
	}
	// build our stdout and stderr handling
	var logStdout, logStderr io.WriteCloser
	if cmd.Verbosity {
		ctx := log.PutProcess(ctx, filepath.Base(cmd.Name))
		logStdout = log.From(ctx).Writer(log.Info)
		defer logStdout.Close()
		if cmd.Stdout != nil {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, logStdout)
		} else {
			cmd.Stdout = logStdout
		}
		logStderr = log.From(ctx).Writer(log.Error)
		defer logStderr.Close()
		if cmd.Stderr != nil {
			cmd.Stderr = io.MultiWriter(cmd.Stderr, logStderr)
		} else {
			cmd.Stderr = logStderr
		}
	}
	// Ready to start
	if cmd.Verbosity {
		log.I(ctx, "Exec: %v", cmd)
	}
	process, err := cmd.Target.Start(cmd)
	if err != nil {
		return log.From(ctx).Err(err, "Failed to start process")
	}
	err = process.Wait(ctx)
	if err != nil {
		return log.From(ctx).Err(err, "Process returned error")
	}
	return nil
}

// Call executes the command, capturing its output.
// This is a helper for the common case where you want to run a command, capture all its output into a string and
// see if it succeeded.
func (cmd Cmd) Call(ctx context.Context) (string, error) {
	buf := &bytes.Buffer{}
	err := cmd.Capture(buf, buf).Run(ctx)
	output := strings.TrimSpace(buf.String())
	return output, err
}

func (cmd Cmd) Format(f fmt.State, c rune) {
	fmt.Fprint(f, cmd.Name)
	for _, arg := range cmd.Args {
		fmt.Fprint(f, " ")
		if strings.ContainsRune(arg, ' ') {
			fmt.Fprint(f, `"`, arg, `"`)
		} else {
			fmt.Fprint(f, arg)
		}
	}
}

// SplitEnv splits the given environment variable string into key and values.
func SplitEnv(s string) (key string, vals []string) {
	parts := strings.Split(s, "=")
	if len(parts) != 2 {
		return "", nil
	}
	return parts[0], strings.Split(parts[1], string(os.PathListSeparator))
}

// JoinEnv combines the given key and values into an environment variable string.
func JoinEnv(key string, vals []string) string {
	return key + "=" + strings.Join(vals, string(os.PathListSeparator))
}
