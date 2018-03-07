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

// ObjectType describes a Java type.
type ObjectType struct {
	Kind TypeTag
	Type ReferenceTypeID
}

// GetObjectType returns the type of the specified object.
func (c *Connection) GetObjectType(object ObjectID) (ObjectType, error) {
	var res ObjectType
	err := c.get(cmdObjectReferenceReferenceType, object, &res)
	return res, err
}

// GetFieldValues returns the values of all the instance fields.
func (c *Connection) GetFieldValues(obj ObjectID, fields ...FieldID) ([]Value, error) {
	var res []Value
	err := c.get(cmdObjectReferenceGetValues, struct {
		Obj    ObjectID
		Fields []FieldID
	}{obj, fields}, &res)
	return res, err
}

// InvokeMethod invokes the specified static method.
func (c *Connection) InvokeMethod(object ObjectID, class ClassID, method MethodID, thread ThreadID, options InvokeOptions, args ...Value) (InvokeResult, error) {
	req := struct {
		Object  ObjectID
		Thread  ThreadID
		Class   ClassID
		Method  MethodID
		Args    []Value
		Options InvokeOptions
	}{object, thread, class, method, args, options}
	var res InvokeResult
	err := c.get(cmdObjectReferenceInvokeMethod, req, &res)
	return res, err
}

// DisableGC disables garbage collection for the specified object.
func (c *Connection) DisableGC(object ObjectID) error {
	return c.get(cmdObjectReferenceDisableCollection, object, nil)
}

// EnableGC enables garbage collection for the specified object.
func (c *Connection) EnableGC(object ObjectID) error {
	return c.get(cmdObjectReferenceEnableCollection, object, nil)
}
