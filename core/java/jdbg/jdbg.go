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

// Package jdbg provides a simpler interface to the jdwp package, offering
// simple type resolving and method invoking functions.
package jdbg

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/java/jdwp"
)

type cache struct {
	arrays  map[string]*Array
	classes map[string]*Class
	idToSig map[jdwp.ReferenceTypeID]string

	objTy       *Class
	stringTy    *Class
	numberTy    *Class
	boolObjTy   *Class
	byteObjTy   *Class
	charObjTy   *Class
	shortObjTy  *Class
	intObjTy    *Class
	longObjTy   *Class
	floatObjTy  *Class
	doubleObjTy *Class
	boolTy      *Simple
	byteTy      *Simple
	charTy      *Simple
	shortTy     *Simple
	intTy       *Simple
	longTy      *Simple
	floatTy     *Simple
	doubleTy    *Simple

	voidTy *Simple
}

// JDbg is a wrapper around a JDWP connection that provides an easier interface
// for usage.
type JDbg struct {
	conn    *jdwp.Connection
	thread  jdwp.ThreadID
	cache   cache
	objects []jdwp.ObjectID // Objects created that have GC disabled
}

// Do calls f with a JDbg instance, returning the error returned by f.
// If any JDWP errors are raised during the call to f, then execution of f is
// immediately terminated, and the JDWP error is returned.
func Do(conn *jdwp.Connection, thread jdwp.ThreadID, f func(jdbg *JDbg) error) error {
	j := &JDbg{
		conn:   conn,
		thread: thread,
		cache: cache{
			arrays:  map[string]*Array{},
			classes: map[string]*Class{},
			idToSig: map[jdwp.ReferenceTypeID]string{},
		},
	}
	defer func() {
		// Reenable GC for all objects used during the call to f()
		for _, o := range j.objects {
			conn.EnableGC(o)
		}
	}()

	return Try(func() error {
		// Prime the cache with basic types.
		j.cache.voidTy = &Simple{j: j, ty: jdwp.TagVoid}
		j.cache.boolTy = &Simple{j: j, ty: jdwp.TagBoolean}
		j.cache.byteTy = &Simple{j: j, ty: jdwp.TagByte}
		j.cache.charTy = &Simple{j: j, ty: jdwp.TagChar}
		j.cache.shortTy = &Simple{j: j, ty: jdwp.TagShort}
		j.cache.intTy = &Simple{j: j, ty: jdwp.TagInt}
		j.cache.longTy = &Simple{j: j, ty: jdwp.TagLong}
		j.cache.floatTy = &Simple{j: j, ty: jdwp.TagFloat}
		j.cache.doubleTy = &Simple{j: j, ty: jdwp.TagDouble}
		j.cache.objTy = j.Class("java.lang.Object")
		j.cache.stringTy = j.Class("java.lang.String")
		j.cache.numberTy = j.Class("java.lang.Number")
		j.cache.boolObjTy = j.Class("java.lang.Boolean")
		j.cache.byteObjTy = j.Class("java.lang.Byte")
		j.cache.charObjTy = j.Class("java.lang.Character")
		j.cache.shortObjTy = j.Class("java.lang.Short")
		j.cache.intObjTy = j.Class("java.lang.Integer")
		j.cache.longObjTy = j.Class("java.lang.Long")
		j.cache.floatObjTy = j.Class("java.lang.Float")
		j.cache.doubleObjTy = j.Class("java.lang.Double")

		// Call f
		return f(j)
	})
}

// Connection returns the JDWP connection.
func (j *JDbg) Connection() *jdwp.Connection { return j.conn }

// ObjectType returns the Java java.lang.Object type.
func (j *JDbg) ObjectType() *Class { return j.cache.objTy }

// StringType returns the Java java.lang.String type.
func (j *JDbg) StringType() *Class { return j.cache.stringTy }

// NumberType returns the Java java.lang.Number type.
func (j *JDbg) NumberType() *Class { return j.cache.numberTy }

// BoolObjectType returns the Java java.lang.Boolean type.
func (j *JDbg) BoolObjectType() *Class { return j.cache.boolObjTy }

