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

package jot_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func TestJot(t *testing.T) {
	assert := assert.To(t)
	buf := &bytes.Buffer{}
	ctx := context.Background()
	ctx = output.NewContext(ctx, note.Sorter(note.Normal.Scribe(buf)))

	jot.At(ctx, severity.Warning).Print("testing warning")
	checkBuffer(assert, "at", buf, "Warning:testing warning")

	jot.With(ctx, "A", "a").Printf("testing %d with", 0)
	checkBuffer(assert, "with", buf, "testing 0 with:A=a")

	jot.Jot(ctx, "testing jot").Send()
	checkBuffer(assert, "jot", buf, "testing jot")
	jot.Jotf(ctx, "testing jotf int %d string %s", 4, "four").Send()
	checkBuffer(assert, "jotf", buf, "testing jotf int 4 string four")

	jot.Print(ctx, "testing print")
	checkBuffer(assert, "print", buf, "testing print")
	jot.Printf(ctx, "testing printf float %f string %s", 3.2, "name")
	checkBuffer(assert, "printf", buf, "testing printf float 3.200000 string name")

	jot.Fail(ctx, fault.Const("fail"), "testing fail")
	checkBuffer(assert, "fail", buf, "Error:testing fail:⦕fail⦖")
	jot.Failf(ctx, fault.Const("failf"), "testing failf int %d string %s", 5, "five")
	checkBuffer(assert, "failf", buf, "Error:testing failf int 5 string five:⦕failf⦖")

	jot.Fatal(ctx, fault.Const("fatal"), "testing fatal")
	checkBuffer(assert, "fatal", buf, "Critical:testing fatal:⦕fatal⦖")
	jot.Fatalf(ctx, fault.Const("fatalf"), "testing fatalf int %d string %s", 5, "five")
	checkBuffer(assert, "fatalf", buf, "Critical:testing fatalf int 5 string five:⦕fatalf⦖")

	jot.Error(ctx).Print("testing error")
	checkBuffer(assert, "error", buf, "Error:testing error")
	jot.Errorf(ctx, "testing %d", 6)
	checkBuffer(assert, "errorf", buf, "Error:testing 6")

	jot.Warning(ctx).Print("testing warning")
	checkBuffer(assert, "warning", buf, "Warning:testing warning")
	jot.Warningf(ctx, "testing %d", 7)
	checkBuffer(assert, "warningf", buf, "Warning:testing 7")

	jot.Notice(ctx).Print("testing notice")
	checkBuffer(assert, "notice", buf, "Notice:testing notice")
	jot.Noticef(ctx, "testing %d", 8)
	checkBuffer(assert, "noticef", buf, "Notice:testing 8")

	jot.Info(ctx).Print("testing info")
	checkBuffer(assert, "info", buf, "Info:testing info")
	jot.Infof(ctx, "testing %d", 9)
	checkBuffer(assert, "infof", buf, "Info:testing 9")

	jot.Debug(ctx).Print("testing debug")
	checkBuffer(assert, "debug", buf, "Debug:testing debug")
	jot.Debugf(ctx, "testing %d", 10)
	checkBuffer(assert, "debugf", buf, "Debug:testing 10")
}

func checkBuffer(assert assert.Manager, name string, buf *bytes.Buffer, expect string) {
	assert.For(name).That(buf.String()).Equals(expect)
	buf.Reset()
}
