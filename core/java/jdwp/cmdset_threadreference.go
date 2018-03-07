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

// GetThreadName returns a thread's name.
func (c *Connection) GetThreadName(id ThreadID) (string, error) {
	var res string
	err := c.get(cmdThreadReferenceName, id, &res)
	return res, err
}

// Suspend suspends the specified thread.
func (c *Connection) Suspend(id ThreadID) error {
	var res struct{}
	return c.get(cmdThreadReferenceSuspend, id, &res)
}

// Resume resumes the specified thread.
func (c *Connection) Resume(id ThreadID) error {
	var res struct{}
	return c.get(cmdThreadReferenceResume, id, &res)
}

// GetThreadStatus returns the status of the thread.
func (c *Connection) GetThreadStatus(id ThreadID) (ThreadStatus, SuspendStatus, error) {
	var res struct {
		T ThreadStatus
		S SuspendStatus
	}
	err := c.get(cmdThreadReferenceStatus, id, &res)
	if err != nil {
		return 0, 0, err
	}
	return res.T, res.S, nil
}

// GetSuspendCount returns the number of times the thread has been suspended
// without a corresponding resume.
func (c *Connection) GetSuspendCount(id ThreadID) (int, error) {
	var count int
	err := c.get(cmdThreadReferenceSuspendCount, id, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FrameInfo describes a single stack frame.
type FrameInfo struct {
	Frame    FrameID
	Location Location
}

// GetFrames returns a number of stack frames.
func (c *Connection) GetFrames(thread ThreadID, start, count int) ([]FrameInfo, error) {
	req := struct {
		Thread       ThreadID
		Start, Count int
	}{thread, start, count}
	var res []FrameInfo
	err := c.get(cmdThreadReferenceFrames, req, &res)
	return res, err
}
