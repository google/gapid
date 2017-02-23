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
	}
	if len(targets) == 0 || options.Install {
		targets = append(targets, "install")
	}
	doNinja(ctx, cfg, options, targets...)

	if doGapic {
		gapic(ctx, cfg).build(ctx, options)
	}
}

func doCMake(ctx log.Context, cfg Config, options BuildOptions, targets ...string) {
	args := []string{
		"-GNinja",
		"-DCMAKE_MAKE_PROGRAM=" + cfg.NinjaPath.System(),
		"-DCMAKE_BUILD_TYPE=" + strings.Title(cfg.Flavor.String()),
		"-DINSTALL_PREFIX=" + cfg.pkg().System(),
	}
	args = append(args, "-DCMAKE_Go_COMPILER="+goExePath.System())
	if !cfg.AndroidNDKRoot.IsEmpty() {
		args = append(args, "-DANDROID_NDK_ROOT="+cfg.AndroidNDKRoot.System())
	}
	if !cfg.JavaHome.IsEmpty() {
		args = append(args, "-DJAVA_HOME="+cfg.JavaHome.System())
	}
	if !cfg.AndroidSDKRoot.IsEmpty() {
		args = append(args, "-DANDROID_HOME="+cfg.AndroidSDKRoot.System())
	}
	switch options.Test {
	case RunTests:
	case BuildTests:
		args = append(args, "-DNO_RUN_TESTS=1")
	case DisableTests:
		args = append(args, "-DNO_TESTS=1")
	}
	args = append(args, srcRoot.System())
	env := shell.CloneEnv().AddPathStart("PATH", cfg.bin().System())
	run(ctx, cfg.out(), cfg.CMakePath, env, args...)
}

func doNinja(ctx log.Context, cfg Config, options BuildOptions, targets ...string) {
	args := targets
	if options.DryRun {
		args = append([]string{"-n"}, args...)
	}
	if options.Verbose {
		args = append([]string{"-v"}, args...)
	}
	env := shell.CloneEnv().AddPathStart("PATH", cfg.bin().System())
	run(ctx, cfg.out(), cfg.NinjaPath, env, args...)
}
