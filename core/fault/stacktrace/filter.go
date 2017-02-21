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

package stacktrace

type (
	// Source is the signature for something that returns a stacktrace.
	Source func() []Entry

	// Matcher is a predicate for stack entries.
	Matcher func(Entry) bool
)

// MatchPackage returns a predicate that matches the specified package.
func MatchPackage(pkg string) Matcher {
	return func(entry Entry) bool {
		return entry.Function.Package == pkg
	}
}

// MatchFunction returns a predicate that matches the specified function.
func MatchFunction(fun string) Matcher {
	return func(entry Entry) bool {
		return entry.Function.Name == fun
	}
}

// TrimBottom trims stack entries from the bottom of the trace.
// It trims from the first matching entry down to the end of the trace.
func TrimBottom(match Matcher, source Source) Source {
	return func() []Entry {
		stack := source()
		for i := len(stack) - 2; i >= 0; i-- {
			if match(stack[i]) {
				return stack[i+1:]
			}
		}
		return stack
	}
}

// TrimTop trims stack entries from the top of the trace.
// It trims from the last matching entry up to the start of the trace.
func TrimTop(match Matcher, source Source) Source {
	return func() []Entry {
		stack := source()
		for i, s := range stack[:len(stack)-1] {
			if match(s) {
				return stack[:i]
			}
		}
		return stack
	}
}
