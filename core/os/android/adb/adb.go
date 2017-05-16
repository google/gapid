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

package adb

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

// ADB is the path to the adb executable, or an empty string if the adb
// executable was not found.
var ADB file.Path

func adb() (file.Path, error) {
	if !ADB.IsEmpty() {
		return ADB, nil
	}

	exe := "adb"

	search := []string{exe}

	// If ANDROID_HOME is set, build a fully rooted path
	// We still want to call LookPath to pick up the extension and check the binary exists
	if home := os.Getenv("ANDROID_HOME"); home != "" {
		search = append(search, filepath.Join(home, "platform-tools", exe))
	}

	for _, path := range search {
		if p, err := file.FindExecutable(path); err == nil {
			ADB = p
			return ADB, nil
		}
	}

	return file.Path{}, fmt.Errorf("adb could not be found from ANDROID_HOME or PATH\n"+
		"ANDROID_HOME: %v\n"+
		"PATH: %v\n"+
		"search: %v",
		os.Getenv("ANDROID_HOME"), os.Getenv("PATH"), search)

}

// Command is a helper that builds a shell.Cmd with the device as its target.
func (b *binding) Command(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(deviceTarget{b})
}

// Shell is a helper that builds a shell.Cmd with d.ShellTarget() as its target
func (b *binding) Shell(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(shellTarget{b})
}

func (b *binding) prepareADBCommand(cmd shell.Cmd, useShell bool) (shell.Process, error) {
	exe, err := adb()
	if err != nil {
		return nil, err
	}
	// Adjust the command to: "adb -s device [shell] <cmd.Name> <cmd.Args...>"
	old := cmd.Args
	cmd.Args = make([]string, 0, len(old)+4)
	cmd.Args = append(cmd.Args, "-s", b.To.Serial)
	if useShell {
		cmd.Args = append(cmd.Args, "shell")
	}
	cmd.Args = append(cmd.Args, cmd.Name)
	cmd.Args = append(cmd.Args, old...)
	cmd.Name = exe.System()
	// And delegate to the normal local target
	return shell.LocalTarget.Start(cmd)
}

type deviceTarget struct{ b *binding }

func (t deviceTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	return t.b.prepareADBCommand(cmd, false)
}

func (t deviceTarget) String() string {
	return "command:" + t.b.String()
}

type shellTarget struct{ b *binding }

func (t shellTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	return t.b.prepareADBCommand(cmd, true)
}

func (t shellTarget) String() string {
	return "shell:" + t.b.String()
}
