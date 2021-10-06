// Copyright (C) 2021 Google Inc.
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

package ffx

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/fuchsia"
	"github.com/google/gapid/core/os/shell"
)

// binding represents an attached Fuchsia device.
type binding struct {
	bind.Simple
}

// verify that binding implements Device
var _ fuchsia.Device = (*binding)(nil)

// FFX is the path to the ffx executable, or an empty string if the ffx
// executable was not found.
var FFX file.Path

func ffx() (file.Path, error) {
	if !FFX.IsEmpty() {
		return FFX, nil
	}

	if ffx_env := os.Getenv("FUCHSIA_FFX_PATH"); ffx_env != "" {
		FFX = file.Abs(ffx_env)
		if !FFX.IsExecutable() {
			return file.Path{}, fmt.Errorf("ffx path: %s is not executable", FFX)
		}
		return FFX, nil
	}

	return file.Path{}, fmt.Errorf("FUCHSIA_FFX_PATH is not set. " +
		"The \"ffx\" tool, from the Fuchsia SDK, is required for Fuchsia device profiling.\n")
}

func (b *binding) Command(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(deviceTarget{b})
}

func (b *binding) augmentFFXCommand(cmd shell.Cmd) (shell.Cmd, error) {
	exe, err := ffx()
	if err != nil {
		return cmd, err
	}

	// Adjust the ffx command to use a specific target:
	//     "ffx -t <target> <cmd.Name> <cmd.Args...>"
	old := cmd.Args
	cmd.Args = make([]string, 0, len(old)+4)
	cmd.Args = append(cmd.Args, "-t", b.To.Serial)
	cmd.Args = append(cmd.Args, cmd.Name)
	cmd.Args = append(cmd.Args, old...)
	cmd.Name = exe.System()
	fmt.Println(cmd)

	// And delegate to the normal local target
	return cmd, nil
}

func (b *binding) CanTrace() bool { return true }

func (b *binding) TraceProviders(ctx context.Context) ([]string, error) {
	exe, err := ffx()
	if err != nil {
		return nil, err
	}

	cmd, err := b.augmentFFXCommand(shell.Command(exe.System(), "trace", "list-providers"))

	if err != nil {
		return nil, err
	}

	providersStdOut, err := cmd.Call(ctx)

	if strings.Contains(providersStdOut, "No devices found") {
		return nil, ErrNoDeviceList
	}
	lines := strings.Split(providersStdOut, "\n")
	var providers []string
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			tokens := strings.Split(line, " ")
			if len(tokens) == 2 {
				providers = append(providers, tokens[1])
			} else {
				return nil, ErrTraceProvidersFormat
			}
		}
	}
	return providers, nil
}

// StartTrace starts a Fuchsia trace.
func (b *binding) StartTrace(ctx context.Context, traceFile file.Path, traceCategories []string, stop task.Signal, ready task.Task) error {
	var categoriesArg string
	for _, category := range traceCategories {
		if len(categoriesArg) > 0 {
			categoriesArg += ","
		}
		categoriesArg += category
	}

	var cmd shell.Cmd
	if len(categoriesArg) > 0 {
		cmd = b.Command("trace", "start", "--output", traceFile.System(), "--categories", categoriesArg)
	} else {
		cmd = b.Command("trace", "start", "--output", traceFile.System())
	}

	stdout, err := cmd.Call(ctx)

	if err != nil {
		return log.Err(ctx, err, stdout)
	}

	if strings.Contains(stdout, "No devices found") {
		return ErrNoDeviceList
	}
	return nil
}

// StopTrace stops a Fuchsia trace.
func (b *binding) StopTrace(ctx context.Context, traceFile file.Path) error {
	cmd := b.Command("trace", "stop", "--output", traceFile.System())

	stdout, err := cmd.Call(ctx)
	if err != nil {
		return log.Err(ctx, err, stdout)
	}

	if strings.Contains(stdout, "No devices found") {
		return ErrNoDeviceList
	}

	return nil
}

type deviceTarget struct{ b *binding }

func (t deviceTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	cmd, err := t.b.augmentFFXCommand(cmd)
	if err != nil {
		return nil, err
	}

	return shell.LocalTarget.Start(cmd)
}

func (t deviceTarget) String() string {
	return "command:" + t.b.String()
}
