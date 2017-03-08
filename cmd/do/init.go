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

	"github.com/google/gapid/core/os/file"
)

func doInit(ctx context.Context, options InitOptions) Config {
	cfg := fetchValidConfig(ctx, ConfigOptions{})
	// We do the build version first in case we need to delete the output
	initBuildVersion(ctx, &cfg, options)
	// Make sure the output folders all exist and the current symlinks are up to date
	initOutput(ctx, &cfg, options)
	// Make sure all our prerequisites are ready
	// TODO: git submodule management should go in here
	return cfg
}

func initOutput(ctx context.Context, cfg *Config, options InitOptions) {
	file.Mkdir(cfg.bin())
	file.Mkdir(cfg.pkg())
	must(file.Relink(cfg.OutRoot.Join("current"), cfg.out(), file.Junction))
	must(file.Relink(cfg.OutRoot.Join("bin"), cfg.bin(), file.Junction))
	must(file.Relink(cfg.OutRoot.Join("pkg"), cfg.pkg(), file.Junction))
}

func initBuildVersion(ctx context.Context, cfg *Config, options InitOptions) {
	// Has the do major version changed since the last build?
	lastMajor, _ := cfg.loadBuildVersion()
	if lastMajor != versionMajor {
		fmt.Printf("Major changes made to CMake. Forcing a full rebuild.\n")
		doClean(ctx, CleanOptions{}, *cfg)
	}
	file.Mkdir(cfg.out())
	cfg.storeBuildVersion()
}
