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

	"github.com/google/gapid/core/os/file"
)

func doGapit(ctx context.Context, cfg Config, options GapitOptions, args ...string) {
	doRunTarget(ctx, cfg, options.BuildAndRunOptions, "gapit", args...)
}

func doRobot(ctx context.Context, cfg Config, options RobotOptions, args ...string) {
	if options.WD.IsEmpty() {
		options.WD = cfg.OutRoot.Join("robot")
	}
	robotArgs := []string{}
	if options.ServerAddress != "" {
		robotArgs = append(robotArgs, "-serveraddress", options.ServerAddress)
	}
	robotArgs = append(robotArgs, args...)
	doRunTarget(ctx, cfg, options.BuildAndRunOptions, "robot", robotArgs...)
}

func doRunTarget(ctx context.Context, cfg Config, options BuildAndRunOptions, target string, args ...string) {
	doBuild(ctx, cfg, options.BuildOptions, target)

	if target == "gapic" {
		gapic(ctx, cfg).run(ctx, options.RunOptions, args...)
	} else {
		doRun(ctx, cfg, options.RunOptions, target, args...)
	}
}

func doGo(ctx context.Context, cfg Config, options RunOptions, args ...string) {
	if options.WD.IsEmpty() {
		options.WD = file.Abs("")
	}
	file.Mkdir(options.WD)
	run(ctx, options.WD, goExePath, nil, args...)
}

func doRun(ctx context.Context, cfg Config, options RunOptions, name string, args ...string) {
	if options.WD.IsEmpty() {
		options.WD = file.Abs("")
	}
	env := env(cfg)
	if !cfg.AndroidNDKRoot.IsEmpty() {
		env.Set("ANDROID_NDK_ROOT", cfg.AndroidNDKRoot.System())
	}
	if !cfg.JavaHome.IsEmpty() {
		env.Set("JAVA_HOME", cfg.JavaHome.System())
	}
	if !cfg.AndroidSDKRoot.IsEmpty() {
		env.Set("ANDROID_HOME", cfg.AndroidSDKRoot.System())
	}
	file.Mkdir(options.WD)
	//TODO: windows support?
	binary := cfg.bin().Join(name)
	run(ctx, options.WD, binary, env, args...)
}
