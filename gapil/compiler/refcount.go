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
	"runtime"
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/semantic"
)

type refRel struct {
	name      string
	reference *codegen.Function // void T_reference(T)
	release   *codegen.Function // void T_release(T)
}

func (f *refRel) declare(c *C, name, ref, rel string, ty codegen.Type) {
	f.reference = c.M.Function(c.T.Void, ref, ty).LinkOnceODR().Inline()
	f.release = c.M.Function(c.T.Void, rel, ty).LinkOnceODR().Inline()
	f.name = name
}

func (f *refRel) delegate(c *C, to refRel) {
	c.Delegate(f.reference, to.reference)
	c.Delegate(f.release, to.release)
}

func (f *refRel) build(
	c *C,
	isNull func(s *S, val *codegen.Value) *codegen.Value,
	getRefPtr func(s *S, val *codegen.Value) *codegen.Value,
	del func(s *S, val *codegen.Value),
) {
	c.Build(f.reference, func(s *S) {
		val := s.Parameter(0)
		s.If(isNull(s, val), func(s *S) {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func(s *S) {
			c.Log(s, log.Fatal, "Attempting to reference released "+f.name+" (%p)", refPtr)
		})
		newCount := s.Add(oldCount, s.Scalar(uint32(1)))
		if debugRefCounts {
			c.LogI(s, f.name+" %p ref_count: %d -> %d", refPtr, oldCount, newCount)
		}
		refPtr.Store(newCount)
	})

	c.Build(f.release, func(s *S) {
		val := s.Parameter(0)
		s.If(isNull(s, val), func(s *S) {
			s.Return(nil)
		})
		refPtr := getRefPtr(s, val)
		oldCount := refPtr.Load()
		s.If(s.Equal(oldCount, s.Scalar(uint32(0))), func(s *S) {
			c.Log(s, log.Fatal, "Attempting to release "+f.name+" with no remaining references! (%p)", refPtr)
		})
		newCount := s.Sub(oldCount, s.Scalar(uint32(1)))
		if debugRefCounts {
			c.LogI(s, f.name+" %p ref_count: %d -> %d", refPtr, oldCount, newCount)
		}
		refPtr.Store(newCount)
		s.If(s.Equal(newCount, s.Scalar(uint32(0))), func(s *S) {
			del(s, val)
		})
	})
}

type refRels struct {
	tys   map[semantic.Type]refRel // Delegate on to impls
	impls map[semantic.Type]refRel // Implementations of lowered map types
}

var slicePrototype = &semantic.Slice{}

// declareRefRels declares all the reference type's reference() and release()
// functions.
func (c *C) declareRefRels() {
	c.refRels = refRels{
		tys:   map[semantic.Type]refRel{},
		impls: map[semantic.Type]refRel{},
	}

	sli := refRel{}
	sli.declare(c, "slice", "gapil_slice_reference", "gapil_slice_release", c.T.Sli)
	c.refRels.tys[slicePrototype] = sli
	c.refRels.impls[slicePrototype] = sli

	str := refRel{}
	str.declare(c, "string", "gapil_string_reference", "gapil_string_release", c.T.StrPtr)
	c.refRels.tys[semantic.StringType] = str
	c.refRels.impls[semantic.StringType] = str

	any := refRel{name: "any", reference: c.callbacks.anyReference, release: c.callbacks.anyRelease}
	c.refRels.tys[semantic.AnyType] = any
	c.refRels.impls[semantic.AnyType] = any

	msg := refRel{name: "message", reference: c.callbacks.msgReference, release: c.callbacks.msgRelease}
	c.refRels.tys[semantic.MessageType] = msg
	c.refRels.impls[semantic.MessageType] = msg

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

	// impls is a map of type mangled type name to the public reference and
	// release functions.
	// This is used to deduplicate types that have the same underlying key and
	// value LLVM types when lowered.
	impls := map[string]refRel{}

	for _, api := range c.APIs {
		declare := func(apiTy semantic.Type) {
			cgTy := c.T.Target(apiTy)
			apiTy = semantic.Underlying(apiTy)
			switch apiTy {
			case semantic.StringType:
				// Already implemented
			default:
				switch apiTy := apiTy.(type) {
				case *semantic.Slice:
					c.refRels.tys[apiTy] = sli

				default:
					if isRefTy(apiTy) {
						name := fmt.Sprintf("%v_%v", api.Name(), apiTy.Name())

						// Use the mangled name of the type to determine whether
						// the reference and release functions have already been
						// declared for the lowered type.
						m := c.Mangle(cgTy)
						mangled := c.Mangler(m)
						impl, seen := impls[mangled]
						if !seen {
							// First instance of this lowered type. Declare it.
							ref := c.Mangler(&mangling.Function{
								Name:       "reference",
								Parent:     m.(mangling.Scope),
								Parameters: []mangling.Type{m},
							})
							rel := c.Mangler(&mangling.Function{
								Name:       "release",
								Parent:     m.(mangling.Scope),
								Parameters: []mangling.Type{m},
							})
							impl.declare(c, name, ref, rel, cgTy)
							impls[mangled] = impl
							c.refRels.impls[apiTy] = impl
						}

						// Delegate the reference and release functions of this type
						// on to the common implementation.
						funcs := refRel{}
						funcs.declare(c, name, name+"_reference", name+"_release", cgTy)
						funcs.delegate(c, impl)
						c.refRels.tys[apiTy] = funcs
					}
				}
			}
		}
		for _, ty := range api.Slices {
			declare(ty)
		}
		for _, ty := range api.Maps {
			declare(ty)
		}
		for _, ty := range api.References {
			declare(ty)
		}
		for _, ty := range api.Classes {
			declare(ty)
		}
	}
}

// buildRefRels implements all the reference type's reference() and release()
// functions.
func (c *C) buildRefRels() {
	r := c.refRels.impls

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

	for _, api := range c.APIs {
		for _, apiTy := range api.Maps {
			if funcs, ok := r[apiTy]; ok {
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
			}
		}

		for _, apiTy := range api.References {
			if funcs, ok := r[apiTy]; ok {
				funcs.build(c,
					func(s *S, refPtr *codegen.Value) *codegen.Value {
						return refPtr.IsNull()
					},
					func(s *S, refPtr *codegen.Value) *codegen.Value {
						return refPtr.Index(0, RefRefCount)
					},
					func(s *S, refPtr *codegen.Value) {
						s.Arena = refPtr.Index(0, RefArena).Load().SetName("arena")
						c.doRelease(s, refPtr.Index(0, RefValue).Load(), apiTy.To)
						c.Free(s, refPtr)
					})
			}
		}

		for _, apiTy := range api.Classes {
			if funcs, ok := r[apiTy]; ok {
				refFields := []*semantic.Field{}
				for _, f := range apiTy.Fields {
					ty := semantic.Underlying(f.Type)
					if _, ok := c.refRels.tys[ty]; ok {
						refFields = append(refFields, f)
					}
				}

				c.Build(funcs.reference, func(s *S) {
					val := s.Parameter(0)
					for _, f := range refFields {
						c.doReference(s, val.Extract(f.Name()), f.Type)
					}
				})
				c.Build(funcs.release, func(s *S) {
					val := s.Parameter(0)
					for _, f := range refFields {
						c.doRelease(s, val.Extract(f.Name()), f.Type)
					}
				})
			}
		}
	}
}

func caller() string {
	locs := []string{}
	for i := 2; i < 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if i := strings.LastIndex(file, "/"); i > 0 {
			file = file[i+1:]
		}
		locs = append(locs, fmt.Sprintf("%v:%v", file, line))
	}
	return strings.Join(locs, " ")
}