// ByteObjectType returns the Java java.lang.Byte type.
func (j *JDbg) ByteObjectType() *Class { return j.cache.byteObjTy }

// CharObjectType returns the Java java.lang.Character type.
func (j *JDbg) CharObjectType() *Class { return j.cache.charObjTy }

// ShortObjectType returns the Java java.lang.Short type.
func (j *JDbg) ShortObjectType() *Class { return j.cache.shortObjTy }

// IntObjectType returns the Java java.lang.Integer type.
func (j *JDbg) IntObjectType() *Class { return j.cache.intObjTy }

// LongObjectType returns the Java java.lang.Long type.
func (j *JDbg) LongObjectType() *Class { return j.cache.longObjTy }

// FloatObjectType returns the Java java.lang.Float type.
func (j *JDbg) FloatObjectType() *Class { return j.cache.floatObjTy }

// DoubleObjectType returns the Java java.lang.Double type.
func (j *JDbg) DoubleObjectType() *Class { return j.cache.doubleObjTy }

// BoolType returns the Java java.lang.Boolean type.
func (j *JDbg) BoolType() *Simple { return j.cache.boolTy }

// ByteType returns the Java byte type.
func (j *JDbg) ByteType() *Simple { return j.cache.byteTy }

// CharType returns the Java char type.
func (j *JDbg) CharType() *Simple { return j.cache.charTy }

// ShortType returns the Java short type.
func (j *JDbg) ShortType() *Simple { return j.cache.shortTy }

// IntType returns the Java int type.
func (j *JDbg) IntType() *Simple { return j.cache.intTy }

// LongType returns the Java long type.
func (j *JDbg) LongType() *Simple { return j.cache.longTy }

// FloatType returns the Java float type.
func (j *JDbg) FloatType() *Simple { return j.cache.floatTy }

// DoubleType returns the Java double type.
func (j *JDbg) DoubleType() *Simple { return j.cache.doubleTy }

// Type looks up the specified type by signature.
// For example: "Ljava/io/File;"
func (j *JDbg) Type(sig string) Type {
	offset := 0
	ty, err := j.parseSignature(sig, &offset)
	if err != nil {
		j.fail("Failed to parse signature: %v", err)
	}
	return ty
}

// Class looks up the specified class by name.
// For example: "java.io.File"
func (j *JDbg) Class(name string) *Class {
	ty := j.Type(fmt.Sprintf("L%s;", strings.Replace(name, ".", "/", -1)))
	if class, ok := ty.(*Class); ok {
		return class
	}
	j.fail("Resolved type was not array but %T", ty)
	return nil
}

// AllClasses returns all the loaded classes.
func (j *JDbg) AllClasses() []*Class {
	classes, err := j.conn.GetAllClasses()
	if err != nil {
		j.fail("Couldn't get all classes: %v", err)
	}
	out := []*Class{}
	for _, class := range classes {
		c, err := j.class(class)
		if err != nil {
			j.fail("Couldn't get class '%v': %v", class.Signature, err)
		}
		out = append(out, c)
	}
	return out
}

// ArrayOf returns the type of the array with specified element type.
func (j *JDbg) ArrayOf(elTy Type) *Array {
	ty := j.Type("[" + elTy.Signature())
	if array, ok := ty.(*Array); ok {
		return array
	}
	j.fail("Resolved type was not array but %T", ty)
	return nil
}

// classFromSig looks up the specified class type by signature.
func (j *JDbg) classFromSig(sig string) (*Class, error) {
	if class, ok := j.cache.classes[sig]; ok {
		return class, nil
	}
	class, err := j.conn.GetClassBySignature(sig)
	if err != nil {
		return nil, err
	}
	return j.class(class)
}

