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

	"github.com/google/gapid/framework/binary"
)

type Leaf struct {
	binary.Generate `java:"disable"`
	A               uint32
}

type Anonymous struct {
	binary.Generate `java:"disable"`
	Leaf
}

type Contains struct {
	binary.Generate `java:"disable"`
	LeafField       Leaf
}

type Array struct {
	binary.Generate `java:"disable"`
	Leaves          [3]Leaf
}

type Slice struct {
	binary.Generate `java:"disable"`
	Leaves          []Leaf
}

type MapKey struct {
	binary.Generate `java:"disable"`
	M               Leafːuint32
}

type MapValue struct {
	binary.Generate `java:"disable"`
	M               Uint32ːLeaf
}

type MapKeyValue struct {
	binary.Generate `java:"disable"`
	M               LeafːLeaf
}

type ArrayInMap struct {
	binary.Generate `java:"disable"`
	M               Uint32ː3_Leaf
}

type SliceInMap struct {
	binary.Generate `java:"disable"`
	M               Uint32ːSliceLeaf
}

type MapInSlice struct {
	binary.Generate `java:"disable"`
	Slice           []Uint32ːuint32
}

type MapInArray struct {
	binary.Generate `java:"disable"`
	Array           [2]Uint32ːuint32
}

type MapOfMaps struct {
	binary.Generate `java:"disable"`
	M               Uint32ːLeafːLeaf
}

type ArrayOfArrays struct {
	binary.Generate `java:"disable"`
	Array           [2][3]Leaf
}

type SliceOfSlices struct {
	binary.Generate `java:"disable"`
	Slice           [][]Leaf
}

type Complex struct {
	binary.Generate `java:"disable"`
	SliceMapArray   []Containsː3_Contains
	SliceArrayMap   [][3]ContainsːContains
	ArraySliceMap   [3][]ContainsːContains
	ArrayMapSlice   [3]ContainsːSliceContains
	MapArraySlice   Containsː3_SliceContains
	MapSliceArray   ContainsːSlice3_Contains
}

type Leafːuint32 map[Leaf]uint32
type LeafːLeaf map[Leaf]Leaf

type Leaf_SortKeys []Leaf

func (s Leaf_SortKeys) Len() int           { return len(s) }
func (s Leaf_SortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Leaf_SortKeys) Less(i, j int) bool { return fmt.Sprint(s[i]) < fmt.Sprint(s[j]) }

func (m Leafːuint32) KeysSorted() []Leaf {
	s := make(Leaf_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m LeafːLeaf) KeysSorted() []Leaf {
	s := make(Leaf_SortKeys, len(m))
	i := 0
	for k, _ := range m {
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

func (m Uint32ːLeaf) KeysSorted() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ː3_Leaf) KeysSorted() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːSliceLeaf) KeysSorted() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːuint32) KeysSorted() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Uint32ːLeafːLeaf) KeysSorted() []uint32 {
	s := make(Uint32_SortKeys, len(m))
	i := 0
	for k, _ := range m {
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

func (m Containsː3_Contains) KeysSorted() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːContains) KeysSorted() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːSliceContains) KeysSorted() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m Containsː3_SliceContains) KeysSorted() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

func (m ContainsːSlice3_Contains) KeysSorted() []Contains {
	s := make(Contains_SortKeys, len(m))
	i := 0
	for k, _ := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}
