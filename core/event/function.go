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

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	boolType    = reflect.TypeOf(true)
)

// checkSignature is an internal function for verifying a function signature matches the expected one.
// It is special in that it allows functions parameterised by one of the types, either input or output.
// You indicate the parameterised entry by having a nil for that type record, and the matching type is
// returned from the fuction.
// It is an error to try to parameterise on more than one, behaviour is undefined in that case.
func checkSignature(ctx context.Context, f reflect.Type, in []reflect.Type, out []reflect.Type) (reflect.Type, error) {
	if !(f.Kind() == reflect.Func) {
		return nil, log.Errf(ctx, nil, "Expected a function. Got: %v", f)
	}
	if f.NumIn() != len(in) {
		return nil, log.Errf(ctx, nil, "Invalid argument count: %d", f.NumIn())
	}
	if f.NumOut() != len(out) {
		return nil, log.Errf(ctx, nil, "Invalid return count: %v", f.NumOut())
	}
	var res reflect.Type
	for i, t := range in {
		check := f.In(i)
		if t == nil {
			res = check
		} else if check != t {
			return nil, log.Errf(ctx, nil, "Incorrect parameter type: %v", check)
		}
	}
	for i, t := range out {
		check := f.Out(i)
		if t == nil {
			res = check
		} else if check != t {
			return nil, log.Errf(ctx, nil, "Incorrect return type: %v", check)
		}
	}
	return res, nil
}

func safeValue(v interface{}, t reflect.Type) reflect.Value {
	if v == nil {
		return reflect.Zero(t)
	}
	return reflect.ValueOf(v)
}

func safeObject(v reflect.Value) interface{} {
	if v.IsNil() {
		return nil
	}
	return v.Interface()
}

func funcToHandler(ctx context.Context, f reflect.Value) Handler {
	t, err := checkSignature(ctx, f.Type(),
		[]reflect.Type{contextType, nil},
		[]reflect.Type{errorType},
	)
	if err != nil {
		log.F(ctx, true, "Checking handler signature. Error: %v", err)
	}
	return func(ctx context.Context, event interface{}) error {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			safeValue(event, t),
		}
		if !args[1].Type().AssignableTo(t) {
			return log.Errf(ctx, nil, "Invalid event type: %v. Expected: %v", args[1].Type(), t)
		}
		result := f.Call(args)
		if !result[0].IsNil() {
			return fault.From(result[0].Interface())
		}
		return nil
	}
}

func funcToProducer(ctx context.Context, f reflect.Value) Producer {
	_, err := checkSignature(ctx, f.Type(),
		[]reflect.Type{contextType},
		[]reflect.Type{nil},
	)
	if err != nil {
		log.F(ctx, true, "Checking producer signature. Error: %v", err)
	}
	return func(ctx context.Context) interface{} {
		args := []reflect.Value{reflect.ValueOf(ctx)}
		return safeObject(f.Call(args)[0])
	}
}

func funcToPredicate(ctx context.Context, f reflect.Value) Predicate {
	t, err := checkSignature(ctx, f.Type(),
		[]reflect.Type{contextType, nil},
		[]reflect.Type{boolType},
	)
	if err != nil {
		log.F(ctx, true, "Checking handler signature. Error: %v", err)
	}
	return func(ctx context.Context, event interface{}) bool {
		args := []reflect.Value{reflect.ValueOf(ctx), safeValue(event, t)}
		if args[1].Type() != t {
			return false
		}
		result := f.Call(args)
		return result[0].Bool()
	}
}
