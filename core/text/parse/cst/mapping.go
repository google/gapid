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

package cst

import "sync"

// Map is the interface to an object into which ast<->cts mappings are stored.
type Map interface {
	// SetCST is called to map an ast node to a CST node.
	SetCST(ast interface{}, cst Node)
	// CST is called to lookup the CST mapping for an AST node.
	CST(ast interface{}) Node
}

// cstMap is a simple implementation of Map that just stores the mappings.
type cstMap struct {
	mappings map[interface{}]Node
	mutex    sync.Mutex
}

func (m *cstMap) SetCST(ast interface{}, cst Node) {
	m.mutex.Lock()
	m.mappings[ast] = cst
	m.mutex.Unlock()
}

func (m *cstMap) CST(ast interface{}) Node {
	m.mutex.Lock()
	out := m.mappings[ast]
	m.mutex.Unlock()
	return out
}

// NewMap returns a new Map instance.
func NewMap() Map {
	return &cstMap{mappings: make(map[interface{}]Node)}
}
