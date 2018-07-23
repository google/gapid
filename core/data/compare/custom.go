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

	"github.com/golang/protobuf/proto"
)

// Action is the optional return value type of functions passes to
// Register and Custom.Register.
type Action int

const (
	// Done is returned by custom comparison functions when the two objects
	// require no further comparisons.
	Done Action = iota

	// Fallback is returned by custom comparison functions when the comparison of
	// the two objects should continue with the fallback comparison method.
	Fallback
)

type customKey struct {
	reference reflect.Type
	value     reflect.Type
}

var (
	globalCustom    = &Custom{}
	comparatorType  = reflect.TypeOf(Comparator{})
	actionType      = reflect.TypeOf(Done)
	protoType       = reflect.TypeOf((*proto.Message)(nil)).Elem()
	protoComparator = reflect.ValueOf(compareProtos)
)

// Custom is a collection of custom comparators that will be used instead of
// the default comparison methods when comparing objects of the registered types.
type Custom struct {
	funcs map[customKey]reflect.Value
}

// Register assigns the function f with signature func(comparator, T, T) to
// be used as the comparator for instances of type T when using
// Custom.Compare(). f may return nothing or a CompareAction.
// Register will panic if f does not match the expected signature, or if a
// comparator for type T has already been registered with this Custom.
func (c *Custom) Register(f interface{}) {
	v := reflect.ValueOf(f)
	t := v.Type()
	if t.Kind() != reflect.Func {
		panic(fmt.Sprintf("Invalid function %v", t))
	}
	if t.NumIn() != 3 {
		panic(fmt.Sprintf("Compare functions must have 3 args, got %v", t))
	}
	if t.In(0) != comparatorType {
		panic(fmt.Sprintf("First argument must be compare.Comparator, got %v", t.In(0)))
	}
	if !(t.NumOut() == 0 || (t.NumOut() == 1 && t.Out(0) == actionType)) {
		panic(fmt.Sprintf("Compare functions must either have no return values or a single Action"))
	}
	key := customKey{t.In(1), t.In(2)}
	if key.reference != key.value {
		panic(fmt.Sprintf("Comparison arguments must be of the same type, got %v and %v", key.reference, key.value))
	}
	if c.funcs == nil {
		c.funcs = map[customKey]reflect.Value{}
	} else if _, found := c.funcs[key]; found {
		panic(fmt.Sprintf("%v to %v already registered", key.reference, key.value))
	}
	c.funcs[key] = v
}

// Compare delivers all the differences it finds to the specified Handler.
// Compare uses the list of custom comparison handlers registered with
// Custom.Register(), falling back to the default comparison method for the type
// when no custom comparison function has been registered with this custom.
// If the reference and value are equal, the handler will never be invoked.
func (c *Custom) Compare(reference, value interface{}, handler Handler) {
	compare(reference, value, handler, c)
}

// DeepEqual compares a value against a reference using the custom comparators
// and returns true if they are equal.
func (c *Custom) DeepEqual(reference, value interface{}) bool {
	var d test
	c.Compare(reference, value, d.set)
	return !bool(d)
}

// Diff returns the differences between the reference and the value.
// Diff uses the list of custom comparison handlers registered with
// Custom.Register(), falling back to the default comparison method for the type
// when no custom comparison function has been registered with this custom.
// The maximum number of differences is controlled by limit, which must be >0.
// If they compare equal, the length of the returned slice will be 0.
func (c *Custom) Diff(reference, value interface{}, limit int) []Path {
	diffs := make(collect, 0, limit)
	c.Compare(reference, value, diffs.add)
	return ([]Path)(diffs)
}

func (c *Custom) call(key customKey, args []reflect.Value) Action {
	if c == nil {
		return Fallback
	}

	comparator, found := c.funcs[key]
	if !found {
		if !key.reference.Implements(protoType) {
			return c.fallback().call(key, args)
		}
		comparator = protoComparator
	}

	action := Done
	if res := comparator.Call(args); len(res) > 0 {
		action = res[0].Interface().(Action)
	}

	switch action {
	case Done:
		return Done
	case Fallback:
		return c.fallback().call(key, args)
	default:
		panic(fmt.Errorf("Unknown action %v", action))
	}
}

func (c *Custom) fallback() *Custom {
	if c == globalCustom {
		return nil
	}
	return globalCustom
}

func compareProtos(c Comparator, reference proto.Message, value interface{}) {
	if v, ok := value.(proto.Message); !ok || !proto.Equal(reference, v) {
		c.AddDiff(reference, value)
	}
}
