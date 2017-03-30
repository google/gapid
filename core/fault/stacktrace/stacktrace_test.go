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
)

/*
--- LINE PADDING --
--- LINE PADDING --
--- LINE PADDING --
*/

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// be very careful re-ordering the top of this file, the stack trace captures line numbers
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

func invokeNestedCapture() []stacktrace.Entry { return nestedCapture() }
func nestedCapture() []stacktrace.Entry       { return stacktrace.Capture() }
func init() {
	for i := range traces {
		traces[i].cap = traces[i].fun()
	}
}

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// Line nubmers below this are not captured
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

type traceEntry struct {
	fun      stacktrace.Source
	cap      []stacktrace.Entry
	expect   []string
	brief    string
	normal   string
	detailed string
}

func (e traceEntry) RawCapture() []stacktrace.Entry { return e.cap }
func (e traceEntry) FilteredCapture() []stacktrace.Entry {
	return stacktrace.TrimTop(top, stacktrace.TrimBottom(bottom, e.RawCapture))()
}

var (
	top    = stacktrace.MatchFunction("init")
	bottom = stacktrace.MatchPackage("github.com/google/gapid/core/fault/stacktrace")

	traces = []traceEntry{{
		fun: stacktrace.Capture,
		expect: []string{
			"⇒ stacktrace_test.go@38:init.1",
		},
		brief:  `Error`,
		normal: `Error:init.1⇒{stacktrace_test.go@38}`,
		detailed: `
Error
    init.1 ⇒ stacktrace_test.go@38`,
	}, {
		fun: nestedCapture,
		expect: []string{
			"⇒ stacktrace_test.go@35:nestedCapture",
			"⇒ stacktrace_test.go@38:init.1",
		},
		brief:  `Error`,
		normal: `Error:nestedCapture⇒{stacktrace_test.go@35}`,
		detailed: `
Error
    nestedCapture ⇒ stacktrace_test.go@35
        init.1 ⇒ stacktrace_test.go@38`,
	}, {
		fun: invokeNestedCapture,
		expect: []string{
			"⇒ stacktrace_test.go@35:nestedCapture",
			"⇒ stacktrace_test.go@34:invokeNestedCapture",
			"⇒ stacktrace_test.go@38:init.1",
		},
		brief:  `Error`,
		normal: `Error:nestedCapture⇒{stacktrace_test.go@35}`,
		detailed: `
Error
    nestedCapture ⇒ stacktrace_test.go@35
        invokeNestedCapture ⇒ stacktrace_test.go@34
        init.1              ⇒ stacktrace_test.go@38`,
	}}
)

func TestCapture(t *testing.T) {
	assert := assert.To(t)
	for _, test := range traces {
		cap := test.FilteredCapture()
		assert.For("stack length").ThatSlice(cap).IsLength(len(test.expect))
		for i, expect := range test.expect {
			assert.For("stack entry %d", i).That(cap[i].String()).Equals(expect)
		}
	}
}

func TestFilter(t *testing.T) {
	assert := assert.To(t)
	cap := traces[0].RawCapture
	raw := cap()
	badTrimTop := stacktrace.TrimTop(stacktrace.MatchPackage("not a package"), cap)()
	badTrimBottom := stacktrace.TrimBottom(stacktrace.MatchPackage("not a package"), cap)()
	goodTrimTop := stacktrace.TrimTop(top, cap)()
	goodTrimBottom := stacktrace.TrimBottom(bottom, cap)()
	assert.For("invalid bottom trim").ThatSlice(raw).IsLength(len(badTrimBottom))
	assert.For("invalid top trim").ThatSlice(raw).IsLength(len(badTrimTop))
	assert.For("valid bottom trim").ThatSlice(goodTrimBottom).IsLength(5)
	assert.For("valid top trim").ThatSlice(goodTrimTop).IsLength(3)
}
