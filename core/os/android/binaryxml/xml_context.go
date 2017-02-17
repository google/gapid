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

package binaryxml

import (
	"strings"
)

type xmlContext struct {
	rootHolder
	strings    *stringPool
	stack      stack
	namespaces map[string]string
	indent     int
	tab        string
}

type stack []chunk

func (s *stack) push(c chunk) {
	*s = append(*s, c)
}
func (s *stack) pop() {
	*s = (*s)[:len(*s)-1]
}
func (s *stack) head() chunk {
	return (*s)[len(*s)-1]
}

func (c *xmlContext) path() string {
	elems := []string{}
	for _, sc := range c.stack {
		se, ok := sc.(*xmlStartElement)
		if ok {
			elems = append(elems, se.name.get())
		}
	}
	return strings.Join(elems, "/")
}
