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
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/text/note"
)

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
	top    = stacktrace.MatchFunction("github.com/google/gapid/core/fault/stacktrace_test", "init")
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

func testStyle(assert assert.Manager, style note.Style, page note.Page, expect string) {
	assert.For(style.Name).That(style.Print(page)).Equals(strings.TrimSpace(expect))
}

func hasTrace(page note.Page) bool {
	for _, section := range page {
		for _, item := range section.Content {
			if reflect.TypeOf(item.Key) == reflect.TypeOf(stacktrace.Key) {
				return true
			}
		}
	}
	return false
}

func TestCaptureOn(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = severity.NewContext(ctx, severity.Error)
	for _, test := range traces {
		ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{Source: test.FilteredCapture})
		trace, _ := memo.From(ctx)
		assert.For("trace found").ThatSlice(trace).IsNotEmpty()
		trace.Sort()
		testStyle(assert, note.Brief, trace, test.brief)
		testStyle(assert, note.Normal, trace, test.normal)
		testStyle(assert, note.Detailed, trace, test.detailed)
	}
	ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{Source: traces[0].FilteredCapture, Condition: func(context.Context) bool { return false }})
	page, _ := memo.From(ctx)
	assert.For("false trace condition").Printf("Expect trace not found").Test(!hasTrace(page))
	ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{Source: func() []stacktrace.Entry { return nil }, Condition: func(context.Context) bool { return true }})
	page, _ = memo.From(ctx)
	assert.For("empty trace source").Printf("Expect trace not found").Test(!hasTrace(page))
	ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{Source: stacktrace.Capture, Condition: func(context.Context) bool { return true }})
	page, _ = memo.From(ctx)
	assert.For("normal trace").Printf("Expect trace found").Test(hasTrace(page))
}

func TestOnError(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	for level := severity.Level(0); level <= 10; level = severity.Level(int(level) + 1) {
		ctx := severity.NewContext(ctx, level)
		got := stacktrace.OnError(ctx)
		expect := false
		switch level {
		case severity.Emergency,
			severity.Critical,
			severity.Alert,
			severity.Error:
			expect = true
		}
		assert.For("OnError at %s", level).That(got).Equals(expect)
	}
}
