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

package schema

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
)

type Constants []ConstantSet

type ConstantSet struct {
	Type    binary.Type // The type of the constant.
	Entries []Constant  // The constant values
}

type Constant struct {
	Name  string
	Value interface{}
}

func (c Constants) Len() int           { return len(c) }
func (c Constants) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Constants) Less(i, j int) bool { return c[i].Type.String() < c[j].Type.String() }

func (c *Constants) Add(t binary.Type, v Constant) {
	for i := range *c {
		s := &(*c)[i]
		if s.Type.String() == t.String() {
			s.Entries = append(s.Entries, v)
			return
		}
	}
	*c = append(*c, ConstantSet{Type: t, Entries: []Constant{v}})
}

func (s *ConstantSet) Len() int      { return len(s.Entries) }
func (s *ConstantSet) Swap(i, j int) { s.Entries[i], s.Entries[j] = s.Entries[j], s.Entries[i] }
func (s *ConstantSet) Less(i, j int) bool {
	vi := reflect.ValueOf(s.Entries[i].Value)
	vj := reflect.ValueOf(s.Entries[j].Value)
	switch vi.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch vj.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if vi.Int() == vj.Int() {
				return s.Entries[i].Name < s.Entries[j].Name
			}
			return vi.Int() < vj.Int()
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch vj.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if vi.Uint() == vj.Uint() {
				return s.Entries[i].Name < s.Entries[j].Name
			}
			return vi.Uint() < vj.Uint()
		}
	}
	si := fmt.Sprint(s.Entries[i].Value)
	sj := fmt.Sprint(s.Entries[j].Value)
	if si == sj {
		return s.Entries[i].Name < s.Entries[j].Name
	}
	return si < sj
}

func (c Constant) String() string { return c.Name }
