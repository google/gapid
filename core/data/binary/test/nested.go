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

package test

import (
	"fmt"
	"sort"
)

type Leaf struct {
	A uint32
}

type Anonymous struct {
	Leaf
}

type Contains struct {
	LeafField Leaf
}

type Array struct {
	Leaves [3]Leaf
}

type Slice struct {
	Leaves []Leaf
}

type MapKey struct {
	M Leafːuint32
}

type MapValue struct {
	M Uint32ːLeaf
}

type MapKeyValue struct {
	M LeafːLeaf
}

type ArrayInMap struct {
	M Uint32ː3_Leaf
}

type SliceInMap struct {
	M Uint32ːSliceLeaf
}

type MapInSlice struct {
	Slice []Uint32ːuint32
}

type MapInArray struct {
	Array [2]Uint32ːuint32
}

type MapOfMaps struct {
	M Uint32ːLeafːLeaf
}

type ArrayOfArrays struct {
	Array [2][3]Leaf
}

type SliceOfSlices struct {
	Slice [][]Leaf
}

type Complex struct {
	SliceMapArray []Containsː3_Contains
	SliceArrayMap [][3]ContainsːContains
	ArraySliceMap [3][]ContainsːContains
	ArrayMapSlice [3]ContainsːSliceContains
	MapArraySlice Containsː3_SliceContains
	MapSliceArray ContainsːSlice3_Contains
}

type Leafːuint32 map[Leaf]uint32
type LeafːLeaf map[Leaf]Leaf

type Leaf_SortKeys []Leaf

func (s Leaf_SortKeys) Len() int           { return len(s) }
func (s Leaf_SortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Leaf_SortKeys) Less(i, j int) bool { return fmt.Sprint(s[i]) < fmt.Sprint(s[j]) }

func (m Leafːuint32) Keys() []Leaf {
	s := make(Leaf_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m LeafːLeaf) Keys() []Leaf {
	s := make(Leaf_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

type Uint32ːLeaf map[uint32]Leaf
type Uint32ː3_Leaf map[uint32][3]Leaf
type Uint32ːSliceLeaf map[uint32][]Leaf
type Uint32ːuint32 map[uint32]uint32
type Uint32ːLeafːLeaf map[uint32]LeafːLeaf

type Uint32_SortKeys []uint32

func (s Uint32_SortKeys) Len() int           { return len(s) }
func (s Uint32_SortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Uint32_SortKeys) Less(i, j int) bool { return s[i] < s[j] }

func (m Uint32ːLeaf) Keys() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ː3_Leaf) Keys() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːSliceLeaf) Keys() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːuint32) Keys() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːLeafːLeaf) Keys() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

type Containsː3_Contains map[Contains][3]Contains
type ContainsːContains map[Contains]Contains
type ContainsːSliceContains map[Contains][]Contains
type Containsː3_SliceContains map[Contains][3][]Contains
type ContainsːSlice3_Contains map[Contains][][3]Contains

type Contains_SortKeys []Contains

func (s Contains_SortKeys) Len() int           { return len(s) }
func (s Contains_SortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Contains_SortKeys) Less(i, j int) bool { return fmt.Sprint(s[i]) < fmt.Sprint(s[j]) }

func (m Containsː3_Contains) Keys() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːContains) Keys() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːSliceContains) Keys() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Containsː3_SliceContains) Keys() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːSlice3_Contains) Keys() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}
