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

package app

import (
	"context"
	"os"
	"runtime/pprof"

	"github.com/google/gapid/core/log"
)

func applyProfiler(ctx context.Context, flags *ProfileFlags) func() {
	closers := []func(){}
	if flags.CPU != "" {
		log.I(ctx, "CPU profiling enabled")
		f, err := os.Create(flags.CPU)
		if err != nil {
			log.F(ctx, true, "CPU profiling failed to start.\nError: %v", err)
		}
		pprof.StartCPUProfile(f)
		closers = append(closers, func() {
			pprof.StopCPUProfile()
			f.Close()
			log.I(ctx, "CPU profile written")
		})
	}
	return func() {
		for _, closer := range closers {
			closer()
		}
	}
}
