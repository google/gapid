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

// TODO: Investigate alternative rehash rule.
//     Right now it is new_capacity = capacity + 16, likely we want new_capacity = capacity << 1
// TODO: Investigate rehashing once #full + #previously_full > 80%
//     If we end up with lots of insertions/deletions, this will prevent linear search

func (c *compiler) defineMapType(t *semantic.Map) {
	mapPtrTy := c.ty.target[t].(codegen.Pointer)
	mapStrTy := mapPtrTy.Element.(*codegen.Struct)
	keyTy := c.targetType(t.KeyType)
	valTy := c.targetType(t.ValueType)
	elTy := c.ty.Struct(fmt.Sprintf("%v…%v", keyTy.TypeName(), valTy.TypeName()),
		// Used: 0 == empty, 1 == has a key, 2 == doesn't have a key, but
		//    can't assume your searched key doesn't exist
		codegen.Field{Name: "used", Type: c.targetType(semantic.Uint64Type)},
		codegen.Field{Name: "k", Type: keyTy},
		codegen.Field{Name: "v", Type: valTy},
	)
	mapStrTy.SetBody(false,
		codegen.Field{Name: mapRefCount, Type: c.ty.Uint32},
		codegen.Field{Name: mapArena, Type: c.ty.arenaPtr},
		codegen.Field{Name: mapCount, Type: c.ty.Uint64},
		codegen.Field{Name: mapCapacity, Type: c.ty.Uint64},
		codegen.Field{Name: mapElements, Type: c.ty.Pointer(elTy)},
	)
	valPtrTy := c.ty.Pointer(valTy)
	c.ty.maps[t] = &MapInfo{
		Type:     mapStrTy,
		Elements: elTy,
		Key:      keyTy,
		Val:      valTy,
		Contains: c.module.Function(c.ty.Bool, t.Name()+"_contains", mapPtrTy, keyTy),
		Index:    c.module.Function(valPtrTy, t.Name()+"_index", mapPtrTy, keyTy, c.ty.Bool),
		Lookup:   c.module.Function(valTy, t.Name()+"_lookup", mapPtrTy, keyTy),
		Remove:   c.module.Function(c.ty.Void, t.Name()+"_remove", mapPtrTy, keyTy),
		Clear:    c.module.Function(nil, t.Name()+"_clear", mapPtrTy),
	}
}

