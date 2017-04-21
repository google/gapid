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

// Package constset provides structures for storing large numbers of
// value-string pairs efficently.
package constset

// Pack holds the symbols and all constant sets.
type Pack struct {
	Symbols Symbols
	Sets    []Set
}

// Symbols holds the symbol names.
type Symbols string

// Get returns the symbol for the entry e.
func (s Symbols) Get(e Entry) string { return string(s[e.O : e.O+e.L]) }

// Set is a list of entries.
type Set struct {
	IsBitfield bool
	Entries    []Entry
}

// Entry is a single value-symbol entry.
type Entry struct {
	V uint64 // The value
	O int    // The symbol table offset
	L int    // The symbol length
}
