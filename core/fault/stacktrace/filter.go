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

import (
	"regexp"
)

// Matcher is a predicate for stack entries.
type Matcher func(Entry) bool

// Filter returns a filtered set of stack entries.
type Filter func([]Entry) []Entry

// MatchPackage returns a predicate that matches the specified package by
// regular expression.
func MatchPackage(pkg string) Matcher {
	re := regexp.MustCompile(pkg)
	return func(entry Entry) bool {
		return re.MatchString(entry.Function.Package)
	}
}

// MatchFunction returns a predicate that matches the specified function by
// regular expression.
func MatchFunction(fun string) Matcher {
	re := regexp.MustCompile(fun)
	return func(entry Entry) bool {
		return re.MatchString(entry.Function.Name)
	}
}

// Filter returns a new capture filtered by f.
func (c Callstack) Filter(f ...Filter) Callstack {
	entries := And(f...)(c.Entries())
	out := make(Callstack, len(entries))
	for i, e := range entries {
		out[i] = e.PC
	}
	return out
}

// And returns a filter where all of the filters need to pass.
func And(filters ...Filter) Filter {
	return func(entries []Entry) []Entry {
		for _, f := range filters {
			entries = f(entries)
		}
		return entries
	}
}

// Trim returns a filter combining the TrimTop and TrimBottom filters with m.
func Trim(m Matcher) Filter {
	return And(TrimTop(m), TrimBottom(m))
}

// TrimTop returns a filter that removes all the deepest entries that doesn't
// satisfy m.
func TrimTop(m Matcher) Filter {
	return func(entries []Entry) []Entry {
		for i, e := range entries {
			if m(e) {
				return entries[i:]
			}
		}
		return []Entry{}
	}
}

// TrimBottom returns a filter that removes all the shallowest entries that
// doesn't satisfy m.
func TrimBottom(m Matcher) Filter {
	return func(entries []Entry) []Entry {
		for i := len(entries) - 1; i >= 0; i-- {
			if m(entries[i]) {
				return entries[:i+1]
			}
		}
		return []Entry{}
	}
}
