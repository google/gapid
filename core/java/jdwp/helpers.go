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
	"context"
	"fmt"
)

// GetClassBySignature returns the single loaded class matching the requested
// signature from the server. If there are no, or more than one class found,
// then an error is returned.
func (c *Connection) GetClassBySignature(signature string) (ClassInfo, error) {
	classes, err := c.GetClassesBySignature(signature)
	if err != nil {
		return ClassInfo{}, err
	}
	if len(classes) != 1 {
		err := fmt.Errorf("%d classes found with the signature '%v'", len(classes), signature)
		return ClassInfo{}, err
	}
	return classes[0], nil
}

// GetLocationMethodName returns the name of the method from the location.
func (c *Connection) GetLocationMethodName(l Location) (string, error) {
	methods, err := c.GetMethods(ReferenceTypeID(l.Class))
	if err != nil {
		return "", err
	}
	method := methods.FindByID(l.Method)
	if method == nil {
		return "", fmt.Errorf("Method not found with ID %v", l.Method)
	}
	return method.Name, nil
}

// GetClassMethod looks up the method with the specified signature on class.
func (c *Connection) GetClassMethod(class ClassID, name, signature string) (Method, error) {
	methods, err := c.GetMethods(ReferenceTypeID(class))
	if err != nil {
		return Method{}, err
	}
	method := methods.FindBySignature(name, signature)
	if method == nil {
		return Method{}, fmt.Errorf("Method '%s%s' not found", name, signature)
	}
	return *method, nil
}

// WaitForClassPrepare blocks until a class with a name that matches the pattern
// is prepared, and then returns the thread that prepared the class.
// All threads are suspended when the method returns.
func (c *Connection) WaitForClassPrepare(ctx context.Context, pattern string) (ThreadID, error) {
	var out ThreadID

	onEvent := func(event Event) bool {
		out = event.(*EventClassPrepare).Thread
		return false
	}

	err := c.WatchEvents(ctx, ClassPrepare, SuspendAll, onEvent, ClassMatchEventModifier(pattern))
	if err != nil {
		return 0, err
	}

	return out, nil
}

// WaitForMethodEntry blocks until the method on class is entered, and then
// returns the method entry event.
// All threads are suspended when the method returns.
func (c *Connection) WaitForMethodEntry(ctx context.Context, class ClassID, method MethodID) (*EventMethodEntry, error) {
	var out *EventMethodEntry

	onEvent := func(event Event) bool {
		e := event.(*EventMethodEntry)
		if e.Location.Method == method {
			out = e
			return false
		}
		c.ResumeAll()
		return true
	}

	err := c.WatchEvents(ctx, MethodEntry, SuspendAll, onEvent, ClassOnlyEventModifier(class))
	if err != nil {
		return nil, err
	}

	return out, nil
}
