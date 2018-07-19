// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the Licensc.
// You may obtain a copy of the License at
//
//      http://www.apachc.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the Licensc.

// Package cloner is a plugin for the gapil compiler to generate deep clone
// functions for reference types, maps and commands.
package cloner

import (
	"fmt"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

// cloner is the compiler plugin that adds cloning functionality.
type cloner struct {
	*compiler.C
	clonableTys []semantic.Type
	clone       map[semantic.Type]*codegen.Function
	cloneImpls  map[semantic.Type]*codegen.Function
	callbacks   callbacks
}

// Build implements the compiler.Plugin interfacc.
func (c *cloner) Build(compiler *compiler.C) {
	*c = cloner{
		C:          compiler,
		clone:      map[semantic.Type]*codegen.Function{},
		cloneImpls: map[semantic.Type]*codegen.Function{},
	}

	for _, api := range c.APIs {
		for _, ty := range api.References {
			c.clonableTys = append(c.clonableTys, ty)
		}
		for _, ty := range api.Maps {
			c.clonableTys = append(c.clonableTys, ty)
		}
	}

	c.parseCallbacks()
	c.declareClones()
	c.implementClones()
}

// declareClones declares all the clone functions for all the clonable types.
func (c *cloner) declareClones() {
	// impls is a map of type mangled name to the public clone function.
	// This is used to deduplicate clone functions that have the same underlying
	// LLVM types when lowered.
	impls := map[string]*codegen.Function{}

	for _, ty := range c.clonableTys {
		ptrTy := c.T.Target(ty).(codegen.Pointer)
		elTy := ptrTy.Element

		// Use the mangled name to determine whether the clone function has
		// already been declared for this lowered type.
		mangled := c.Mangler(c.Mangle(elTy))
		impl, seen := impls[mangled]
		if !seen {
			impl = c.Method(false, elTy, ptrTy, "clone", c.T.ArenaPtr, c.T.VoidPtr).
				LinkPrivate().
				LinkOnceODR()
			impls[mangled] = impl
			c.cloneImpls[ty] = impl
		}

		// Delegate the clone method of this type on to the common implmentation.
		f := c.M.Function(ptrTy, ty.Name()+"_clone", ptrTy, c.T.ArenaPtr, c.T.VoidPtr).
			LinkPrivate().
			LinkOnceODR().
			Inline()
		c.Delegate(f, impl)
		c.clone[ty] = f
	}
}

// implementClones implements all the private clone functions, and all the
// public clone functions.
func (c *cloner) implementClones() {
	for ty, f := range c.cloneImpls {
		switch ty := ty.(type) {
		case *semantic.Reference:
			c.C.Build(f, func(s *compiler.S) {
				this, arena, tracker := s.Parameter(0), s.Parameter(1), s.Parameter(2)
				s.Arena = arena

				refPtrTy := this.Type().(codegen.Pointer)
				refTy := refPtrTy.Element

				s.IfElse(this.IsNull(), func(s *compiler.S) {
					s.Return(s.Zero(refPtrTy))
				}, func(s *compiler.S) {
					existing := s.Call(c.callbacks.cloneTrackerLookup, tracker, this.Cast(c.T.VoidPtr)).Cast(refPtrTy)
					s.IfElse(existing.IsNull(), func(s *compiler.S) {
						clone := c.Alloc(s, s.Scalar(uint64(1)), refTy)
						s.Call(c.callbacks.cloneTrackerTrack, tracker, this.Cast(c.T.VoidPtr), clone.Cast(c.T.VoidPtr))
						clone.Index(0, compiler.RefRefCount).Store(s.Scalar(uint32(1)))
						clone.Index(0, compiler.RefArena).Store(s.Arena)
						c.cloneTo(s, ty.To, clone.Index(0, compiler.RefValue), this.Index(0, compiler.RefValue).Load(), tracker)
						s.Return(clone)
					}, func(s *compiler.S) {
						s.Return(existing)
					})
				})
			})
		case *semantic.Map:
			c.C.Build(f, func(s *compiler.S) {
				this, arena, tracker := s.Parameter(0), s.Parameter(1), s.Parameter(2)
				s.Arena = arena

				mapPtrTy := this.Type().(codegen.Pointer)

				s.IfElse(this.IsNull(), func(s *compiler.S) {
					s.Return(s.Zero(mapPtrTy))
				}, func(s *compiler.S) {
					existing := s.Call(c.callbacks.cloneTrackerLookup, tracker, this.Cast(c.T.VoidPtr)).Cast(mapPtrTy)
					s.IfElse(existing.IsNull(), func(s *compiler.S) {
						mapInfo := c.T.Maps[ty]
						clone := c.Alloc(s, s.Scalar(uint64(1)), mapInfo.Type)
						s.Call(c.callbacks.cloneTrackerTrack, tracker, this.Cast(c.T.VoidPtr), clone.Cast(c.T.VoidPtr))
						clone.Index(0, compiler.MapRefCount).Store(s.Scalar(uint32(1)))
						clone.Index(0, compiler.MapArena).Store(s.Arena)
						clone.Index(0, compiler.MapCount).Store(s.Scalar(uint64(0)))
						clone.Index(0, compiler.MapCapacity).Store(s.Scalar(uint64(0)))
						clone.Index(0, compiler.MapElements).Store(s.Zero(c.T.Pointer(mapInfo.Elements)))
						c.IterateMap(s, this, semantic.Uint64Type, func(i, k, v *codegen.Value) {
							dstK, srcK := s.Local("key", mapInfo.Key), k.Load()
							c.cloneTo(s, ty.KeyType, dstK, srcK, tracker)
							dstV, srcV := s.Call(mapInfo.Index, clone, dstK.Load(), s.Scalar(true)), v.Load()
							c.cloneTo(s, ty.ValueType, dstV, srcV, tracker)
						})
						s.Return(clone)
					}, func(s *compiler.S) {
						s.Return(existing)
					})
				})
			})
		default:
			c.Fail("Unhandled type: %v", ty.Name())
		}
	}

	for _, api := range c.APIs {
		for _, cmd := range api.Functions {
			params := c.T.CmdParams[cmd]
			paramsPtr := c.T.Pointer(params)
			f := c.M.Function(paramsPtr, cmd.Name()+"__clone", paramsPtr, c.T.ArenaPtr, c.T.VoidPtr).LinkOnceODR()
			c.C.Build(f, func(s *compiler.S) {
				this, arena, tracker := s.Parameter(0), s.Parameter(1), s.Parameter(2)
				s.Arena = arena
				clone := c.Alloc(s, s.Scalar(1), params)
				thread := semantic.BuiltinThreadGlobal.Name()
				c.cloneTo(s, semantic.Uint64Type, clone.Index(0, thread), this.Index(0, thread).Load(), tracker)
				for _, p := range cmd.FullParameters {
					c.cloneTo(s, p.Type, clone.Index(0, p.Name()), this.Index(0, p.Name()).Load(), tracker)
				}
				s.Return(clone)
			})
		}
	}
}

// cloneTo emits the logic to clone the value src to the pointer dst.
func (c *cloner) cloneTo(s *compiler.S, ty semantic.Type, dst, src, tracker *codegen.Value) {
	if f, ok := c.clone[ty]; ok {
		dst.Store(s.Call(f, src, s.Arena, tracker))
		return
	}

	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Pseudonym:
		c.cloneTo(s, ty.To, dst, src, tracker)
	case *semantic.Builtin:
		switch ty {
		case semantic.Int8Type,
			semantic.Int16Type,
			semantic.Int32Type,
			semantic.Int64Type,
			semantic.IntType,
			semantic.Uint8Type,
			semantic.Uint16Type,
			semantic.Uint32Type,
			semantic.Uint64Type,
			semantic.UintType,
			semantic.CharType,
			semantic.SizeType,
			semantic.BoolType,
			semantic.Float32Type,
			semantic.Float64Type:
			dst.Store(src)

		case semantic.StringType:
			existing := s.Call(c.callbacks.cloneTrackerLookup, tracker, src.Cast(c.T.VoidPtr)).Cast(c.T.StrPtr)
			s.IfElse(existing.IsNull(), func(s *compiler.S) {
				l := src.Index(0, compiler.StringLength).Load()
				d := src.Index(0, compiler.StringData, 0)
				clone := c.MakeString(s, l, d)
				s.Call(c.callbacks.cloneTrackerTrack, tracker, src.Cast(c.T.VoidPtr), clone.Cast(c.T.VoidPtr))
				dst.Store(clone)
			}, func(s *compiler.S) {
				dst.Store(existing)
			})

		default:
			panic(fmt.Errorf("cloneTo not implemented for builtin type %v", ty))
		}
	case *semantic.Enum:
		dst.Store(src)
	case *semantic.Class:
		for _, f := range ty.Fields {
			dst, src := dst.Index(0, f.Name()), src.Extract(f.Name())
			c.cloneTo(s, f.Type, dst, src, tracker)
		}
	case *semantic.Slice:
		// TODO: Attempting to clone a slice requires a context, which we
		// currently do not have. Weak-copy for now.
		dst.Store(src)

		// size := src.Extract(compiler.SliceSize)
		// c.MakeSliceAt(s, size, dst)
		// c.CopySlice(s, dst, src)

	case *semantic.StaticArray:
		for i := 0; i < int(ty.Size); i++ {
			// TODO: Be careful of large arrays!
			c.cloneTo(s, ty.ValueType, dst.Index(0, i), src.Extract(i), tracker)
		}

	case *semantic.Pointer:
		dst.Store(src)

	default:
		panic(fmt.Errorf("cloneTo not implemented for type %v", ty))
	}
}
