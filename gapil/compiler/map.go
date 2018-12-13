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

			valPtrTy := c.T.Pointer(mi.Val)

			name := fmt.Sprintf("%v_%v", api.Name(), t.Name())
			mi.MapMethods = MapMethods{
				Contains:  c.M.Function(c.T.Bool, name+"_contains", c.T.Pointer(mi.Type), mi.Key).LinkPrivate().Inline(),
				Index:     c.M.Function(valPtrTy, name+"_index", c.T.Pointer(mi.Type), mi.Key, c.T.Bool).LinkPrivate().Inline(),
				Remove:    c.M.Function(c.T.Void, name+"_remove", c.T.Pointer(mi.Type), mi.Key).LinkPrivate().Inline(),
				Clear:     c.M.Function(c.T.Void, name+"_clear", c.T.Pointer(mi.Type)).LinkPrivate().Inline(),
				ClearKeep: c.M.Function(c.T.Void, name+"_clear_keep", c.T.Pointer(mi.Type)).LinkPrivate().Inline(),
			}

			// Use the mangled name of the map to determine whether the map has
			// already been declared for the lowered map type.
			mangled := c.Mangler(c.Mangle(mi.Type))
			impl, seen := impls[mangled]

			if !seen {
				// First instance of this lowered map type. Define it.
				copy := *mi
				impl = &copy
				impls[mangled] = impl
				impl.MapMethods = MapMethods{
					Contains:  c.Method(true, mi.Type, c.T.Bool, "contains", mi.Key).LinkOnceODR(),
					Index:     c.Method(false, mi.Type, valPtrTy, "index", mi.Key, c.T.Bool).LinkOnceODR(),
					Remove:    c.Method(false, mi.Type, c.T.Void, "remove", mi.Key).LinkOnceODR(),
					Clear:     c.Method(false, mi.Type, c.T.Void, "clear").LinkOnceODR(),
					ClearKeep: c.Method(false, mi.Type, c.T.Void, "clear_keep").LinkOnceODR(),
				}
				c.T.mapImpls = append(c.T.mapImpls, mapImpl{t.KeyType, t.ValueType, impl})
			}

			// Delegate the methods of this map on to the common implmentation.
			c.Delegate(mi.Contains, impl.Contains)
			c.Delegate(mi.Index, impl.Index)
			c.Delegate(mi.Remove, impl.Remove)
			c.Delegate(mi.Clear, impl.Clear)
			c.Delegate(mi.ClearKeep, impl.ClearKeep)

			c.T.Maps[t] = mi
		}
	}
}

// If we know the values are going to be small & sequential, we can
// swap out this hash.
func (c *C) hash64Bit(s *S, value *codegen.Value) *codegen.Value {
	rotateRight := func(value *codegen.Value, bits int) *codegen.Value {
		v := s.ShiftRight(value, s.Scalar(uint64(bits)))
		v = s.ShiftLeft(value, s.Scalar(uint64(64-bits)))
		v = s.Or(v, v)
		return v.SetName(">>>")
	}

	shiftLeft := func(value *codegen.Value, bits int) *codegen.Value {
		return s.ShiftLeft(value, s.Scalar(uint64(bits)))
	}

	v := value
	v = s.Invert(v).SetName("_hash1")
	v = s.Add(v, shiftLeft(v, 21)).SetName("_hash2")
	v = s.Xor(v, rotateRight(v, 24)).SetName("_hash3")
	v = s.Add(s.Add(v, shiftLeft(v, 3)), shiftLeft(v, 8)).SetName("_hash4")
	v = s.Xor(v, rotateRight(v, 14)).SetName("_hash5")
	v = s.Add(s.Add(v, shiftLeft(v, 2)), shiftLeft(v, 4)).SetName("_hash6")
	v = s.Xor(v, rotateRight(v, 28)).SetName("_hash7")
	v = s.Add(v, shiftLeft(v, 31)).SetName("_hash8")
	return v
}

func (c *C) hashVariableValue(s *S, pointer *codegen.Value, numBytes *codegen.Value) *codegen.Value {
	u64Type := c.T.Target(semantic.Uint64Type)
	numBytes = numBytes.Load().Cast(u64Type)
	v := s.Local("_hash", u64Type)
	v.Store(s.Scalar(uint64(0)))
	s.ForN(numBytes, func(s *S, it *codegen.Value) *codegen.Value {
		tv := v.Load()
		l6 := s.ShiftLeft(tv, s.Scalar(uint64(6)))
		l16 := s.ShiftLeft(tv, s.Scalar(uint64(16)))
		dat := pointer.Index(0, it).Load().Cast(u64Type)
		r := s.Add(dat, l6)
		r = s.Add(r, l16)
		r = s.Add(r, tv)
		v.Store(r)
		return nil
	})
	return v.Load()
}

