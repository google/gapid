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

package main

import (
	"runtime"

	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

func env(cfg Config) *shell.Env {
	path := []string{}

	if runtime.GOOS == "windows" {
		if !cfg.MSYS2Path.IsEmpty() {
			path = append(path,
				cfg.MSYS2Path.Join("usr/bin").System(),     // Required for sh.exe and other unixy tools.
				cfg.MSYS2Path.Join("mingw64/bin").System(), // Required to pick up DLLs
			)
		}

		// Add windows and system32 to path
		cmd, err := file.FindExecutable("cmd.exe")
		if err != nil {
			panic("Couldn't find cmd.exe on PATH")
		}
		system32 := cmd.Parent()
		windows := system32.Parent()
		path = append(path, system32.System(), windows.System())
		path = append(path, exePaths(cfg, "node.exe", "adb.exe", "git.exe", "ffmpeg.exe")...)
	} else {
		path = append(path, exePaths(cfg,
			"sh", "uname", "sed", "clang", "gcc", "node", "adb",
		)...)
	}

	path = append(path,
		cfg.bin().System(),
		cfg.JavaHome.Join("bin").System(),
	)

	env := shell.CloneEnv()
	env.Unset("PATH")
	env.AddPathEnd("PATH", path...)
	return env
}

func exePaths(cfg Config, exes ...string) []string {
	path := []string{}
	added := map[file.Path]bool{}
	for _, name := range exes {
		// First search PATH
		exe, err := file.FindExecutable(name)
		if err != nil {
			// Not found on PATH, try ${AndroidSDKRoot}/platform-tools.
			exe, err = file.FindExecutable(cfg.AndroidSDKRoot.Join("platform-tools", name).System())
			if err != nil {
				continue
			}
		}
		dir := exe.Parent()
		if !added[dir] {
			path = append(path, dir.System())
			added[dir] = true
		}
	}
	return path
}
