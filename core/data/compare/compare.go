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

package compare

import (
	"fmt"
	"reflect"
)

const (
	// Missing is used when values are found to be missing from a container
	Missing = invalid("Missing")
)

var (
	// LimitReached the value panic is given to abort a comparison early.
	LimitReached = diffLimit{}
)

type diffLimit struct{}
type invalid string

func (i invalid) Format(f fmt.State, r rune) { fmt.Fprint(f, "⚠ ", string(i)) }

// Register assigns the function f with signature func(comparator, T, T) to
// be used as the default comparator for instances of type T.
// f may return nothing or a CompareAction.
// Register will panic if f does not match the expected signature, or if a
// custom comparator for type T has already been registered.
func Register(f interface{}) {
	globalCustom.Register(f)
}

// Compare delivers all the differences it finds to the specified Handler.
// If the reference and value are equal, the handler will never be invoked.
func Compare(reference, value interface{}, handler Handler) {
	compare(reference, value, handler, globalCustom)
}

func compare(reference, value interface{}, handler Handler, custom *Custom) {
	defer func() {
		if err := recover(); err != nil {
			if _, isLimit := err.(diffLimit); !isLimit {
				// unhandled panic, propagate
				panic(err)
			}
		}
	}()
	t := Comparator{Path: Path{}, Handler: handler, seen: seen{}, custom: custom}
	t.Compare(reference, value)
}

// DeepEqual compares a value against a reference and returns true if they are
// equal.
func DeepEqual(reference, value interface{}) bool {
	var d test
	Compare(reference, value, d.set)
	return !bool(d)
}

// Diff returns the differences between the reference and the value.
// The maximum number of differences is controlled by limit, which must be >0.
// If they compare equal, the length of the returned slice will be 0.
func Diff(reference, value interface{}, limit int) []Path {
	diffs := make(collect, 0, limit)
	Compare(reference, value, diffs.add)
	return ([]Path)(diffs)
}

// IsNil returns true if o is nil (either directly or a boxed nil).
func IsNil(o interface{}) bool {
	if o == nil {
		return true
	}
	v := reflect.ValueOf(o)
	switch v.Kind() {
	case reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface:
		return v.IsNil()
	}
	return false
}
