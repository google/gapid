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

package dom

// Table represents an HTML <table> element.
type Table struct{ *Element }

// Td represents an HTML <td> element.
type Td struct{ *Element }

// Tr represents an HTML <tr> element.
type Tr struct{ *Element }

// Th represents an HTML <th> element.
type Th struct{ *Element }

// NewTable returns a new Table element.
func NewTable() *Table {
	t := &Table{newEl("Table")}
	t.Set("border", "1")
	return t
}

// NewTd returns a new Td element.
func NewTd() *Td { return &Td{newEl("Td")} }

// NewTr returns a new Tr element.
func NewTr() *Tr { return &Tr{newEl("Tr")} }

// NewTh returns a new Th element.
func NewTh() *Th { return &Th{newEl("Th")} }
