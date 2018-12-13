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

package jdwp

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/binary"
)

// debug adds panic handlers to encode() and decode() so that incorrectly
// handled types can be more easily identified.
const debug = false

func unbox(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

// encode writes the value v to w, using the JDWP encoding scheme.
func (c *Connection) encode(w binary.Writer, v reflect.Value) error {
	if debug {
		defer func() {
			if r := recover(); r != nil {
				panic(fmt.Errorf("Type %T %v %v", v.Interface(), v.Type().Name(), v.Kind()))
			}
		}()
	}

	t := v.Type()
	o := v.Interface()

	switch v.Type() {
	case reflect.TypeOf((*EventModifier)(nil)).Elem():
		// EventModifier's are prefixed with their 1-byte modKind.
		w.Uint8(o.(EventModifier).modKind())

	case reflect.TypeOf((*Value)(nil)).Elem():
		// values are prefixed with their 1-tag type.
		switch o.(type) {
		case ArrayID:
			w.Uint8(uint8(TagArray))
		case byte:
			w.Uint8(uint8(TagByte))
		case Char:
			w.Uint8(uint8(TagChar))
		case ObjectID:
			w.Uint8(uint8(TagObject))
		case float32:
			w.Uint8(uint8(TagFloat))
		case float64:
			w.Uint8(uint8(TagDouble))
		case int, int32:
			w.Uint8(uint8(TagInt))
		case int16:
			w.Uint8(uint8(TagShort))
		case int64:
			w.Uint8(uint8(TagLong))
		case nil:
			w.Uint8(uint8(TagVoid))
		case bool:
			w.Uint8(uint8(TagBoolean))
		case StringID:
			w.Uint8(uint8(TagString))
		case ThreadID:
			w.Uint8(uint8(TagThread))
		case ThreadGroupID:
			w.Uint8(uint8(TagThreadGroup))
		case ClassLoaderID:
			w.Uint8(uint8(TagClassLoader))
		case ClassObjectID:
			w.Uint8(uint8(TagClassObject))
		default:
			panic(fmt.Errorf("Got Value of type %T", o))
		}
	}

	switch o := o.(type) {
	case ReferenceTypeID, ClassID, InterfaceID, ArrayTypeID:
		binary.WriteUint(w, c.idSizes.ReferenceTypeIDSize*8, unbox(v).Uint())

	case MethodID:
		binary.WriteUint(w, c.idSizes.MethodIDSize*8, unbox(v).Uint())

	case FieldID:
		binary.WriteUint(w, c.idSizes.FieldIDSize*8, unbox(v).Uint())

	case ObjectID, ThreadID, ThreadGroupID, StringID, ClassLoaderID, ClassObjectID, ArrayID:
		binary.WriteUint(w, c.idSizes.ObjectIDSize*8, unbox(v).Uint())

	case []byte: // Optimisation
		w.Uint32(uint32(len(o)))
		w.Data(o)

	default:
		switch t.Kind() {
		case reflect.Ptr, reflect.Interface:
			return c.encode(w, v.Elem())
		case reflect.String:
			w.Uint32(uint32(v.Len()))
			w.Data([]byte(v.String()))
		case reflect.Uint8:
			w.Uint8(uint8(v.Uint()))
		case reflect.Uint64:
			w.Uint64(uint64(v.Uint()))
		case reflect.Int8:
			w.Int8(int8(v.Int()))
		case reflect.Int16:
			w.Int16(int16(v.Int()))
		case reflect.Int32, reflect.Int:
			w.Int32(int32(v.Int()))
		case reflect.Int64:
			w.Int64(v.Int())
		case reflect.Float32:
			w.Float32(float32(v.Float()))
		case reflect.Float64:
			w.Float64(v.Float())
		case reflect.Bool:
			w.Bool(v.Bool())
		case reflect.Struct:
			for i, count := 0, v.NumField(); i < count; i++ {
				c.encode(w, v.Field(i))
			}
		case reflect.Slice:
			count := v.Len()
			w.Uint32(uint32(count))
			for i := 0; i < count; i++ {
				c.encode(w, v.Index(i))
			}
		default:
			panic(fmt.Errorf("Unhandled type %T %v %v", o, t.Name(), t.Kind()))
		}
	}
	return w.Error()
}

// decode reads the value v from r, using the JDWP encoding scheme.
func (c *Connection) decode(r binary.Reader, v reflect.Value) error {
	if debug {
		defer func() {
			if r := recover(); r != nil {
				panic(fmt.Errorf("Type %T %v %v", v.Interface(), v.Type().Name(), v.Kind()))
			}
		}()
	}

	switch v.Type() {
	case reflect.TypeOf((*Event)(nil)).Elem():
		var kind EventKind
		if err := c.decode(r, reflect.ValueOf(&kind)); err != nil {
			return err
		}
		event := kind.event()
		v.Set(reflect.ValueOf(event))
		v = v.Elem()
		// Continue to decode event body below.

	case reflect.TypeOf((*Value)(nil)).Elem():
		tag := Tag(r.Uint8())
		var ty reflect.Type
		switch tag {
		case TagArray:
			ty = reflect.TypeOf(ArrayID(0))
		case TagByte:
			ty = reflect.TypeOf(byte(0))
		case TagChar:
			ty = reflect.TypeOf(Char(0))
		case TagObject:
			ty = reflect.TypeOf(ObjectID(0))
		case TagFloat:
			ty = reflect.TypeOf(float32(0))
		case TagDouble:
			ty = reflect.TypeOf(float64(0))
		case TagInt:
			ty = reflect.TypeOf(int(0))
		case TagShort:
			ty = reflect.TypeOf(int16(0))
		case TagLong:
			ty = reflect.TypeOf(int64(0))
		case TagBoolean:
			ty = reflect.TypeOf(false)
		case TagString:
			ty = reflect.TypeOf(StringID(0))
		case TagThread:
			ty = reflect.TypeOf(ThreadID(0))
		case TagThreadGroup:
			ty = reflect.TypeOf(ThreadGroupID(0))
		case TagClassLoader:
			ty = reflect.TypeOf(ClassLoaderID(0))
		case TagClassObject:
			ty = reflect.TypeOf(ClassObjectID(0))
		case TagVoid:
			v.Set(reflect.New(v.Type()).Elem())
			return r.Error()
		default:
			panic(fmt.Errorf("Unhandled value type %v", tag))
		}
		data := reflect.New(ty).Elem()
		c.decode(r, data)
		v.Set(data)
		return r.Error()
	}

	t := v.Type()
	o := v.Interface()
	switch o := o.(type) {
	case ReferenceTypeID, ClassID, InterfaceID, ArrayTypeID:
		v.Set(reflect.ValueOf(binary.ReadUint(r, c.idSizes.ReferenceTypeIDSize*8)).Convert(t))

	case MethodID:
		v.Set(reflect.ValueOf(binary.ReadUint(r, c.idSizes.MethodIDSize*8)).Convert(t))

	case FieldID:
		v.Set(reflect.ValueOf(binary.ReadUint(r, c.idSizes.FieldIDSize*8)).Convert(t))

	case ObjectID, ThreadID, ThreadGroupID, StringID, ClassLoaderID, ClassObjectID, ArrayID:
		v.Set(reflect.ValueOf(binary.ReadUint(r, c.idSizes.ObjectIDSize*8)).Convert(t))

	case EventModifier:
		panic("Cannot decode EventModifiers")

	default:
		switch t.Kind() {
		case reflect.Ptr, reflect.Interface:
			return c.decode(r, v.Elem())
		case reflect.String:
			data := make([]byte, r.Uint32())
			r.Data(data)
			v.Set(reflect.ValueOf(string(data)).Convert(t))
		case reflect.Bool:
			v.Set(reflect.ValueOf(r.Bool()).Convert(t))
		case reflect.Uint8:
			v.Set(reflect.ValueOf(r.Uint8()).Convert(t))
		case reflect.Uint64:
			v.Set(reflect.ValueOf(r.Uint64()).Convert(t))
		case reflect.Int8:
			v.Set(reflect.ValueOf(r.Int8()).Convert(t))
		case reflect.Int16:
			v.Set(reflect.ValueOf(r.Int16()).Convert(t))
		case reflect.Int32, reflect.Int:
			v.Set(reflect.ValueOf(r.Int32()).Convert(t))
		case reflect.Int64:
			v.Set(reflect.ValueOf(r.Int64()).Convert(t))
		case reflect.Struct:
			for i, count := 0, v.NumField(); i < count; i++ {
				c.decode(r, v.Field(i))
			}
		case reflect.Slice:
			count := int(r.Uint32())
			slice := reflect.MakeSlice(t, count, count)
			for i := 0; i < count; i++ {
				c.decode(r, slice.Index(i))
			}
			v.Set(slice)
		default:
			panic(fmt.Errorf("Unhandled type %T %v %v", o, t.Name(), t.Kind()))
		}
	}
	return r.Error()
}
