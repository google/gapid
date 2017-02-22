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

// The do command wraps CMake, simplifying the building GAPID in common
// configurations.
package main

import (
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

func doGlob(ctx log.Context, cfg Config) {
	outPath := cfg.out()
	run(ctx, outPath, cfg.CMakePath, nil, "-DFORCE_GLOB=true", outPath.System())
}

func doBuild(ctx log.Context, cfg Config, options BuildOptions, targets ...string) {
	runCMake := options.Rescan || !cfg.cacheFile().Exists()
	if runCMake {
		doCMake(ctx, cfg, options, targets...)
	}
	if len(targets) == 0 || options.Install {
		targets = append(targets, "install")
	}
	doNinja(ctx, cfg, options, targets...)
}

func doCMake(ctx log.Context, cfg Config, options BuildOptions, targets ...string) {
	env := shell.CloneEnv().AddPathStart("PATH",
		cfg.bin().System(),
		cfg.JavaHome.Join("bin").System(),
	)
	args := []string{
		"-GNinja",
		"-DCMAKE_MAKE_PROGRAM=" + cfg.NinjaPath.Slash(),
		"-DCMAKE_BUILD_TYPE=" + strings.Title(cfg.Flavor.String()),
		"-DINSTALL_PREFIX=" + cfg.pkg().System(),
	}
	args = append(args, "-DCMAKE_Go_COMPILER="+goExePath.Slash())
	if !cfg.AndroidNDKRoot.IsEmpty() {
		args = append(args, "-DANDROID_NDK_ROOT="+cfg.AndroidNDKRoot.Slash())
	}
	if !cfg.JavaHome.IsEmpty() {
		args = append(args, "-DJAVA_HOME="+cfg.JavaHome.Slash())
	}
	if !cfg.AndroidSDKRoot.IsEmpty() {
		args = append(args, "-DANDROID_HOME="+cfg.AndroidSDKRoot.Slash())
	}
	if !cfg.PythonPath.IsEmpty() {
		args = append(args, "-DPYTHON_EXECUTABLE="+cfg.PythonPath.Slash())
	}
	if !cfg.MSYS2Path.IsEmpty() {
		args = append(args, "-DMSYS2_PATH="+cfg.MSYS2Path.Slash())
		env.AddPathStart("PATH", cfg.MSYS2Path.Join("mingw64/bin").System()) // Required to pick up DLLs
	}
	switch options.Test {
	case RunTests:
	case BuildTests:
		args = append(args, "-DNO_RUN_TESTS=1")
	case DisableTests:
		args = append(args, "-DNO_TESTS=1")
	}
	args = append(args, srcRoot.System())
	run(ctx, cfg.out(), cfg.CMakePath, env, args...)
}

func doNinja(ctx log.Context, cfg Config, options BuildOptions, targets ...string) {
	env := shell.CloneEnv().AddPathStart("PATH",
		cfg.bin().System(),
		cfg.JavaHome.Join("bin").System(),
	)
	if !cfg.MSYS2Path.IsEmpty() {
		env.AddPathStart("PATH", cfg.MSYS2Path.Join("mingw64/bin").System()) // Required to pick up DLLs
	}
	args := targets
	if options.DryRun {
		args = append([]string{"-n"}, args...)
	}
	if options.Verbose {
		args = append([]string{"-v"}, args...)
	}
	if options.Verbose {
		args = append([]string{"-j1"}, args...)
	}
	run(ctx, cfg.out(), cfg.NinjaPath, env, args...)
}
