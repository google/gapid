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
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/text/note"
)

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// be very careful modifying this file, the entire file is line number sensitive
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

const (
	failure = fault.Const("Because we fail")
	misery  = fault.Const("Sadness")
)

func myErrorFunc(ctx context.Context, name string) error {
	ctx = memo.Enter(ctx, "Leaf")
	jot.Fail(ctx, misery, "In the deep")
	return cause.Explain(ctx, failure, "always")
}

func myFailingFunc(ctx context.Context) error {
	ctx = memo.Enter(ctx, "Failing")
	err := myErrorFunc(ctx, "dancing")
	return cause.Explain(ctx, err, "doing my thing")
}

func Example_Stacktrace() {
	ctx := context.Background()
	ctx = output.NewContext(ctx, note.Sorter(output.Stdout(note.Detailed)))
	ctx = stacktrace.CaptureOn(ctx, stacktrace.Controls{
		Condition: stacktrace.OnError,
		Source: stacktrace.TrimTop(stacktrace.MatchPackage("testing"),
			stacktrace.TrimBottom(stacktrace.MatchPackage("github.com/google/gapid/core/context/jot"),
				stacktrace.Capture)),
	})
	jot.Info(ctx).Tag("fun").Print("Normal message")
	err := myFailingFunc(ctx)
	jot.Fail(ctx, err, "Error message")
	// Output:
	// Info:fun:Normal message
	// Error:Failing->Leaf:In the deep
	//     ⦕Sadness⦖
	//     myErrorFunc ⇒ stack_example_test.go@40
	//         myFailingFunc      ⇒ stack_example_test.go@46
	//         Example_Stacktrace ⇒ stack_example_test.go@60
	// Error:Error message
	//     Failing:doing my thing
	//         Failing->Leaf:always
	//             ⦕Because we fail⦖
	//     Example_Stacktrace ⇒ stack_example_test.go@61
}
