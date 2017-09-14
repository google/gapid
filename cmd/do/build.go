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
	"context"
	"fmt"
	"strings"
)

func doGlob(ctx context.Context, cfg Config) {
	outPath := cfg.out()
	run(ctx, outPath, cfg.CMakePath, nil, "-DFORCE_GLOB=true", outPath.System())
}

func doBuild(ctx context.Context, cfg Config, options BuildOptions, targets ...string) {
	doGapic := len(targets) == 0 // Building everything implies gapic.
	for i, t := range targets {
		if t == "gapic" {
			doGapic = true
			options.Install = true
			// Remove gapic from the targets list.
			copy(targets[i:], targets[i+1:])
			targets = targets[:len(targets)-1]
			break
		}
	}

	runCMake := options.Rescan || !cfg.cacheFile().Exists()
	if runCMake {
		doCMake(ctx, cfg, options, targets...)
		doGenerate(ctx, cfg)
	}
	if len(targets) == 0 || options.Install {
		targets = append(targets, "install")
	}
	doNinja(ctx, cfg, options, targets...)

	if doGapic {
		gapic(ctx, cfg).build(ctx, options)
	}
}

func doCMake(ctx context.Context, cfg Config, options BuildOptions, targets ...string) {
	env := env(cfg)
	args := []string{
		"-GNinja",
		"-DCMAKE_MAKE_PROGRAM=" + cfg.NinjaPath.Slash(),
		"-DCMAKE_BUILD_TYPE=" + strings.Title(cfg.Flavor.String()),
		"-DINSTALL_PREFIX=" + cfg.pkg().Slash(),
		"-DCMAKE_Go_COMPILER=" + goExePath.Slash(),
	}
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
	if cfg.ArmLinuxGapii {
		args = append(args, "-DARMLINUX_GAPII=1")
	}
	switch options.Test {
	case RunTests:
	case BuildTests:
		args = append(args, "-DNO_RUN_TESTS=1")
	case DisableTests:
		args = append(args, "-DNO_TESTS=1")
	}
	if options.BuildNum > 0 {
		args = append(args, fmt.Sprintf("-DGAPID_BUILD_NUMBER=%d", options.BuildNum))
	}
	if options.BuildSha != "" {
		args = append(args, "-DGAPID_BUILD_SHA="+options.BuildSha)
	}
	args = append(args, srcRoot.System())
	run(ctx, cfg.out(), cfg.CMakePath, env, args...)
}

func doGenerate(ctx context.Context, cfg Config) {
	doNinja(ctx, cfg, BuildOptions{}, "generate")
}

func doNinja(ctx context.Context, cfg Config, options BuildOptions, targets ...string) {
	env := env(cfg)
	args := targets
	if options.DryRun {
		args = append([]string{"-n"}, args...)
	}
	if options.NumJobs > 0 {
		args = append([]string{fmt.Sprintf("-j%d", options.NumJobs)}, args...)
	}
	if options.Verbose {
		args = append([]string{"-v"}, args...)
	}
	// Make sure Ninja calling cmake will find go (i.e. for llvm).
	env.AddPathStart("PATH", goExePath.Parent().System())
	run(ctx, cfg.out(), cfg.NinjaPath, env, args...)
}
