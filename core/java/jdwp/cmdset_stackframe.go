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

// GetThisObject returns the this object for the specified thread and stack
// frame.
func (c *Connection) GetThisObject(thread ThreadID, frame FrameID) (TaggedObjectID, error) {
	req := struct {
		Thread ThreadID
		Frame  FrameID
	}{thread, frame}
	res := TaggedObjectID{}
	err := c.get(cmdStackFrameThisObject, req, &res)
	return res, err
}

type VariableRequest struct {
	Index int
	Tag   uint8
}

// GetValues returns the set of objects for the specified thread and frame,
// based on their slots.
func (c *Connection) GetValues(thread ThreadID, frame FrameID, slots []VariableRequest) ([]Value, error) {
	req := struct {
		Thread ThreadID
		Frame  FrameID
		Slots  []VariableRequest
	}{thread, frame, slots}
	res := ValueSlice{}
	err := c.get(cmdStackFrameGetValues, req, &res)
	return res, err
}

type VariableAssignmentRequest struct {
	Index int
	Value Value
}

// SetValues sets the values for the local variables given thread and frame
func (c *Connection) SetValues(thread ThreadID, frame FrameID, slots []VariableAssignmentRequest) error {
	req := struct {
		Thread ThreadID
		Frame  FrameID
		Slots  []VariableAssignmentRequest
	}{thread, frame, slots}

	err := c.get(cmdStackFrameSetValues, req, nil)
	return err
}
