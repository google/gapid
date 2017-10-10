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

package log

import "sync"

// Indirect is a Handler that can be dynamically retarget to another logger.
type Indirect struct {
	mutex  sync.RWMutex
	target Handler
}

// SetTarget assigns the handler target to l, returning the old target.
func (i *Indirect) SetTarget(l Handler) Handler {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	old := i.target
	i.target = l
	return old
}

// Target returns the target handler.
func (i *Indirect) Target() Handler {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.target
}

func (i *Indirect) Handle(m *Message) {
	if t := i.Target(); t != nil {
		t.Handle(m)
	}
}

func (i *Indirect) Close() {
	if t := i.Target(); t != nil {
		t.Close()
	}
}
