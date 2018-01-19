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

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/semantic"
)

type serialization struct {
	*compiler
}

func (c *compiler) buildSerialization() {
	c.serialization = &serialization{c}
	if c.settings.EmitEncode {
		c.serialization.buildEncoderFuncs()
	}
}

func (c *serialization) buildEncoderFuncs() {
	for apiTy, cgTy := range c.ty.target {
		switch apiTy := apiTy.(type) {
		case *semantic.Class:
			f := c.module.Function(c.ty.Void, "gapil_encode_"+apiTy.Name(), c.ty.ctxPtr, c.ty.Pointer(cgTy))
			c.build(f, func(s *scope) {
				for i, f := range apiTy.Fields {
					c.encodeField(s, i, f.Type)
				}
			})
		}
	}
}

func (c *serialization) encodeClass(s *scope, val *codegen.Value, ty *semantic.Class) {
	c.logf(s, log.Info, "encoding class: %s '"+ty.Name()+"'")
}

func (c *serialization) encodeRef(s *scope, val *codegen.Value, ty *semantic.Reference) {
	c.logf(s, log.Info, "encoding reference: %s '"+ty.Name()+"'")
}

func (c *serialization) encodeField(s *scope, fieldIdx int, ty semantic.Type) {
	c.logf(s, log.Info, "encoding field %s of type '"+ty.Name()+"'", fieldIdx)
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.Uint8Type:
		case semantic.Int8Type:
		case semantic.Uint16Type:
		case semantic.Int16Type:
		case semantic.Uint32Type:
		case semantic.Int32Type:
		case semantic.Uint64Type:
		case semantic.Int64Type:
		case semantic.IntType:
		case semantic.SizeType:
		}
	}
}
