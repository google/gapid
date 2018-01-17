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
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/semantic"
)

const debugLogRefCounts = false

type refRel struct {
	name      string
	reference codegen.Function // void T_reference(context*, T)
	release   codegen.Function // void T_release(context*, T)
}

func (f *refRel) declare(c *compiler, name string, ty codegen.Type) {
	m := c.module
	f.reference = m.Function(c.ty.Void, name+"_reference", c.ty.ctxPtr, ty)
	f.release = m.Function(c.ty.Void, name+"_release", c.ty.ctxPtr, ty)
	f.name = name
}

func (f *refRel) build(
	c *compiler,
	isNull func(s *scope, val *codegen.Value) *codegen.Value,
	getRefPtr func(s *scope, val *codegen.Value) *codegen.Value,
	del func(s *scope, ctx, val *codegen.Value),
) {
	f.reference.Build(func(b *codegen.Builder) {
		s := c.scope(b)
		val := s.Parameter(1)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.logf(s, log.Fatal, "Attempting to reference released "+f.name)
		})
		newCount := s.Add(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.logf(s, log.Info, f.name+" ref_count: %d -> %d", oldCount, newCount)
		}
		refPtr.Store(newCount)
	})

	f.release.Build(func(b *codegen.Builder) {
		s := c.scope(b)
		ctx, val := s.Parameter(0), s.Parameter(1)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.logf(s, log.Fatal, "Attempting to release "+f.name+" with no remaining references!")
		})
		newCount := s.Sub(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.logf(s, log.Info, f.name+" ref_count: %d -> %d", oldCount, newCount)
		}
		refPtr.Store(newCount)
		s.If(s.Equal(newCount, s.Scalar(uint32(0))), func() {
			del(s, ctx, val)
		})
	})
}

type refRels map[semantic.Type]refRel

func (r *refRels) build(c *compiler) {
	*r = map[semantic.Type]refRel{}

	sli := refRel{}
	sli.declare(c, "slice", c.ty.sli)
	sli.build(c,
		func(s *scope, sli *codegen.Value) *codegen.Value {
			poolPtr := sli.Extract(slicePool)
			return s.Equal(poolPtr, s.Zero(poolPtr.Type()))
		},
		func(s *scope, sli *codegen.Value) *codegen.Value {
			poolPtr := sli.Extract(slicePool)
			return poolPtr.Index(0, poolRefCount)
		},
		func(s *scope, ctx, sli *codegen.Value) {
			poolPtr := sli.Extract(slicePool)
			s.Call(c.callbacks.freePool, ctx, poolPtr)
		})

	str := refRel{}
	str.declare(c, "string", c.ty.strPtr)
	str.build(c,
		func(s *scope, strPtr *codegen.Value) *codegen.Value {
			return s.Equal(strPtr, s.Zero(c.ty.strPtr))
		},
		func(s *scope, strPtr *codegen.Value) *codegen.Value {
			return strPtr.Index(0, stringRefCount)
		},
		func(s *scope, ctx, strPtr *codegen.Value) {
			s.Call(c.callbacks.freeString, ctx, strPtr)
		})
	(*r)[semantic.StringType] = str

	var isRefTy func(ty semantic.Type) bool
	isRefTy = func(ty semantic.Type) bool {
		ty = semantic.Underlying(ty)
		if ty == semantic.StringType {
			return true
		}
		switch ty := ty.(type) {
		case *semantic.Slice, *semantic.Reference, *semantic.Map:
			return true
		case *semantic.Class:
			for _, f := range ty.Fields {
				if isRefTy(f.Type) {
					return true
				}
			}
		}
		return false
	}

	// Forward declare all the reference types.
	for apiTy, cgTy := range c.ty.target {
		apiTy = semantic.Underlying(apiTy)
		switch apiTy {
		case semantic.StringType:
			// Already implemented

		default:
			switch apiTy := apiTy.(type) {
			case *semantic.Slice:
				(*r)[apiTy] = sli

			default:
				if isRefTy(apiTy) {
					funcs := refRel{}
					funcs.declare(c, apiTy.Name(), cgTy)
					(*r)[apiTy] = funcs
				}
			}
		}
	}

	// Implement all the reference types.
	for apiTy, funcs := range *r {
		switch apiTy {
		case semantic.StringType:
			// Already implemented

		default:
			switch apiTy := apiTy.(type) {
			case *semantic.Slice:
				// Already implemented

			case *semantic.Reference:
				cgTy := c.ty.target[apiTy]
				funcs.build(c,
					func(s *scope, refPtr *codegen.Value) *codegen.Value {
						return s.Equal(refPtr, s.Zero(cgTy))
					},
					func(s *scope, refPtr *codegen.Value) *codegen.Value {
						return refPtr.Index(0, refRefCount)
					},
					func(s *scope, ctx, refPtr *codegen.Value) {
						c.release(s, refPtr.Index(0, refValue).Load(), apiTy.To)
						s.Call(c.callbacks.free, ctx, refPtr.Cast(c.ty.voidPtr))
					})

			case *semantic.Map:
				cgTy := c.ty.target[apiTy]
				funcs.build(c,
					func(s *scope, mapPtr *codegen.Value) *codegen.Value {
						return s.Equal(mapPtr, s.Zero(cgTy))
					},
					func(s *scope, mapPtr *codegen.Value) *codegen.Value {
						return mapPtr.Index(0, mapRefCount)
					},
					func(s *scope, ctx, mapPtr *codegen.Value) {
						elPtr := mapPtr.Index(0, mapElements).Load()
						s.Call(c.callbacks.free, ctx, elPtr.Cast(c.ty.voidPtr))
						s.Call(c.callbacks.free, ctx, mapPtr.Cast(c.ty.voidPtr))
					})

			case *semantic.Class:
				refFields := []*semantic.Field{}
				for _, f := range apiTy.Fields {
					if _, ok := (*r)[f.Type]; ok {
						refFields = append(refFields, f)
					}
				}

				funcs.reference.Build(func(b *codegen.Builder) {
					s := c.scope(b)
					ptr := s.Parameter(1)
					for _, f := range refFields {
						c.reference(s, ptr.Extract(f.Name()), f.Type)
					}
				})
				funcs.release.Build(func(b *codegen.Builder) {
					s := c.scope(b)
					ptr := s.Parameter(1)
					for _, f := range refFields {
						c.release(s, ptr.Extract(f.Name()), f.Type)
					}
				})
			default:
				fail("Unhandled reference type %T", apiTy)
			}
		}
	}
}

func (c *compiler) reference(s *scope, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[semantic.Underlying(ty)]; ok {
		s.Call(f.reference, s.ctx, val)
	}
}

func (c *compiler) release(s *scope, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[semantic.Underlying(ty)]; ok {
		s.Call(f.release, s.ctx, val)
	}
}

func (c *compiler) deferRelease(s *scope, val *codegen.Value, ty semantic.Type) {
	s.Defer(func() { c.release(s, val, ty) })
}

func (c *compiler) isRefCounted(ty semantic.Type) bool {
	_, ok := c.refRels[ty]
	return ok
}
