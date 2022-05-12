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

package compiler

import (
	"fmt"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/semantic"
)

// mapImpl holds information about the types used in a map implementation.
// Maps of different semantic types may reduce down to the same single
// implementation.
type mapImpl struct {
	k, v semantic.Type
	i    *MapInfo
}

// This is a basic linear-probing hash map
// It works out to the following implementation
// enum used {
//    empty,
//    full,
//    previously_full
// }
// struct element {
//    used used;
//    KeyType k;
//    ValueType v;
// }
// struct Map {
//    uint32_t ref_count;
//    arena*   arena;
//    uint64_t count;
//    uint64_t capacity;
//    element* elements;
// }

// In order to look up something in the map
// 1) h = Hashed Key % capacity
// 2) Check element[h].
//  If used == empty, then the element does not exist in the map
// 3) If used == previously_full, then it is not here, check h = h + 1: goto 2
// 4) If used == full, if k != key: h = h + 1: goto 2, otherwise you found it

// Removal is as simple as
// 1) Find element
// 2) set used == previously_full

// Once the map hits > 80% capacity, we should rehash the map larger.
//  Otherwise collisions will turn this into a linear search.

// Insertion into the map is similar (assuming it doesn't exist)
//   1) h = Hashed Key % capacity
//   2) rehash if necessary
//   3) find first bucket >= h, where used != full (mod capacity)
//   4) Insert there, mark used = full

// TODO: Investigate rehashing once #full + #previously_full > 80%
//     If we end up with lots of insertions/deletions, this will prevent linear search

func (c *C) defineMapTypes() {
	// impls is a map of type mangled name to the public MapInfo structure.
	// This is used to deduplicate maps that have the same underlying key and
	// value LLVM types when lowered.
	impls := map[string]*MapInfo{}

	for _, api := range c.APIs {
		for _, t := range api.Maps {
			mi := &MapInfo{
				Key:  c.T.Target(t.KeyType),
				Val:  c.T.Target(t.ValueType),
				Type: c.T.target[t].(codegen.Pointer).Element.(*codegen.Struct),
			}

			mi.Elements = c.T.Struct(fmt.Sprintf("%v…%v", mi.Key.TypeName(), mi.Val.TypeName()),
				// Used: 0 == empty, 1 == has a key, 2 == doesn't have a key, but
				//    can't assume your searched key doesn't exist
				codegen.Field{Name: "used", Type: c.T.Target(semantic.Uint64Type)},
				codegen.Field{Name: "k", Type: mi.Key},
				codegen.Field{Name: "v", Type: mi.Val},
			)

			mi.Type.SetBody(false,
				codegen.Field{Name: MapRefCount, Type: c.T.Uint32},
				codegen.Field{Name: MapArena, Type: c.T.ArenaPtr},
				codegen.Field{Name: MapCount, Type: c.T.Uint64},
				codegen.Field{Name: MapCapacity, Type: c.T.Uint64},
				codegen.Field{Name: MapElements, Type: c.T.Pointer(mi.Elements)},
			)

			// Use the mangled name of the map to determine whether the map has
			// already been declared for the lowered map type.
			mangled := c.Mangler(c.Mangle(mi.Type))
			impl, seen := impls[mangled]

			if !seen {
				// First instance of this lowered map type. Define it.
				copy := *mi
				impl = &copy
				impls[mangled] = impl
				c.T.mapImpls = append(c.T.mapImpls, mapImpl{t.KeyType, t.ValueType, impl})
			}

			c.T.Maps[t] = mi
		}
	}
}

// IterateMap emits a map iteration calling cb for each element in the map
// where:
// i is a pointer to the sequential index of the element starting from 0 of type
//   idxTy.
// k is a pointer to the element key.
// v is a pointer to the element value.
func (c *C) IterateMap(s *S, mapPtr *codegen.Value, idxTy semantic.Type, cb func(i, k, v *codegen.Value)) {
	capacity := mapPtr.Index(0, MapCapacity).Load()
	elPtr := mapPtr.Index(0, MapElements).Load()
	iTy := c.T.Target(idxTy)
	i := s.LocalInit("i", s.Scalar(0).Cast(iTy))
	s.ForN(capacity.Cast(iTy), func(s *S, it *codegen.Value) *codegen.Value {
		used := elPtr.Index(it, "used").Load()
		s.If(s.Equal(used, s.Scalar(mapElementFull)), func(s *S) {
			k := elPtr.Index(it, "k")
			v := elPtr.Index(it, "v")
			cb(i, k, v)
			i.Store(s.Add(i.Load(), s.Scalar(1).Cast(iTy)))
		})
		return nil
	})
}
