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

package note

import "github.com/google/gapid/core/fault"

// ErrNoteLimit is thrown when an note list reaches capacity.
const ErrNoteLimit = fault.Const("Note limit reached")

// CollectAll returns a handler that collects notes into the list.
func CollectAll(l *Pad) Handler {
	return func(p Page) error {
		*l = append(*l, p)
		return nil
	}
}

// Collect returns a handler that collects notes into the list.
// If it reaches the specified limit of notes, it will return ErrNoteLimit.
func Collect(l *Pad, limit int) Handler {
	return func(p Page) error {
		if len(*l) >= limit {
			return ErrNoteLimit
		}
		*l = append(*l, p)
		return nil
	}
}
