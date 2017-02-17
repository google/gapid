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

package severity_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func TestNoLimit(t *testing.T) {
	ctx := context.Background()
	for level := severity.Emergency; level <= severity.Debug; level++ {
		testFilter(t, ctx, "nolimit", level, true)
	}
}

func TestLimit(t *testing.T) {
	ctx := context.Background()
	for limit := severity.Emergency; limit <= severity.Debug; limit++ {
		ctx := severity.Filter(ctx, limit)
		for level := severity.Emergency; level <= severity.Debug; level++ {
			testFilter(t, ctx, limit, level, level <= limit)
		}
	}
}

func TestNames(t *testing.T) {
	l := severity.Level(-1)
	for _, test := range []struct {
		level severity.Level
		name  string
	}{
		{severity.Emergency, "Emergency"},
		{severity.Alert, "Alert"},
		{severity.Critical, "Critical"},
		{severity.Error, "Error"},
		{severity.Warning, "Warning"},
		{severity.Notice, "Notice"},
		{severity.Info, "Info"},
		{severity.Debug, "Debug"},
		{severity.Level(10), "10"},
	} {
		if test.level.String() != test.name {
			t.Errorf("Name incorrect, expected %q got %q", test.name, test.level.String())
		}
		l.Choose(test.level)
		if test.level != l {
			t.Errorf("Choose failed, expected %q got %q", test.level, l)
		}
	}
}

func TestLevelKey(t *testing.T) {
	ctx := context.Background()
	buf, _ := json.Marshal(severity.LevelKey)
	if string(buf) != `"Level"` {
		t.Errorf("Level key name incorrect, expected \"Level\" got %q", buf)
	}
	ctx = severity.NewContext(ctx, severity.Info)
	keyList := keys.Get(ctx)
	if len(keyList) != 1 || keyList[0] != severity.LevelKey {
		t.Errorf("Level key not in key list, got %v", keyList)
	}
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	severity.DefaultLevel = severity.Info
	got := severity.FromContext(ctx)
	if got != severity.Info {
		t.Errorf("Empty context did not return expected default %q got %q", severity.Info, got)
	}
	severity.DefaultLevel = severity.Emergency
	got = severity.FromContext(ctx)
	if got != severity.Emergency {
		t.Errorf("Changing the default failed %q got %q", severity.Emergency, got)
	}
	ctx = severity.NewContext(ctx, severity.Debug)
	got = severity.FromContext(ctx)
	if got != severity.Debug {
		t.Errorf("Changing the context failed %q got %q", severity.Debug, got)
	}
}

func TestGetFilter(t *testing.T) {
	ctx := context.Background()
	got := severity.GetFilter(ctx)
	if got != severity.DefaultFilter {
		t.Errorf("Empty context did not return expected filter default %q got %q", severity.DefaultFilter, got)
	}
	ctx = severity.Filter(ctx, severity.Error)
	got = severity.GetFilter(ctx)
	if got != severity.Error {
		t.Errorf("Setting the filter failed %q gave %q", severity.Error, got)
	}
}

func TestFindLevel(t *testing.T) {
	ctx := context.Background()
	page := note.Page{}
	severity.DefaultLevel = severity.Info
	got := severity.FindLevel(page)
	if got != severity.Info {
		t.Errorf("Empty page did not return expected default %q got %q", severity.Info, got)
	}
	memo.PrintKey.Transcribe(ctx, &page, "a")
	severity.DefaultLevel = severity.Emergency
	got = severity.FindLevel(page)
	if got != severity.Emergency {
		t.Errorf("Changing the default failed, expected %q got %q", severity.Emergency, got)
	}
	severity.LevelKey.Transcribe(ctx, &page, severity.Debug)
	got = severity.FindLevel(page)
	if got != severity.Debug {
		t.Errorf("Changing the page failed, expected %q got %q", severity.Debug, got)
	}
	memo.PrintKey.Transcribe(ctx, &page, "a")
	got = severity.FindLevel(page)
	if got != severity.Debug {
		t.Errorf("Changing the page hid the severity, expected %q got %q", severity.Debug, got)
	}
}

func TestSection(t *testing.T) {
	ctx := context.Background()
	ctx = memo.Print(ctx, "Message")
	ctx = severity.NewContext(ctx, severity.Error)
	ctx = memo.Tag(ctx, "Tag")
	page := memo.Transcribe(ctx)
	page.Sort()
	got := note.Normal.Print(page)
	const expect = "Error:Tag:Message"
	if expect != got {
		t.Errorf("Expected value %q got %q", expect, got)
	}
}

func testFilter(t *testing.T, ctx context.Context, tag interface{}, level severity.Level, exp bool) {
	ctx = severity.NewContext(ctx, level)
	ctx = memo.Print(ctx, "Message")
	_, ok := memo.From(ctx)
	if ok != exp {
		t.Errorf("%v: Got %v expected %v testing %v", tag, ok, exp, level)
	}
}
