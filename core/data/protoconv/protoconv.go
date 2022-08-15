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
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/generic"
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
func (ctx *ToProtoContext) GetReferenceID(value interface{}) (id int64, isNew bool) {
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
// getValue is callback function which be called if we see the ID for first time
// (returned as value+constructor, since that is needed to support cycles).
func (ctx *FromProtoContext) GetReferencedObject(id int64, nilValue interface{}, getValue func() (newValue interface{}, initValue func())) interface{} {
	if id == 0 {
		return nilValue
	}
	if ctx.refs == nil {
		ctx.refs = map[int64]interface{}{}
	}
	if value, ok := ctx.refs[id]; ok {
		return value
	}
	value, initValue := getValue()
	ctx.refs[id] = value
	initValue()
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
//
//	func(context.Context, O) (P, error)
//
// toObject must be a function with the signature:
//
//	func(context.Context, P) (O, error)
//
// Where P is the proto message type and O is the object type.
func Register(toProto, toObject interface{}) {
	type (
		O = generic.T1
		P = generic.T2
	)
	var (
		OTy = generic.T1Ty
		PTy = generic.T2Ty
	)

	sigs := []generic.Sig{
		generic.Sig{Name: "toProto", Interface: func(context.Context, O) (P, error) { return P{}, nil }, Function: toProto},
		generic.Sig{Name: "toObject", Interface: func(context.Context, P) (O, error) { return O{}, nil }, Function: toObject},
	}
	m := generic.CheckSigs(sigs...)
	if !m.Ok() {
		panic(m.Errors)
	}

	objTy, protoTy := m.Bindings[OTy], m.Bindings[PTy]
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
