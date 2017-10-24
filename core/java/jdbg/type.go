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

package jdbg

import (
	"fmt"

	"github.com/google/gapid/core/java/jdwp"
)

// Type represents a Java type.
type Type interface {
	// String returns the string representation of the type.
	String() string
	// Signature returns the Java signature for the type.
	Signature() string
	// CastableTo returns true if this type can be cast to ty.
	CastableTo(ty Type) bool
	// Call invokes the static method with the specified arguments.
	Call(method string, args ...interface{}) Value

	call(object Value, method string, args []interface{}) Value
	field(object Value, name string) Value
	jdbg() *JDbg
}

// Simple is a primitive type.
type Simple struct {
	j  *JDbg
	ty jdwp.Tag
}

func (t *Simple) String() string {
	switch t.ty {
	case jdwp.TagVoid:
		return "void"
	case jdwp.TagBoolean:
		return "boolean"
	case jdwp.TagByte:
		return "byte"
	case jdwp.TagChar:
		return "char"
	case jdwp.TagShort:
		return "short"
	case jdwp.TagInt:
		return "int"
	case jdwp.TagLong:
		return "long"
	case jdwp.TagFloat:
		return "float"
	case jdwp.TagDouble:
		return "double"
	default:
		return t.ty.String()
	}
}

// Signature returns the Java signature for the type.
func (t *Simple) Signature() string {
	switch t.ty {
	case jdwp.TagVoid:
		return "V"
	case jdwp.TagBoolean:
		return "Z"
	case jdwp.TagByte:
		return "B"
	case jdwp.TagChar:
		return "C"
	case jdwp.TagShort:
		return "S"
	case jdwp.TagInt:
		return "I"
	case jdwp.TagLong:
		return "J"
	case jdwp.TagFloat:
		return "F"
	case jdwp.TagDouble:
		return "D"
	default:
		return t.ty.String()
	}
}

// CastableTo returns true if this type can be cast to ty.
func (t *Simple) CastableTo(ty Type) bool {
	return t == ty || ty == t.j.ObjectType()
}

// Call invokes the static method with the specified arguments.
func (t *Simple) Call(method string, args ...interface{}) Value {
	return t.call(Value{}, method, args)
}

func (t *Simple) call(object Value, method string, args []interface{}) Value {
	t.j.fail("Type '%v' does not support methods", t.ty)
	return Value{}
}

func (t *Simple) field(object Value, name string) Value {
	t.j.fail("Type '%v' does not support fields", t.ty)
	return Value{}
}

func (t *Simple) jdbg() *JDbg { return t.j }

// Array is the type of an array.
type Array struct {
	*Class
	el Type
}

func (t *Array) String() string { return fmt.Sprintf("%v[]", t.el) }

// CastableTo returns true if this type can be cast to ty.
func (t *Array) CastableTo(ty Type) bool {
	if t == ty {
		return true
	}
	return t.Class.CastableTo(ty)
}

// New constructs a new array of the specified size.
func (t *Array) New(size int) Value {
	array, err := t.j.conn.NewArray(jdwp.ArrayTypeID(t.class.TypeID), size)
	if err != nil {
		t.j.fail("Failed to create array: %v", err)
	}
	if array.Type != jdwp.TagArray {
		t.j.fail("NewArray returned %v, not array", array.Type)
	}
	return newValue(t, jdwp.ArrayID(array.Object))
}

type classResolvedInfo struct {
	methods    methods
	fields     jdwp.Fields
	interfaces []*Class
	allMethods methods // Include super's allMethods
	error      error
}

// Class is the type of an object.
type Class struct {
	j          *JDbg
	signature  string
	name       string
	class      jdwp.ClassInfo
	implements []*Class
	fields     jdwp.Fields
	super      *Class
	resolved   *classResolvedInfo
}

func (t *Class) String() string { return t.name }

// ID returns the JDWP class identifier.
func (t *Class) ID() jdwp.ClassID { return t.class.ClassID() }

// Signature returns the Java signature for the type.
func (t *Class) Signature() string { return t.signature }

// New returns a new instance of the class type using the specified parameters.
func (t *Class) New(args ...interface{}) Value {
	m := t.j.resolveMethod(false, t, constructor, args)
	values := t.j.marshalN(args)
	res, err := t.j.conn.NewInstance(
		m.class.class.ClassID(), m.id, t.j.thread, jdwp.InvokeSingleThreaded, values...)
	if err != nil {
		t.j.fail("NewInstance() returned: %v", err)
	}
	t.j.errFromException(res.Exception, m)
	return newValue(t, res.Result)
}

// CastableTo returns true if this type can be cast to ty.
func (t *Class) CastableTo(ty Type) bool {
	if t == ty {
		return true
	}
	for _, i := range t.implements {
		if i.CastableTo(ty) {
			return true
		}
	}
	if t.super != nil {
		return t.super.CastableTo(ty)
	}
	return false
}

