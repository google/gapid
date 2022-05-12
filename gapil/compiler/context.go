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

	c.plugins.foreach(func(p ContextDataPlugin) {
		p.OnPreBuildContext(c)
	})

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

	c.T.Ctx.SetBody(false, fields...)
}

func (c *C) buildContextFuncs() {
	// Always declare these functions - their types may be used even if we're
	// not emitting contexts.
	c.ctx.create = c.M.Function(c.T.CtxPtr, "gapil_create_context", c.T.ArenaPtr)
	c.ctx.destroy = c.M.Function(c.T.Void, "gapil_destroy_context", c.T.CtxPtr)
}
