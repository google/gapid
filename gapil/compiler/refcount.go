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
	reference codegen.Function // void T_reference(T)
	release   codegen.Function // void T_release(T)
}

func (f *refRel) declare(c *C, name string, ty codegen.Type) {
	f.reference = c.M.Function(c.T.Void, name+"_reference", ty).LinkOnceODR().Inline()
	f.release = c.M.Function(c.T.Void, name+"_release", ty).LinkOnceODR().Inline()
	f.name = name
}

func (f *refRel) build(
	c *C,
	isNull func(s *S, val *codegen.Value) *codegen.Value,
	getRefPtr func(s *S, val *codegen.Value) *codegen.Value,
	del func(s *S, val *codegen.Value),
) {
	c.Build(f.reference, func(s *S) {
		val := s.Parameter(0)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.Log(s, log.Fatal, "Attempting to reference released "+f.name)
		})
		newCount := s.Add(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.Log(s, log.Info, f.name+" ref_count: %d -> %d", oldCount, newCount)
		}
		refPtr.Store(newCount)
	})

	c.Build(f.release, func(s *S) {
		val := s.Parameter(0)
		s.If(isNull(s, val), func() {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func() {
			c.Log(s, log.Fatal, "Attempting to release "+f.name+" with no remaining references!")
		})
		newCount := s.Sub(oldCount, s.Scalar(uint32(1)))
		if debugLogRefCounts {
			c.Log(s, log.Info, f.name+" ref_count: %d -> %d", oldCount, newCount)
		}
		refPtr.Store(newCount)
		s.If(s.Equal(newCount, s.Scalar(uint32(0))), func() {
			del(s, val)
		})
	})
}

type refRels map[semantic.Type]refRel

var slicePrototype = &semantic.Slice{}

// declareRefRels declares all the reference type's reference() and release()
// functions.
func (c *C) declareRefRels() {
	r := map[semantic.Type]refRel{}
	c.refRels = r

	sli := refRel{}
	sli.declare(c, "slice", c.T.Sli)
	r[slicePrototype] = sli

	str := refRel{}
	str.declare(c, "string", c.T.StrPtr)
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
	for apiTy, cgTy := range c.T.target {
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
func (c *C) buildRefRels() {
	r := c.refRels

	sli := r[slicePrototype]
	sli.build(c,
		func(s *S, sli *codegen.Value) *codegen.Value {
			poolPtr := sli.Extract(SlicePool)
			return s.Equal(poolPtr, s.Zero(poolPtr.Type()))
		},
		func(s *S, sli *codegen.Value) *codegen.Value {
			poolPtr := sli.Extract(SlicePool)
			return poolPtr.Index(0, PoolRefCount)
		},
		func(s *S, sli *codegen.Value) {
			poolPtr := sli.Extract(SlicePool)
			s.Call(c.callbacks.freePool, poolPtr)
		})

	str := r[semantic.StringType]
	str.build(c,
		func(s *S, strPtr *codegen.Value) *codegen.Value {
			return s.Equal(strPtr, s.Zero(c.T.StrPtr))
		},
		func(s *S, strPtr *codegen.Value) *codegen.Value {
			return strPtr.Index(0, StringRefCount)
		},
		func(s *S, strPtr *codegen.Value) {
			s.Call(c.callbacks.freeString, strPtr)
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
					func(s *S, refPtr *codegen.Value) *codegen.Value {
						return refPtr.IsNull()
					},
					func(s *S, refPtr *codegen.Value) *codegen.Value {
						return refPtr.Index(0, RefRefCount)
					},
					func(s *S, refPtr *codegen.Value) {
						s.Arena = refPtr.Index(0, RefArena).Load().SetName("arena")
						c.release(s, refPtr.Index(0, RefValue).Load(), apiTy.To)
						c.Free(s, refPtr)
					})

			case *semantic.Map:
				funcs.build(c,
					func(s *S, mapPtr *codegen.Value) *codegen.Value {
						return mapPtr.IsNull()
					},
					func(s *S, mapPtr *codegen.Value) *codegen.Value {
						return mapPtr.Index(0, MapRefCount)
					},
					func(s *S, mapPtr *codegen.Value) {
						s.Arena = mapPtr.Index(0, MapArena).Load().SetName("arena")
						s.Call(c.T.Maps[apiTy].Clear, mapPtr)
						c.Free(s, mapPtr)
					})

			case *semantic.Class:
				refFields := []*semantic.Field{}
				for _, f := range apiTy.Fields {
					if _, ok := r[f.Type]; ok {
						refFields = append(refFields, f)
					}
				}

				c.Build(funcs.reference, func(s *S) {
					ptr := s.Parameter(0)
					for _, f := range refFields {
						c.reference(s, ptr.Extract(f.Name()), f.Type)
					}
				})
				c.Build(funcs.release, func(s *S) {
					ptr := s.Parameter(0)
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

func (c *C) reference(s *S, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[semantic.Underlying(ty)]; ok {
		s.Call(f.reference, val)
	}
}

func (c *C) release(s *S, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels[semantic.Underlying(ty)]; ok {
		s.Call(f.release, val)
	}
}

func (c *C) deferRelease(s *S, val *codegen.Value, ty semantic.Type) {
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

func (c *C) isRefCounted(ty semantic.Type) bool {
	_, ok := c.refRels[ty]
	return ok
}