// If we know the values are going to be small & sequential, we can
// swap out this hash.
func (c *compiler) hash64Bit(s *scope, value *codegen.Value) *codegen.Value {
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

func (c *compiler) hashVariableValue(s *scope, pointer *codegen.Value, numBytes *codegen.Value) *codegen.Value {
	u64Type := c.targetType(semantic.Uint64Type)
	numBytes = numBytes.Load().Cast(u64Type)
	v := s.Local("_hash", u64Type)
	v.Store(s.Scalar(uint64(0)))
	s.ForN(numBytes, func(it *codegen.Value) *codegen.Value {
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

func (c *compiler) hashValue(s *scope, t semantic.Type, value *codegen.Value) *codegen.Value {
	keyType := c.targetType(t)
	u64Type := c.targetType(semantic.Uint64Type)
	u32Type := c.targetType(semantic.Uint32Type)
	if keyType != value.Type() {
		fail("hashValue must be called with the given type, %+v, %+v", keyType, value.Type())
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
			return c.hash64Bit(s, value.Cast(u64Type))
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
	case *semantic.Pointer,
		*semantic.Enum:
		return c.hash64Bit(s, value.Cast(u64Type))
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

func (c *compiler) buildMapType(t *semantic.Map) {
	mi, ok := c.ty.maps[t]
	if !ok {
		fail("Unknown map")
	}

	elTy := mi.Elements
	u64Type := c.targetType(semantic.Uint64Type)

	c.build(mi.Contains, func(s *scope) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		h := c.hashValue(s, t.KeyType, k)
		capacity := m.Index(0, mapCapacity).Load()
		elements := m.Index(0, mapElements).Load()
		s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
			check := s.Rem(s.Add(h, it), capacity)
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementEmpty)), func() {
				s.Return(s.Scalar(false))
			})
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func() {
				key := elements.Index(check, "k")
				found := c.equal(s, key.Load(), k)
				s.If(found, func() { s.Return(s.Scalar(true)) })
			})
			return nil
		})
		s.Return(s.Scalar(false))
	})

	f32Type := c.targetType(semantic.Float32Type)
	c.build(mi.Index, func(s *scope) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		s.arena = m.Index(0, mapArena).Load().SetName("arena")
		addIfNotFound := s.Parameter(2).SetName("addIfNotFound")

		countPtr := m.Index(0, mapCount)
		capacityPtr := m.Index(0, mapCapacity)
		elementsPtr := m.Index(0, mapElements)
		count := countPtr.Load()
		capacity := capacityPtr.Load()
		elements := elementsPtr.Load()

		h := c.hashValue(s, t.KeyType, k)
		// Search for existing
		s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
			check := s.Rem(s.Add(h, it), capacity)
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func() {
				found := c.equal(s, elements.Index(check, "k").Load(), k)
				s.If(found, func() {
					s.Return(elements.Index(check, "v"))
				})
			})

			return s.Not(c.equal(s, valid, s.Scalar(mapElementEmpty)))
		})

		s.If(addIfNotFound, func() {
			resize := s.LocalInit("resize", elements.IsNull())
			s.If(s.Not(resize.Load()), func() {
				used := s.Div(count.Cast(f32Type), capacity.Cast(f32Type))
				resize.Store(s.GreaterThan(used, s.Scalar(float32(mapMaxCapacity))))
			})

			getStorageBucket := func(h, table, tablesize *codegen.Value) *codegen.Value {
				newBucket := s.Local("newBucket", u64Type)
				s.ForN(tablesize, func(it *codegen.Value) *codegen.Value {
					check := s.Rem(s.Add(h, it), tablesize).SetName("hash_bucket")
					newBucket.Store(check)
					valid := table.Index(check, "used").Load()
					notFound := c.equal(s, valid, s.Scalar(mapElementFull))
					return notFound
				})
				return newBucket.Load()
			}

			s.If(resize.Load(), func() {
				// Grow
				s.IfElse(elements.IsNull(), func() {
					capacity := s.Scalar(uint64(minMapSize))
					capacityPtr.Store(capacity)
					elements := c.alloc(s, s.arena, capacity, elTy)
					elementsPtr.Store(elements)
					s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
						elements.Index(it, "used").Store(s.Scalar(mapElementEmpty))
						return nil
					})
				}, /* else */ func() {
					newCapacity := s.MulS(capacity, uint64(mapGrowMultiplier))
					capacityPtr.Store(newCapacity)
					newElements := c.alloc(s, s.arena, newCapacity, elTy)
					s.ForN(newCapacity, func(it *codegen.Value) *codegen.Value {
						newElements.Index(it, "used").Store(s.Scalar(mapElementEmpty))
						return nil
					})

					s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
						valid := elements.Index(it, "used").Load()
						s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func() {
							k := elements.Index(it, "k").Load()
							v := elements.Index(it, "v").Load()
							h := c.hashValue(s, t.KeyType, k)
							bucket := getStorageBucket(h, newElements, newCapacity)
							newElements.Index(bucket, "k").Store(k)
							newElements.Index(bucket, "v").Store(v)
							newElements.Index(bucket, "used").Store(s.Scalar(mapElementFull))
						})
						return nil
					})
					c.free(s, s.arena, elements)
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
			v := c.initialValue(s, t.ValueType)
			valPtr.Store(v)
			countPtr.Store(s.AddS(count, uint64(1)))

			c.reference(s, v, t.ValueType)
			c.reference(s, k, t.KeyType)

			s.Return(valPtr)
		})
	})

	c.build(mi.Lookup, func(s *scope) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		ptr := s.Call(mi.Index, m, k, s.Scalar(false))
		s.If(ptr.IsNull(), func() {
			s.Return(c.initialValue(s, t.ValueType))
		})
		v := ptr.Load()
		c.reference(s, v, t.ValueType)
		s.Return(v)
	})

	c.build(mi.Remove, func(s *scope) {
		m := s.Parameter(0).SetName("map")
		k := s.Parameter(1).SetName("key")
		s.arena = m.Index(0, mapArena).Load().SetName("arena")
		countPtr := m.Index(0, mapCount)
		capacity := m.Index(0, mapCapacity).Load()
		h := c.hashValue(s, t.KeyType, k)
		elements := m.Index(0, mapElements).Load()
		// Search for existing
		s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
			check := s.Rem(s.Add(h, it), capacity)
			valid := elements.Index(check, "used").Load()
			s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func() {
				found := c.equal(s, elements.Index(check, "k").Load(), k)
				s.If(found, func() {
					elPtr := elements.Index(check)
					// Release references to el
					if c.isRefCounted(t.KeyType) {
						c.release(s, elPtr.Index(0, "k").Load(), t.KeyType)
					}
					if c.isRefCounted(t.ValueType) {
						c.release(s, elPtr.Index(0, "v").Load(), t.ValueType)
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

	c.build(mi.Clear, func(s *scope) {
		m := s.Parameter(0).SetName("map")
		capacity := m.Index(0, mapCapacity).Load()
		elements := m.Index(0, mapElements).Load()
		if c.isRefCounted(t.KeyType) || c.isRefCounted(t.ValueType) {
			s.ForN(capacity, func(it *codegen.Value) *codegen.Value {
				valid := elements.Index(it, "used").Load()
				s.If(c.equal(s, valid, s.Scalar(mapElementFull)), func() {
					if c.isRefCounted(t.KeyType) {
						c.release(s, elements.Index(it, "k").Load(), t.KeyType)
					}
					if c.isRefCounted(t.ValueType) {
						c.release(s, elements.Index(it, "v").Load(), t.ValueType)
					}
				})
				return nil
			})
		}
		arena := m.Index(0, mapArena).Load().SetName("arena")
		c.free(s, arena, elements)
		m.Index(0, mapCount).Store(s.Scalar(uint64(0)))
		m.Index(0, mapCapacity).Store(s.Scalar(uint64(0)))
	})
}
