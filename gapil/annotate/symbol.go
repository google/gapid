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

package annotate

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/snippets"
)

func category(expr semantic.Expression) snippets.SymbolCategory {
	switch expr.(type) {
	case *semantic.Global:
		return snippets.SymbolCategory_Global
	case *semantic.Local:
		return snippets.SymbolCategory_Local
	case *semantic.Parameter:
		return snippets.SymbolCategory_Parameter
	}
	panic(fmt.Errorf("No symbol category for %T:%v", expr, expr))
}

// SymbolTable is a basic mapping between names and values
type SymbolTable map[string]*Location

func (s SymbolTable) String() string {
	buf := &bytes.Buffer{}
	first := true
	for k, v := range s {
		if v.leader().isEmpty() {
			continue
		}
		if !first {
			buf.WriteString(", ")
		} else {
			first = false
		}
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(fmt.Sprintf("%s", v))
	}
	return buf.String()
}

// ScopedSymbolTable is a symbol table with scopes which allow shadowing.
type ScopedSymbolTable []SymbolTable

// enter starts a new scope
func (t *ScopedSymbolTable) enter() {
	(*t) = append(*t, make(SymbolTable))
}

// leave ends the current scope. It will panic if there is no current scope.
func (t *ScopedSymbolTable) leave() {
	l := len(*t)
	if l == 0 {
		panic(fmt.Errorf("Scope imbalance, leave() called more times than enter()"))
	}
	(*t) = (*t)[:l-1]
}

// lookup finds a mapping for name in the outermost scope and return the
// leader of the equivalence set it is in.
func (t *ScopedSymbolTable) lookup(name string) *Location {
	if name == "_" {
		return nil
	}

	for i := len(*t) - 1; i >= 0; i-- {
		s := (*t)[i]
		if loc, ok := s[name]; ok {
			return loc.leader()
		}
	}
	panic(fmt.Errorf("Failed to locate %s", name))
}

// declares a new name in the symbol table. It will panic if the name is
// already in use. The location for the new declaration is returned.
func (t *ScopedSymbolTable) declare(name string) *Location {
	if name == "_" {
		return nil
	}

	location := &Location{}
	t.add(name, location)
	return location
}

// add inserts new name in the symbol table. It will panic if the name is
// already in use.
func (t *ScopedSymbolTable) add(name string, location *Location) {
	l := len(*t)
	if l == 0 {
		panic(fmt.Errorf("Declare called without any scope: %s", name))
	}
	s := (*t)[l-1]
	if _, found := s[name]; found {
		panic(fmt.Errorf("Attempt to redeclare %s", name))
	}

	s[name] = location
}

// SymbolSpace is a ScopedSymbolTable for each SymbolCategory.
type SymbolSpace map[snippets.SymbolCategory]*ScopedSymbolTable

func MakeSymbolSpace() SymbolSpace {
	space := make(SymbolSpace)
	for _, cat := range []snippets.SymbolCategory{
		snippets.SymbolCategory_Global, snippets.SymbolCategory_Local, snippets.SymbolCategory_Parameter} {
		space[cat] = &ScopedSymbolTable{}
	}
	return space
}

// enter starts a new scope
func (s SymbolSpace) enter() {
	for _, t := range s {
		t.enter()
	}
}

// leave ends the current scope. It will panic if there is no current scope.
func (s SymbolSpace) leave() {
	for _, t := range s {
		t.leave()
	}
}

func (s SymbolSpace) String() string {
	buf := &bytes.Buffer{}
	first := true
	for k, v := range s {
		if !first {
			buf.WriteString(", ")
		} else {
			first = false
		}
		buf.WriteString(fmt.Sprintf("%s{%s}", k, *v))
	}
	return buf.String()
}
