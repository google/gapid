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

package semantic

import (
	"fmt"
	"sort"
)

// Symbols is an object with named members and no other functionality.
type Symbols struct {
	sorted  bool
	entries byName
}

// AddNamed inserts a named node into the symbol space.
func (s *Symbols) AddNamed(entry NamedNode) {
	s.entries = append(s.entries, namedEntry{name: entry.Name(), node: entry})
	s.sorted = false
}

// Add inserts a node into the symbol space with the specified name.
func (s *Symbols) Add(name string, entry Node) {
	s.entries = append(s.entries, namedEntry{name: name, node: entry})
	s.sorted = false
}

func (s *Symbols) Visit(visitor func(string, Node)) {
	s.sort()
	for _, e := range s.entries {
		visitor(e.name, e.node)
	}
}

func (s *Symbols) Find(name string) (Node, error) {
	i := s.find(name)
	if i >= len(s.entries) || s.entries[i].name != name {
		return nil, nil
	}
	match := s.entries[i].node
	if i+1 < len(s.entries) && s.entries[i+1].name == name {
		return match, fmt.Errorf("Ambiguous match")
	}
	return match, nil
}

func (s *Symbols) FindAll(name string) []Node {
	i := s.find(name)
	result := []Node{}
	for ; i < len(s.entries) && s.entries[i].name == name; i++ {
		result = append(result, s.entries[i].node)
	}
	return result
}

func (s *Symbols) find(name string) int {
	s.sort()
	return sort.Search(len(s.entries), func(i int) bool { return s.entries[i].name >= name })
}

func (s *Symbols) sort() {
	if !s.sorted {
		//we use a stable sort for deterministic iteration order
		sort.Stable(s.entries)
		s.sorted = true
	}
}

type namedEntry struct {
	name string
	node Node
}

type byName []namedEntry

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].name < a[j].name }
