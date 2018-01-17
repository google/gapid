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
	c.build(f.reference, func(s *scope) {
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

	c.build(f.release, func(s *scope) {
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

var slicePrototype = &semantic.Slice{}

// declareRefRels declares all the reference type's reference() and release()
// functions.
func (c *compiler) declareRefRels() {
	r := map[semantic.Type]refRel{}
	c.refRels = r

	sli := refRel{}
	sli.declare(c, "slice", c.ty.sli)
	r[slicePrototype] = sli

	str := refRel{}
	str.declare(c, "string", c.ty.strPtr)
	r[semantic.StringType] = str

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
				r[apiTy] = sli

			default:
				if isRefTy(apiTy) {
					funcs := refRel{}
					funcs.declare(c, apiTy.Name(), cgTy)
					r[apiTy] = funcs
				}
			}
		}
	}
}

// buildRefRels implements all the reference type's reference() and release()
// functions.
func (c *compiler) buildRefRels() {
	r := c.refRels

	sli := r[slicePrototype]
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

	str := r[semantic.StringType]
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

	for apiTy, funcs := range r {
		switch apiTy {
		case semantic.StringType:
			// Already implemented

		default:
			switch apiTy := apiTy.(type) {
			case *semantic.Slice:
				// Already implemented

			case *semantic.Reference:
				funcs.build(c,
					func(s *scope, refPtr *codegen.Value) *codegen.Value {
						return refPtr.IsNull()
					},
					func(s *scope, refPtr *codegen.Value) *codegen.Value {
						return refPtr.Index(0, refRefCount)
					},
					func(s *scope, ctx, refPtr *codegen.Value) {
						c.release(s, refPtr.Index(0, refValue).Load(), apiTy.To)
						c.free(s, refPtr)
					})

			case *semantic.Map:
				funcs.build(c,
					func(s *scope, mapPtr *codegen.Value) *codegen.Value {
						return mapPtr.IsNull()
					},
					func(s *scope, mapPtr *codegen.Value) *codegen.Value {
						return mapPtr.Index(0, mapRefCount)
					},
					func(s *scope, ctx, mapPtr *codegen.Value) {
						s.Call(c.ty.maps[apiTy].Clear, s.ctx, mapPtr)
						c.free(s, mapPtr)
					})

			case *semantic.Class:
				refFields := []*semantic.Field{}
				for _, f := range apiTy.Fields {
					if _, ok := r[f.Type]; ok {
						refFields = append(refFields, f)
					}
				}

				c.build(funcs.reference, func(s *scope) {
					ptr := s.Parameter(1)
					for _, f := range refFields {
						c.reference(s, ptr.Extract(f.Name()), f.Type)
					}
				})
				c.build(funcs.release, func(s *scope) {
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
	s.onExit(func() {
		if s.IsBlockTerminated() {
			// The last instruction written to the current block was a
			// terminator instruction. This should only happen if we've emitted
			// a return statement and the scopes around this statement are
			// closing. The løogic in compiler.return_ will have already exited
			// all the contexts, so we can safely return here.
			//
			// TODO: This is really icky - more time should be spent thinking
			// of ways to avoid special casing return statements like this.
			return
		}
		c.release(s, val, ty)
	})
}

func (c *compiler) isRefCounted(ty semantic.Type) bool {
	_, ok := c.refRels[ty]
	return ok
}
