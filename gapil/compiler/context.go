// Copyright (C) 2018 Google Inc.
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

import "github.com/google/gapid/core/codegen"

//#include "gapil/runtime/cc/runtime.h"
import "C"

func (c *C) declareContextType() {
	fields := c.T.FieldsOf(C.context{})

	// Append all the plugin context fields.
	c.plugins.foreach(func(p ContextDataPlugin) {
			customFields := p.ContextData(c)
			for _, f := range customFields {
				fields = append(fields, codegen.Field{
					Name: f.Name,
					Type: f.Type,
				})
			}
			c.T.customCtxFields = append(c.T.customCtxFields, customFields...)
	})

	c.T.Ctx = c.T.Struct("context", fields...)
	c.T.CtxPtr = c.T.Pointer(c.T.Ctx)
}

func (c *C) buildContextFuncs() {
	if !c.settings.EmitContext {
		return
	}

	c.ctx.create = c.M.Function(c.T.CtxPtr, "gapil_create_context", c.T.ArenaPtr)
	c.ctx.destroy = c.M.Function(c.T.Void, "gapil_destroy_context", c.T.CtxPtr)

	c.Build(c.ctx.create, func(s *S) {
		s.Arena = s.Parameter(0)

		ctx := c.Alloc(s, s.Scalar(1), c.T.Ctx).SetName("ctx")
		s.Memzero(ctx.Cast(c.T.VoidPtr), s.SizeOf(c.T.Ctx).Cast(c.T.Uint32))

		nextPoolID := c.Alloc(s, s.Scalar(1), c.T.Uint32).SetName("next_pool_id")
		nextPoolID.Store(s.Scalar(uint32(1)))

		ctx.Index(0, ContextLocation).Store(s.Scalar(uint32(0xffffffff)))
		ctx.Index(0, ContextArena).Store(s.Arena)
		ctx.Index(0, ContextNextPoolID).Store(nextPoolID)

		// Initialize custom plugin context fields
		for _, f := range c.T.customCtxFields {
			if f.Init != nil {
				ctx.Index(0, f.Name).Store(f.Init(s))
			}
		}

		// State init
		if c.settings.EmitExec {
			globals := c.Alloc(s, s.Scalar(1), c.T.Globals).SetName("globals")
			// Start by zeroing out the entire state block's memory.
			// While this might seem redundant (as we're about to initialize
			// everything below), there might be alignment and padding in the
			// structures that hold non-deterministic values. These will cause
			// issues with tests.
			s.Memzero(globals.Cast(c.T.VoidPtr), s.SizeOf(c.T.Globals).Cast(c.T.Uint32))
			ctx.Index(0, ContextGlobals).Store(globals)

			for _, g := range c.API.Globals {
				var val *codegen.Value
				if g.Default != nil {
					val = c.expression(s, g.Default)
				} else {
					val = c.initialValue(s, g.Type)
				}
				val.SetName(g.Name())
				c.reference(s, val, g.Type)
				globals.Index(0, g.Name()).Store(val)
			}
		}

		s.Return(ctx)
	})

	c.Build(c.ctx.destroy, func(s *S) {
		ctx := s.Parameter(0)
		s.Arena = ctx.Index(0, ContextArena).Load()

		// Terminate custom plugin context fields
		for _, f := range c.T.customCtxFields {
			if f.Term != nil {
				f.Term(s, ctx.Index(0, f.Name).Load())
			}
		}

		c.Free(s, ctx.Index(0, ContextNextPoolID).Load())
		if c.settings.EmitExec {
			c.Free(s, ctx.Index(0, ContextGlobals).Load())
		}
		c.Free(s, ctx)
	})
}
