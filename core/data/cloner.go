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

package data

// Cloner holds onto references as a type is being
// cloned, making sure to handle back-references and
// circular references
type Cloner struct {
	cloned map[interface{}]interface{}
}

func (c *Cloner) Get(i interface{}) (interface{}, bool) {
	if cloned, ok := c.cloned[i]; ok {
		return cloned, true
	}
	return nil, false
}

func (c *Cloner) Add(src, dst interface{}) {
	c.cloned[src] = dst
}

func NewCloner() *Cloner {
	return &Cloner{make(map[interface{}]interface{})}
}