func (c *C) hashValue(s *S, t semantic.Type, value *codegen.Value) *codegen.Value {
	keyType := c.T.Target(t)
	u64Type := c.T.Target(semantic.Uint64Type)
	u32Type := c.T.Target(semantic.Uint32Type)
	if keyType != value.Type() {
		fail("hashValue passed unexpected type. Got: %+v, expect: %+v", value.Type(), keyType)
	}

	switch t := semantic.Underlying(t).(type) {
	case *semantic.Builtin:
		switch t {
		case semantic.BoolType,
			semantic.IntType,
			semantic.UintType,
			semantic.SizeType,
			semantic.CharType,
			semantic.Int8Type,
			semantic.Uint8Type,
			semantic.Int16Type,
			semantic.Uint16Type,
			semantic.Int32Type,
			semantic.Uint32Type,
			semantic.Int64Type,
			semantic.Uint64Type:
			return value.Cast(u64Type)
		case semantic.Float32Type:
			return c.hash64Bit(s, value.Bitcast(u32Type).Cast(u64Type))
		case semantic.Float64Type:
			return c.hash64Bit(s, value.Bitcast(u32Type))
		case semantic.StringType:
			return c.hashVariableValue(s, value.Index(0, "data"), value.Index(0, "length"))
		default:
			fail("Cannot determine the hash for %T, %v", t, t)
			return nil
		}
	case *semantic.Enum:
		return value.Cast(u64Type)
	case *semantic.Pointer:
		return s.ShiftRight(value.Cast(u64Type), s.Scalar(uint64(2)))
	case *semantic.StaticArray:
		fail("Cannot use a static array as a hash key")
		return nil
	case *semantic.Reference:
		fail("Cannot use a reference as a hash key")
		return nil
	case *semantic.Class:
		// Cannot hash a class
		fail("Cannot hash a class")
		return nil
	default:
		fail("Cannot determine the hash of %T", t)
		return nil
	}
}

func (c *C) buildMapTypes() {
	for _, m := range c.T.mapImpls {
		c.buildMap(m.k, m.v, m.i)
	}
}

