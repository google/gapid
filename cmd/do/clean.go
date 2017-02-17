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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

func doClean(ctx log.Context, options CleanOptions, cfg Config) {
	cleanOutput := true
	if options.Generated {
		cleanOutput = false
		doCleanGenerated(ctx, cfg)
	}
	if cleanOutput {
		doCleanOutput(ctx, cfg)
	}
}

func doCleanOutput(ctx log.Context, cfg Config) {
	if cfg.cacheFile().Exists() {
		file.RemoveAll(cfg.out())
	}
}

func doCleanGenerated(ctx log.Context, cfg Config) {
	doBuild(ctx, cfg, BuildOptions{}, "clean_generated")
	binary := cfg.bin().Join("clean_generated")
	run(ctx, srcRoot, binary, nil, "-log-level", "Info")
}