func (c *C) reference(s *S, val *codegen.Value, ty semantic.Type) {
	if got, expect := val.Type(), c.T.Target(ty); got != expect {
		fail("reference() called with a value of an unexpected type. Got %+v, expect %+v", got, expect)
	}
	if !c.isRefCounted(semantic.Underlying(ty)) {
		return
	}
	if debugRefCounts {
		c.LogI(s, fmt.Sprintf("reference(%v: %%v): %v", ty, caller()), val)
	}
	if debugDisableRefCounts {
		return
	}
	if debugDisableRefCountOpts {
		c.doReference(s, val, ty)
	} else {
		s.pendingRefRels.add(c, val, ty, 1)
	}
}

func (c *C) release(s *S, val *codegen.Value, ty semantic.Type) {
	if got, expect := val.Type(), c.T.Target(ty); got != expect {
		fail("release() called with a value of an unexpected type. Got %+v, expect %+v", got, expect)
	}
	if !c.isRefCounted(semantic.Underlying(ty)) {
		return
	}
	if debugRefCounts {
		c.LogI(s, fmt.Sprintf("release(%v: %%v): %v", ty, caller()), val)
	}
	if debugDisableRefCounts {
		return
	}
	if debugDisableRefCountOpts {
		c.doRelease(s, val, ty)
	} else {
		s.pendingRefRels.add(c, val, ty, -1)
	}
}