func (c *C) buildMap(keyTy, valTy semantic.Type, mi *MapInfo) {
	elTy := mi.Elements
	u64Type := c.T.Target(semantic.Uint64Type)

	checkRefCount := func(s *S, m *codegen.Value) {
		cnt := m.Index(0, MapRefCount).Load()
		s.If(s.LessOrEqualTo(cnt, s.Zero(cnt.Type())), func(s *S) {
			c.LogF(s, "Map refcount is %d", cnt)
		})
	}

	c.Build(mi.Contains, func(s *S) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")

		checkRefCount(s, m)

		h := c.hashValue(s, keyTy, k)

		capacity := m.Index(0, MapCapacity).Load()
		elements := m.Index(0, MapElements).Load()
		s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
			check := s.And(s.Add(h, it), s.Sub(capacity, s.Scalar(uint64(1))))
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementEmpty)), func(s *S) {
				s.Return(s.Scalar(false))
			})
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
				key := elements.Index(check, "k")
				found := c.equal(s, key.Load(), k)
				s.If(found, func(s *S) { s.Return(s.Scalar(true)) })
			})
			return nil
		})
		s.Return(s.Scalar(false))
	})

	f32Type := c.T.Target(semantic.Float32Type)
	c.Build(mi.Index, func(s *S) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		addIfNotFound := s.Parameter(2).SetName("addIfNotFound")
		s.Arena = m.Index(0, MapArena).Load().SetName("arena")

		checkRefCount(s, m)

		countPtr := m.Index(0, MapCount)
		capacityPtr := m.Index(0, MapCapacity)
		elementsPtr := m.Index(0, MapElements)
		count := countPtr.Load()
		capacity := capacityPtr.Load()
		elements := elementsPtr.Load()

		h := c.hashValue(s, keyTy, k)
		// Search for existing
		s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
			check := s.And(s.Add(h, it), s.Sub(capacity, s.Scalar(uint64(1))))
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
				found := c.equal(s, elements.Index(check, "k").Load(), k)
				s.If(found, func(s *S) {
					s.Return(elements.Index(check, "v"))
				})
			})

			return s.Not(c.equal(s, valid, s.Scalar(mapElementEmpty)))
		})

		s.If(addIfNotFound, func(s *S) {
			resize := s.LocalInit("resize", elements.IsNull())
			s.If(s.Not(resize.Load()), func(s *S) {
				used := s.Div(count.Cast(f32Type), capacity.Cast(f32Type))
				resize.Store(s.GreaterThan(used, s.Scalar(float32(mapMaxCapacity))))
			})

			getStorageBucket := func(h, table, tablesize *codegen.Value) *codegen.Value {
				newBucket := s.Local("newBucket", u64Type)
				s.ForN(tablesize, func(s *S, it *codegen.Value) *codegen.Value {
					check := s.And(s.Add(h, it), s.Sub(tablesize, s.Scalar(uint64(1)))).SetName("hash_bucket")
					newBucket.Store(check)
					valid := table.Index(check, "used").Load()
					notFound := c.equal(s, valid, s.Scalar(mapElementFull))
					return notFound
				})
				return newBucket.Load()
			}

			s.If(resize.Load(), func(s *S) {
				// Grow
				s.IfElse(elements.IsNull(), func(s *S) {
					capacity := s.Scalar(uint64(minMapSize))
					capacityPtr.Store(capacity)
					elements := c.Alloc(s, capacity, elTy)
					elementsPtr.Store(elements)
					s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
						elements.Index(it, "used").Store(s.Scalar(mapElementEmpty))
						return nil
					})
				}, /* else */ func(s *S) {
					newCapacity := s.MulS(capacity, uint64(mapGrowMultiplier))
					capacityPtr.Store(newCapacity)
					newElements := c.Alloc(s, newCapacity, elTy)
					s.ForN(newCapacity, func(s *S, it *codegen.Value) *codegen.Value {
						newElements.Index(it, "used").Store(s.Scalar(mapElementEmpty))
						return nil
					})

					s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
						valid := elements.Index(it, "used").Load()
						s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
							k := elements.Index(it, "k").Load()
							v := elements.Index(it, "v").Load()
							h := c.hashValue(s, keyTy, k)
							bucket := getStorageBucket(h, newElements, newCapacity)
							newElements.Index(bucket, "k").Store(k)
							newElements.Index(bucket, "v").Store(v)
							newElements.Index(bucket, "used").Store(s.Scalar(mapElementFull))
						})
						return nil
					})
					c.Free(s, elements)
					elementsPtr.Store(newElements)
				})
			})

			count := countPtr.Load()
			capacity := capacityPtr.Load()
			elements := elementsPtr.Load()
			bucket := getStorageBucket(h, elements, capacity)
			elements.Index(bucket, "k").Store(k)
			elements.Index(bucket, "used").Store(s.Scalar(mapElementFull))
			valPtr := elements.Index(bucket, "v")
			v := c.initialValue(s, valTy)
			valPtr.Store(v)
			countPtr.Store(s.AddS(count, uint64(1)))

			c.reference(s, v, valTy)
			c.reference(s, k, keyTy)

			s.Return(valPtr)
		})
		s.Return(s.Zero(c.T.Pointer(mi.Val)))
	})

	c.Build(mi.Remove, func(s *S) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		s.Arena = m.Index(0, MapArena).Load().SetName("arena")

		checkRefCount(s, m)

		countPtr := m.Index(0, MapCount)
		capacity := m.Index(0, MapCapacity).Load()
		h := c.hashValue(s, keyTy, k)
		elements := m.Index(0, MapElements).Load()
		// Search for existing
		s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
			check := s.And(s.Add(h, it), s.Sub(capacity, s.Scalar(uint64(1))))
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
				found := c.equal(s, elements.Index(check, "k").Load(), k)
				s.If(found, func(s *S) {
					elPtr := elements.Index(check)
					// Release references to el
					if c.isRefCounted(keyTy) {
						c.release(s, elPtr.Index(0, "k").Load(), keyTy)
					}
					if c.isRefCounted(valTy) {
						c.release(s, elPtr.Index(0, "v").Load(), valTy)
					}
					// Replace element with last
					elPtr.Index(0, "used").Store(s.Scalar(mapElementUsed))
					count := countPtr.Load()
					countM1 := s.SubS(count, uint64(1)).SetName("count-1")
					// Decrement count
					countPtr.Store(countM1)
					s.Return(nil)
				})
			})

			return s.Not(c.equal(s, valid, s.Scalar(mapElementEmpty)))
		})
	})

	c.Build(mi.Clear, func(s *S) {
		m := s.Parameter(0).SetName("map")
		s.Arena = m.Index(0, MapArena).Load().SetName("arena")

		capacity := m.Index(0, MapCapacity).Load()
		elements := m.Index(0, MapElements).Load()
		if c.isRefCounted(keyTy) || c.isRefCounted(valTy) {
			s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
				valid := elements.Index(it, "used").Load()
				s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
					if c.isRefCounted(keyTy) {
						c.release(s, elements.Index(it, "k").Load(), keyTy)
					}
					if c.isRefCounted(valTy) {
						c.release(s, elements.Index(it, "v").Load(), valTy)
					}
				})
				return nil
			})
		}
		c.Free(s, elements)
		m.Index(0, MapCount).Store(s.Scalar(uint64(0)))
		m.Index(0, MapCapacity).Store(s.Scalar(uint64(0)))
	})

	c.Build(mi.ClearKeep, func(s *S) {
		m := s.Parameter(0).SetName("map")
		s.Arena = m.Index(0, MapArena).Load().SetName("arena")

		capacity := m.Index(0, MapCapacity).Load()
		elements := m.Index(0, MapElements).Load()
		if c.isRefCounted(keyTy) || c.isRefCounted(valTy) {
			s.ForN(capacity, func(s *S, it *codegen.Value) *codegen.Value {
				valid := elements.Index(it, "used").Load()
				s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func(s *S) {
					if c.isRefCounted(keyTy) {
						c.release(s, elements.Index(it, "k").Load(), keyTy)
					}
					if c.isRefCounted(valTy) {
						c.release(s, elements.Index(it, "v").Load(), valTy)
					}
				})
				elements.Index(it, "used").Store(s.Scalar(mapElementEmpty))
				return nil
			})
		}
		m.Index(0, MapCount).Store(s.Scalar(uint64(0)))
	})
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
