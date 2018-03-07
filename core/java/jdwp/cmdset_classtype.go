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

// InvokeResult holds the return values for a method invokation.
type InvokeResult struct {
	Result    Value
	Exception TaggedObjectID
}

// GetSuperClass returns the immediate super class of the specified class.
func (c *Connection) GetSuperClass(class ClassID) (ClassID, error) {
	var res ClassID
	err := c.get(cmdClassTypeSuperclass, class, &res)
	return res, err
}

// InvokeStaticMethod invokes the specified static method.
func (c *Connection) InvokeStaticMethod(class ClassID, method MethodID, thread ThreadID, options InvokeOptions, args ...Value) (InvokeResult, error) {
	req := struct {
		Class   ClassID
		Thread  ThreadID
		Method  MethodID
		Args    []Value
		Options InvokeOptions
	}{class, thread, method, args, options}
	var res InvokeResult
	err := c.get(cmdClassTypeInvokeMethod, req, &res)
	return res, err
}

// NewInstanceResult holds the return values for a constructor invokation.
type NewInstanceResult struct {
	Result    TaggedObjectID
	Exception TaggedObjectID
}

// NewInstance invokes the specified constructor.
func (c *Connection) NewInstance(class ClassID, constructor MethodID, thread ThreadID, options InvokeOptions, args ...Value) (NewInstanceResult, error) {
	req := struct {
		Class       ClassID
		Thread      ThreadID
		Constructor MethodID
		Args        []Value
		Options     InvokeOptions
	}{class, thread, constructor, args, options}
	var res NewInstanceResult
	err := c.get(cmdClassTypeNewInstance, req, &res)
	return res, err
}