// Call invokes a static method on the class.
func (t *Class) Call(method string, args ...interface{}) Value {
	return t.call(Value{}, method, args)
}

// Field returns the value of the static field with the given name.
func (t *Class) Field(name string) Value {
	field := t.resolve().fields.FindByName(name)
	values, err := t.j.conn.GetStaticFieldValues(t.class.TypeID, field.ID)
	if err != nil {
		t.j.fail("GetValues() returned: %v", err)
	}
	return t.j.value(values[0])
}

// Super returns the super class type.
func (t *Class) Super() *Class {
	return t.super
}

func (t *Class) call(object Value, method string, args []interface{}) Value {
	m := t.j.resolveMethod(object != nilValue, t, method, args)
	values := t.j.marshalN(args)

	var res jdwp.InvokeResult
	var err error
	if m.mod&jdwp.ModStatic != 0 {
		res, err = t.j.conn.InvokeStaticMethod(
			t.class.ClassID(), m.id, t.j.thread, jdwp.InvokeSingleThreaded, values...)
	} else {
		if object == nilValue {
			t.j.fail("Cannot call non-static method '%v' without an object", method)
		}
		var obj interface{}
		obj = object.val
		object, ok := obj.(jdwp.Object)
		if !ok {
			t.j.fail("Cannot call methods on %T types", obj)
		}
		res, err = t.j.conn.InvokeMethod(
			object.ID(), t.class.ClassID(), m.id, t.j.thread, jdwp.InvokeSingleThreaded, values...)
	}
	if err != nil {
		t.j.err(err)
	}

	t.j.errFromException(res.Exception, m)

	result, isResultObject := res.Result.(jdwp.Object)
	if !isResultObject {
		if _, expectedClass := m.sig.Return.(*Class); expectedClass {
			panic(fmt.Errorf("Call of %v returned value %T(%v) when %v was expected",
				method, res.Result, res.Result, m.sig.Return))
		}
		return newValue(m.sig.Return, res.Result)
	}

	if result.ID() == 0 {
		return Value{m.sig.Return, result.ID()} // null pointer
	}

	tyID, err := t.j.conn.GetObjectType(result.ID())
	if err != nil {
		t.j.fail("GetObjectType() returned: %v", err)
	}

	ty := t.j.typeFromID(tyID.Type)
	if !ty.CastableTo(m.sig.Return) {
		t.j.fail("Call of %v returned type %v which is not castable to %v",
			method, ty, m.sig.Return)
	}

	return newValue(ty, res.Result)
}

func (t *Class) field(object Value, name string) Value {
	if object == nilValue {
		t.j.fail("Cannot get field '%v' on nill object", name)
	}
	f := t.fields.FindByName(name)
	if f == nil {
		t.j.fail("Class '%v' does not contain field '%v'", t.name, name)
	}
	obj, ok := object.val.(jdwp.Object)
	if !ok {
		t.j.fail("Class '%v' does not support fields", t.name)
	}
	vals, err := t.j.conn.GetFieldValues(obj.ID(), f.ID)
	if err != nil {
		t.j.fail("GetFieldValues() returned: %v", err)
	}
	if len(vals) != 1 {
		t.j.fail("GetFieldValues() returned %n values, expected 1", len(vals))
	}
	return t.j.value(vals[0])
}

func (t *Class) jdbg() *JDbg { return t.j }

func (t *Class) resolve() *classResolvedInfo {
	if t.resolved != nil {
		return t.resolved
	}
	t.resolved = &classResolvedInfo{}

	f, err := t.j.conn.GetFields(t.class.TypeID)
	if err != nil {
		t.resolved.error = err
		return t.resolved
	}
	t.resolved.fields = f

	m, err := t.j.conn.GetMethods(t.class.TypeID)
	if err != nil {
		t.resolved.error = err
		return t.resolved
	}
	t.resolved.methods = make([]method, 0, len(m))
	for _, m := range m {
		sig, err := t.j.parseMethodSignature(m.Signature)
		if err != nil {
			// Probably uses a type that hasn't been loaded. Ignore.
			continue
		}
		t.resolved.methods = append(t.resolved.methods, method{
			id:    m.ID,
			mod:   m.ModBits,
			name:  m.Name,
			sig:   sig,
			class: t,
		})
	}
	// build allMethods from methods and super's allMethods
	var superMethods methods
	if t.super != nil {
		if resolved := t.super.resolve(); resolved.error == nil {
			superMethods = resolved.allMethods
		}
	}
	all := make(methods, 0, len(t.resolved.methods)+len(superMethods))
	t.resolved.allMethods = append(append(all, t.resolved.methods...), superMethods...)

	return t.resolved
}

func (j *JDbg) errFromException(exception jdwp.TaggedObjectID, method method) {
	if (exception.Type == 0 || exception.Type == jdwp.TagObject) && exception.Object == 0 {
		return
	}
	str := j.object(exception.Object).Call("toString").Get()
	j.fail("Exception raised calling: %v\n%v", method, str)
}
