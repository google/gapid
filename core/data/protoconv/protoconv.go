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

// Package protoconv provides a mechanism to register functions to convert
// objects to and from proto messages.
package protoconv

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

var (
	mutex         sync.Mutex
	protoToObject = map[reflect.Type]reflect.Value{}
	objectToProto = map[reflect.Type]reflect.Value{}
	tyContext     = reflect.TypeOf((*context.Context)(nil)).Elem()
	tyError       = reflect.TypeOf((*error)(nil)).Elem()
	tyMessage     = reflect.TypeOf((*proto.Message)(nil)).Elem()
)

// ToProtoContext stores internal state for conversion to proto
type ToProtoContext struct {
	// Map needed to identify multiple references to that same object/map.
	refs map[interface{}]int64
}

// GetReferenceID returns unique identifier for the given object.
// nilValue is the default-initialized instance which will be mapped to ID 0.
func (ctx *ToProtoContext) GetReferenceID(value interface{}, nilValue interface{}) (id int64, isNew bool) {
	if value == nilValue {
		return 0, false
	}
	if ctx.refs == nil {
		ctx.refs = map[interface{}]int64{}
	}
	if id, ok := ctx.refs[value]; ok {
		return id, false
	}
	id = int64(len(ctx.refs) + 1)
	ctx.refs[value] = id
	return id, true
}

// FromProtoContext stores internal state for conversion from proto
type FromProtoContext struct {
	// Map needed to resolve objects which are referenced multiple times.
	refs map[int64]interface{}
}

// GetReferencedObject returns referenceable object with the given ID.
// nilValue is the default-initialized instance to return for ID 0.
// getValue is callback function which be called if we see the ID for first time.
func (ctx *FromProtoContext) GetReferencedObject(id int64, nilValue interface{}, getValue func() interface{}) interface{} {
	if id == 0 {
		return nilValue
	}
	if ctx.refs == nil {
		ctx.refs = map[int64]interface{}{}
	}
	if value, ok := ctx.refs[id]; ok {
		return value
	}
	value := getValue()
	ctx.refs[id] = value
	return value
}

// ErrNoConverterRegistered is the error returned from ToProto or ToObject when
// the object's type is not registered for conversion.
type ErrNoConverterRegistered struct {
	Object interface{}
}

func (e ErrNoConverterRegistered) Error() string {
	return fmt.Sprintf("No converter registered for type %T", e.Object)
}

// Register registers the converters toProto and toObject.
// toProto must be a function with the signature:
//   func(context.Context, O) (P, error)
// toObject must be a function with the signature:
//   func(context.Context, P) (O, error)
// Where P is the proto message type and O is the object type.
func Register(toProto, toObject interface{}) {
	toP, toO := reflect.TypeOf(toProto), reflect.TypeOf(toObject)
	if toP.Kind() != reflect.Func {
		panic("toProto must be a function")
	}
	if toO.Kind() != reflect.Func {
		panic("toObject must be a function")
	}
	if err := checkToFunc(toP); err != nil {
		panic(fmt.Errorf(`toProto does not match signature:
  func(context.Context, O) (P, error)
Got:   %v
Error: %v`, printFunc(toP), err))
	}
	if err := checkToFunc(toO); err != nil {
		panic(fmt.Errorf(`toObject does not match signature:
  func(context.Context, P) (O, error)
Got:   %v
Error: %v`, printFunc(toO), err))
	}
	if toP.In(1) != toO.Out(0) {
		panic(fmt.Errorf(`Object type is different between toProto and toObject
toProto:  %v
toObject: %v`, printFunc(toP), printFunc(toO)))
	}
	if toO.In(1) != toP.Out(0) {
		panic(fmt.Errorf(`Proto type is different between toProto and toObject
toProto:  %v
toObject: %v`, printFunc(toP), printFunc(toO)))
	}
	objTy, protoTy := toP.In(1), toO.In(1)
	if !protoTy.Implements(tyMessage) {
		panic(fmt.Errorf("Proto type %v does not implement proto.Message", protoTy))
	}
	mutex.Lock()
	defer mutex.Unlock()
	protoToObject[protoTy] = reflect.ValueOf(toObject)
	objectToProto[objTy] = reflect.ValueOf(toProto)
}

// ToProto converts obj to a proto message using the converter registered with
// Register.
func ToProto(ctx context.Context, obj interface{}) (proto.Message, error) {
	mutex.Lock()
	f, ok := objectToProto[reflect.TypeOf(obj)]
	mutex.Unlock()
	if !ok {
		return nil, ErrNoConverterRegistered{obj}
	}
	out := f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(obj)})
	if err := out[1].Interface(); err != nil {
		return nil, log.Errf(ctx, err.(error), "Failed to convert from %T to proto %v",
			obj, f.Type().Out(0))
	}
	return out[0].Interface().(proto.Message), nil
}

// ToObject converts obj to a proto message using the converter registered with
// Register.
func ToObject(ctx context.Context, msg proto.Message) (interface{}, error) {
	mutex.Lock()
	f, ok := protoToObject[reflect.TypeOf(msg)]
	mutex.Unlock()
	if !ok {
		return nil, ErrNoConverterRegistered{msg}
	}
	out := f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(msg)})
	if err := out[1].Interface(); err != nil {
		return nil, log.Errf(ctx, err.(error), "Failed to convert proto from %T to %v",
			msg, f.Type().Out(0))
	}
	return out[0].Interface(), nil
}

func printFunc(f reflect.Type) string {
	if f.Kind() != reflect.Func {
		return "Not a function"
	}
	b := bytes.Buffer{}
	b.WriteString("func(")
	for i := 0; i < f.NumIn(); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.In(i).String())
	}
	b.WriteString(")")
	switch f.NumOut() {
	case 0:
	case 1:
		b.WriteString(" ")
		b.WriteString(f.Out(0).String())
	default:
		b.WriteString(" (")
		for i := 0; i < f.NumOut(); i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f.Out(i).String())
		}
		b.WriteString(")")
	}
	return b.String()
}

func checkToFunc(got reflect.Type) error {
	if got.Kind() != reflect.Func {
		return fault.Const("Not a function")
	}
	if got.NumIn() != 2 {
		return fault.Const("Incorrect number of parameters")
	}
	if got.NumOut() != 2 {
		return fault.Const("Incorrect number of results")
	}
	if got.In(0) != tyContext {
		return fault.Const("Parameter 0 is not context.Context")
	}
	if got.Out(1) != tyError {
		return fault.Const("Output 1 is not error")
	}
	return nil
}
