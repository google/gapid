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
	"bufio"
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/google/gapid/core/app/status"
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
	if flags.Mem != "" {
		log.I(ctx, "Mem profiling enabled")
		f, err := os.Create(flags.Mem)
		if err != nil {
			log.F(ctx, true, "Mem profiling failed to start.\nError: %v", err)
		}
		closers = append(closers, func() {
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.F(ctx, true, "Failed to write memory profile: %v", err)
			}
			f.Close()
			log.I(ctx, "Mem profile written")
		})
	}
	if flags.Trace != "" {
		log.I(ctx, "Trace profiling enabled")
		f, err := os.Create(flags.Trace)
		if err != nil {
			log.F(ctx, true, "Trace profiling failed to start.\nError: %v", err)
		}
		writer := bufio.NewWriterSize(f, 1024*1024*1024)
		stop := status.RegisterTracer(writer)
		closers = append(closers, func() {
			stop()
			writer.Flush()
			f.Close()
			log.I(ctx, "Trace profile written")
		})
	}
	if flags.Pprof {
		log.I(ctx, "Enabling pprof on :6060")
		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()
	}
	return func() {
		for _, closer := range closers {
			closer()
		}
	}
}