func (c *C) deferRelease(s *S, val *codegen.Value, ty semantic.Type) {
	if got, expect := val.Type(), c.T.Target(ty); got != expect {
		fail("deferRelease() called with a value of an unexpected type. Got %+v, expect %+v", got, expect)
	}
	if !c.isRefCounted(semantic.Underlying(ty)) {
		return
	}
	if debugRefCounts {
		c.LogI(s, fmt.Sprintf("deferRelease(%v: %%v): %v", ty, caller()), val)
	}
	if debugDisableRefCounts {
		return
	}
	s.pendingRefRels.add(c, val, ty, -1)
}

type pendingRefRels struct {
	c       *C
	indices map[*codegen.Value]int
	list    []pendingRefRel
}

// pendingRefRel is a pending reference count change when the scope closes.
type pendingRefRel struct {
	val    *codegen.Value
	ty     semantic.Type
	delta  int
	caller string
}

func (p *pendingRefRels) add(c *C, val *codegen.Value, ty semantic.Type, delta int) {
	if !c.isRefCounted(ty) {
		return
	}
	if p.c == nil {
		*p = pendingRefRels{
			c:       c,
			indices: map[*codegen.Value]int{},
		}
	}
	if i, ok := p.indices[val]; ok {
		p.list[i].delta = p.list[i].delta + delta
	} else {
		p.indices[val] = len(p.list)
		r := pendingRefRel{
			val:   val,
			ty:    ty,
			delta: delta,
		}
		if debugRefCounts {
			r.caller = caller()
		}
		p.list = append(p.list, r)
	}
}

func (p *pendingRefRels) apply(s *S) {
	if p.c == nil {
		return
	}
	if debugRefCounts {
		p.c.LogI(s, "vvvvvvvvvvvvvvvvvvvvvv")
		p.c.LogI(s, "pendingRefRels.apply()")
		for _, r := range p.list {
			if r.delta != 0 {
				p.c.LogI(s, fmt.Sprintf("%+d %v: %%v", r.delta, r.ty), r.val)
			}
		}
		p.c.LogI(s, "^^^^^^^^^^^^^^^^^^^^^^")
	}

	for _, r := range p.list {
		for i := 0; i < r.delta; i++ {
			if f, ok := p.c.refRels.tys[semantic.Underlying(r.ty)]; ok {
				if debugRefCounts {
					p.c.LogI(s, fmt.Sprintf("pending-reference(%v: %%v): %v", r.ty, r.caller), r.val)
				}
				s.Call(f.reference, r.val)
			}
		}
	}
	for _, r := range p.list {
		for i := 0; i > r.delta; i-- {
			if f, ok := p.c.refRels.tys[semantic.Underlying(r.ty)]; ok {
				if debugRefCounts {
					p.c.LogI(s, fmt.Sprintf("pending-release(%v: %%v): %v", r.ty, r.caller), r.val)
				}
				s.Call(f.release, r.val)
			}
		}
	}
}

func (c *C) doReference(s *S, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels.tys[semantic.Underlying(ty)]; ok {
		if debugRefCounts {
			c.LogI(s, fmt.Sprintf("doReference(%v: %%v): %v", ty, caller()), val)
		}
		s.Call(f.reference, val)
	}
}

func (c *C) doRelease(s *S, val *codegen.Value, ty semantic.Type) {
	if f, ok := c.refRels.tys[semantic.Underlying(ty)]; ok {
		if debugRefCounts {
			c.LogI(s, fmt.Sprintf("doRelease(%v: %%v): %v", ty, caller()), val)
		}
		s.Call(f.release, val)
	}
}

func (c *C) isRefCounted(ty semantic.Type) bool {
	_, ok := c.refRels.tys[semantic.Underlying(ty)]
	return ok
}
