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

package fuzz_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/resolver"
)

// TestCrashers checks that each of the crashers reported by the fuzzer no
// longer crash. If the test passes, the crashers directory can be safely
// deleted.
func TestCrashers(t *testing.T) {
	ctx := log.Testing(t)
	files, err := filepath.Glob("./fuzz-wd/crashers/*")
	if err != nil {
		jot.Fail(ctx, err, "failed to find crashers")
		return
	}
	for _, file := range files {
		ctx := ctx.S("file", file)
		if filepath.Ext(file) != "" {
			continue
		}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			jot.Fail(ctx, err, "failed to open file")
			return
		}
		if err := compile(data); err != nil {
			jot.Fail(ctx, err, "crashed")
		}
	}
}

func timebomb(fuse time.Duration) (defuse func()) {
	stop := make(chan struct{})
	go func() {
		select {
		case <-time.Tick(fuse):
			fmt.Println("timeout")
			stack := make([]byte, 1<<16)
			stack = stack[:runtime.Stack(stack, true)]
			fmt.Println(string(stack))
			panic("timeout")
		case <-stop:
		}
	}()
	return func() { close(stop) }
}

func compile(data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 1<<16)
			stack = stack[:runtime.Stack(stack, true)]
			err = fmt.Errorf("%v\n%s", r, string(stack))
		}
	}()
	defer timebomb(time.Second * 3)()
	processor := gapil.Processor{
		Mappings:            resolver.NewMappings(),
		Loader:              gapil.NewDataLoader(data),
		Parsed:              map[string]gapil.ParseResult{},
		Resolved:            map[string]gapil.ResolveResult{},
		ResolveOnParseError: true,
	}
	fmt.Println("Testing: ", string(data))
	processor.Resolve("fuzz")
	fmt.Println("** OKAY ** ")
	return nil
}
