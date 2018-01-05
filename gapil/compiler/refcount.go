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
	reference codegen.Function
	release   codegen.Function
}

func (f *refRel) build(
	c *compiler,
	name string,
	ty codegen.Type,
	isNull func(s *scope, val *codegen.Value) *codegen.Value,
	getRefPtr func(s *scope, val *codegen.Value) *codegen.Value,
	del func(s *scope, ctx, val *codegen.Value),
) {
	m := c.module
	// void T_reference(context*, T)
	f.reference = m.Function(c.ty.Void, name+"_reference", c.ty.ctxPtr, ty)
	f.reference.Build(func(b *codegen.Builder) {
		s := c.scope(b)
		val := s.Parameter(1)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.logf(s, log.Fatal, "Attempting to reference released "+name)
		})
		newCount := s.Add(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.logf(s, log.Info, name+" ref_count: %d -> %d", oldCount, newCount)
		}
		refPtr.Store(newCount)
	})

	// void T_release(context*, T)
	f.release = m.Function(c.ty.Void, name+"_release", c.ty.ctxPtr, ty)
	f.release.Build(func(b *codegen.Builder) {
		s := c.scope(b)
		ctx, val := s.Parameter(0), s.Parameter(1)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.logf(s, log.Fatal, "Attempting to release "+name+" with no remaining references!")
		})
		newCount := s.Sub(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.logf(s, log.Info, name+" ref_count: %d -> %d", oldCount, newCount)
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
	sli.build(c, "slice", c.ty.sli,
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
	str.build(c, "string", c.ty.strPtr,
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

	for apiTy, cgTy := range c.ty.target {
		switch apiTy := apiTy.(type) {
		case *semantic.Slice:
			(*r)[apiTy] = sli

		case *semantic.Reference:
			funcs := refRel{}
			funcs.build(c, apiTy.Name(), cgTy,
				func(s *scope, refPtr *codegen.Value) *codegen.Value {
					return s.Equal(refPtr, s.Zero(cgTy))
				},
				func(s *scope, refPtr *codegen.Value) *codegen.Value {
					return refPtr.Index(0, refRefCount)
				},
				func(s *scope, ctx, refPtr *codegen.Value) {
					c.release(s, refPtr.Index(0, refValue), apiTy.To)
					s.Call(c.callbacks.free, ctx, refPtr.Cast(c.ty.voidPtr))
				})
			(*r)[apiTy] = funcs

		case *semantic.Map:
			funcs := refRel{}
			funcs.build(c, apiTy.Name(), cgTy,
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
			(*r)[apiTy] = funcs
		}
	}

	for apiTy, cgTy := range c.ty.target {
		switch apiTy := semantic.Underlying(apiTy).(type) {
		case *semantic.Class: // Must come after all other types are constructed.
			refFields := []*semantic.Field{}
			for _, f := range apiTy.Fields {
				if _, ok := (*r)[f.Type]; ok { // TODO: This needs to be split into two passes to handle structs in structs.
					refFields = append(refFields, f)
				}
			}
			if len(refFields) == 0 {
				continue
			}

			name := apiTy.Name()
			funcs := refRel{}

			// void T_reference(context*, T)
			funcs.reference = c.module.Function(c.ty.Void, name+"_reference", c.ty.ctxPtr, cgTy)
			funcs.reference.Build(func(b *codegen.Builder) {
				s := c.scope(b)
				ptr := s.Parameter(1)
				for _, f := range refFields {
					c.reference(s, ptr.Extract(f.Name()), f.Type)
				}
			})

			// void T_release(context*, T)
			funcs.release = c.module.Function(c.ty.Void, name+"_release", c.ty.ctxPtr, cgTy)
			funcs.release.Build(func(b *codegen.Builder) {
				s := c.scope(b)
				ptr := s.Parameter(1)
				for _, f := range refFields {
					c.release(s, ptr.Extract(f.Name()), f.Type)
				}
			})

			(*r)[apiTy] = funcs
		}
	}
}

func (c *compiler) reference(s *scope, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[ty]; ok {
		s.Call(f.reference, s.ctx, val)
	}
}

func (c *compiler) release(s *scope, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[ty]; ok {
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
