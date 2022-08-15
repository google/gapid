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

package event

import (
	"context"
	"reflect"

	"github.com/google/gapid/core/log"
)

// AsHandler wraps a destination into an event handler.
// The destination can be one of
//
//	func(context.Context, T, error) error
//	chan T
//	chan<- T
//
// If it is not, this function will panic, as this is assumed to be a programming error.
// If the handler is invoked with an event that is not of type T the handler will return an error.
func AsHandler(ctx context.Context, f interface{}) Handler {
	switch dst := f.(type) {
	case Handler:
		return dst
	case chan interface{}:
		return chanHandler(dst, nil)
	case chan<- interface{}:
		return chanHandler(dst, nil)
	}
	v := reflect.ValueOf(f)
	switch v.Kind() {
	//TODO: allow array types for sinks?
	//TODO: any other magical translations needed?
	case reflect.Func:
		return funcToHandler(ctx, v)
	case reflect.Chan:
		return chanToHandler(ctx, v)
	}
	log.F(ctx, true, "Expected an event handler. Type: %v", v.Type())
	return nil
}

// AsProducer wraps an event generator into an event producer.
// The generator can be one of
//
//	func(context.Context) T
//	Source
//	chan T
//	<-chan T
//	[]T
//	[n]T
//
// If it is not, this function will panic, as this is assumed to be a programming error.
func AsProducer(ctx context.Context, f interface{}) Producer {
	switch src := f.(type) {
	case Source:
		return src.Next
	case Producer:
		return src
	case chan interface{}:
		return chanProducer(src)
	case <-chan interface{}:
		return chanProducer(src)
	}
	v := reflect.ValueOf(f)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		length := v.Len()
		i := 0
		return func(ctx context.Context) interface{} {
			if i >= length {
				return nil
			}
			res := v.Index(i).Interface()
			i++
			return res
		}
	case reflect.Func:
		return funcToProducer(ctx, v)
	case reflect.Chan:
		return chanToProducer(ctx, v)
	}
	log.F(ctx, true, "Expected an event source. Type: %v", v.Type())
	return nil
}

// AsPredicate wraps a function into an event predicate.
// The function must be a function of the form
//
//	func(context.Context, T) bool
//
// If it is not, this function will panic, as this is assumed to be a programming error.
// If the handler is invoked with an event that is not of type T the handler will return an error.
func AsPredicate(ctx context.Context, f interface{}) Predicate {
	switch pred := f.(type) {
	case Predicate:
		return pred
	}
	v := reflect.ValueOf(f)
	switch v.Kind() {
	case reflect.Func:
		return funcToPredicate(ctx, v)
	}
	log.F(ctx, true, "Expected an event source. Type: %v", v.Type())
	return nil
}