func (j *JDbg) class(class jdwp.ClassInfo) (*Class, error) {
	sig := class.Signature
	if cached, ok := j.cache.classes[class.Signature]; ok {
		return cached, nil
	}

	name := strings.Replace(strings.TrimRight(strings.TrimLeft(sig, "[L"), ";"), "/", ".", -1)

	ty := &Class{j: j, signature: sig, name: name, class: class}
	j.cache.classes[sig] = ty
	j.cache.idToSig[class.TypeID] = sig

	superid, err := j.conn.GetSuperClass(class.ClassID())
	if err != nil {
		return nil, err
	}

	if superid != 0 {
		ty.super = j.typeFromID(jdwp.ReferenceTypeID(superid)).(*Class)
	}

	implementsids, err := j.conn.GetImplemented(class.TypeID)
	if err != nil {
		return nil, err
	}

	ty.implements = make([]*Class, len(implementsids))
	for i, id := range implementsids {
		ty.implements[i] = j.typeFromID(jdwp.ReferenceTypeID(id)).(*Class)
	}

	ty.fields, err = j.conn.GetFields(class.TypeID)
	if err != nil {
		return nil, err
	}

	return ty, nil
}

func (j *JDbg) typeFromID(id jdwp.ReferenceTypeID) Type {
	sig, ok := j.cache.idToSig[id]
	if !ok {
		var err error
		sig, err = j.conn.GetTypeSignature(id)
		if err != nil {
			j.fail("GetTypeSignature() returned: %v", err)
		}
		j.cache.idToSig[id] = sig
	}
	return j.Type(sig)
}

// This returns the this object for the current stack frame.
func (j *JDbg) This() Value {
	frames, err := j.conn.GetFrames(j.thread, 0, 1)
	if err != nil {
		j.fail("GetFrames() returned: %v", err)
	}

	this, err := j.conn.GetThisObject(j.thread, frames[0].Frame)
	if err != nil {
		j.fail("GetThisObject() returned: %v", err)
	}

	return j.object(this.Object)
}

func (j *JDbg) String(val string) Value {
	str, err := j.conn.CreateString(val)
	if err != nil {
		j.fail("CreateString() returned: %v", err)
	}
	return j.object(str)
}

// findVar finds the variable with the given name in the given frame
func (j *JDbg) findVar(name string, frame jdwp.FrameInfo) jdwp.VariableRequest {
	table, err := j.conn.VariableTable(
		jdwp.ReferenceTypeID(frame.Location.Class),
		frame.Location.Method)

	if err != nil {
		j.fail("VariableTable returned: %v", err)
	}

	variable := jdwp.VariableRequest{-1, 0}

	for _, slot := range table.Slots {
		if name == slot.Name {
			variable.Index = slot.Slot
			variable.Tag = slot.Signature[0]
		}
	}

	if variable.Index == -1 {
		j.fail("Could not find variable with name %s", name)
	}
	return variable
}

// GetStackObject returns an object by name that exists in the current
// stack-frame.
func (j *JDbg) GetStackObject(name string) Value {
	frames, err := j.conn.GetFrames(j.thread, 0, 1)
	if err != nil {
		j.fail("GetFrames() returned: %v", err)
	}
	variable := j.findVar(name, frames[0])

	values, err := j.conn.GetValues(j.thread, frames[0].Frame, []jdwp.VariableRequest{variable})
	if err != nil {
		j.fail("GetValues() returned: %v", err)
	}
	return j.value(values[0])
}

// SetStackObject sets and object in the current stack-frame to the
// given value.
func (j *JDbg) SetStackObject(name string, val Value) {
	frames, err := j.conn.GetFrames(j.thread, 0, 1)
	if err != nil {
		j.fail("GetFrames() returned: %v", err)
	}

	variable := j.findVar(name, frames[0])
	v := val.val.(jdwp.Value)
	assign := jdwp.VariableAssignmentRequest{variable.Index, v}
	err = j.conn.SetValues(j.thread, frames[0].Frame, []jdwp.VariableAssignmentRequest{assign})
	if err != nil {
		j.fail("GetValues() returned: %v", err)
	}
}

func (j *JDbg) object(id jdwp.Object) Value {
	tyID, err := j.conn.GetObjectType(id.ID())
	if err != nil {
		j.fail("GetObjectType() returned: %v", err)
	}

	ty := j.typeFromID(tyID.Type)
	return newValue(ty, id)
}

func (j *JDbg) value(o interface{}) Value {
	switch v := o.(type) {
	case jdwp.Object:
		return j.object(v)
	default:
		j.fail("Unhandled variable type %T", o)
		return Value{}
	}
}
