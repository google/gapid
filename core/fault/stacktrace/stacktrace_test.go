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

package stacktrace_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
)

/*
--- LINE PADDING --
--- LINE PADDING --
*/

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// be very careful re-ordering the top of this file, the stack trace captures line numbers
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

//go:noinline
func nested3() stacktrace.Callstack { return nested2() }

//go:noinline
func nested2() stacktrace.Callstack { return nested1() }

//go:noinline
func nested1() stacktrace.Callstack { return stacktrace.Capture() }

//go:noinline
func init() {
	for i := range traces {
		traces[i].stack = traces[i].fun()
	}
}

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// Line nubmers below this are not captured
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

type traceEntry struct {
	fun    func() stacktrace.Callstack
	stack  stacktrace.Callstack
	expect [][]string
}

func (e traceEntry) Filtered() stacktrace.Callstack {
	return e.stack.Filter(stacktrace.Trim(filter))
}

var (
	filter = stacktrace.MatchPackage("core/.*")

	traces = []traceEntry{{
		fun: stacktrace.Capture,
		expect: [][]string{{
			"⇒ core/fault/stacktrace/stacktrace_test.go@46:init.0",
		}},
	}, {
		fun: nested1,
		expect: [][]string{{
			"⇒ core/fault/stacktrace/stacktrace_test.go@41:nested1",
			"⇒ core/fault/stacktrace/stacktrace_test.go@46:init.0",
		}},
	}, {
		fun: nested3,
		expect: [][]string{
			{
				"⇒ core/fault/stacktrace/stacktrace_test.go@41:nested1",
				"⇒ core/fault/stacktrace/stacktrace_test.go@38:nested2",
				"⇒ core/fault/stacktrace/stacktrace_test.go@35:nested3",
				"⇒ core/fault/stacktrace/stacktrace_test.go@46:init.0",
			},
		},
	}}
)

func TestCapture(t *testing.T) {
	assert := assert.To(t)
	for _, test := range traces {
		entries := test.Filtered().Entries()
		lines := make([]string, len(entries))
		for i, e := range entries {
			lines[i] = e.String()
		}
		assert.For("stack").ThatSlice(lines).EqualsOneOf(test.expect)
	}
}

func TestPassThroughFilter(t *testing.T) {
	ctx := log.Testing(t)
	c := traces[0].Filtered()
	noFilter := func(e []stacktrace.Entry) []stacktrace.Entry { return e }
	assert.For(ctx, "stack").ThatString(c.Filter(noFilter).String()).Equals(c.String())
}
