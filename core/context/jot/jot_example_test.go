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
	"context"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func myNestedFunc(ctx context.Context, name string) {
	ctx = memo.Enter(ctx, "nested")
	jot.Debug(ctx).With("name", name).Print("being questioned")
	jot.Info(ctx).With("name", name).With("state", "hungry").Send()
}

func myFunc(ctx context.Context) {
	ctx = memo.Enter(ctx, "printer")
	jot.Notice(ctx).Print("Who are you and what do you want?")
	myNestedFunc(ctx, "George")
	jot.Warning(ctx).Jot("George is angry").Cause(fault.Const("Denial of food")).Print("")
}

func Example_Simple() {
	ctx := context.Background()
	ctx = severity.Filter(ctx, severity.Debug)
	//ctx = output.NewContext(ctx, note.Sorter(output.Stdout(note.Normal)))
	myFunc(ctx)
	// Output:
	// Notice:printer:Who are you and what do you want?
	// Debug:printer->nested:being questioned:name=George
	// Info:printer->nested:name=George,state=hungry
	// Warning:printer:George is angry:⦕Denial of food⦖
}

type normal struct{}

func (normal) String() string { return "normal" }

func Example_Variations() {
	ctx := context.Background()
	ctx = severity.Filter(ctx, severity.Info)
	ctx = severity.NewContext(ctx, severity.Notice)
	ctx = output.NewContext(ctx, note.Sorter(output.Stdout(note.Normal)))
	jot.At(ctx, severity.Info).With("state", "long").Send()
	jot.Info(ctx).With("state", normal{}).Send()
	jot.Error(ctx).With("state", "late").Send()
	jot.Debug(ctx).With("state", "ignored").Send()
	jot.Notice(ctx).With("number", 1).Send()
	jot.Warning(ctx).With("pi", 1.142).Send()
	jot.Warning(ctx).With("message", `Such a long string, this will get truncated at some point, right?. Wait for it. Wait.
Ok not quite yet. Now maybe?
No, not quite.
How are you? The weather is pretty dull outside. I wonder what happened to summer.
My cat's breath smells of cat food.
Once a fella met a fella in a field of fitches. Said a fella to a fella, can a fella tell a fella where a fella itches?
Probably.
A bean, a bean, a half a bean, a bean, a bean and a half.
Oh what's that? Could it be the end of this string?
Yeah, I think it really might be. Horray!`).
		Send()
	jot.Warning(ctx).With("message",
		"Such a long string, this will get truncated at some point, right?. Nope this one will just fit, just wait and see. The end.").
		Send()
	jot.Print(ctx, "You are simple")
	jot.Printf(ctx, "You are %s", "printf")

	// Output:
	// Info:state=long
	// Info:state=normal
	// Error:state=late
	// Notice:number=1
	// Warning:pi=1.142
	// Warning:message=Such a long string, this will get truncated at some point, right?. Wait for it. Wait.
	// Ok not quite yet. Now maybe?
	// No, not qu...
	// Warning:message=Such a long string, this will get truncated at some point, right?. Nope this one will just fit, just wait and see. The end.
	// Notice:You are simple
	// Notice:You are printf
}
