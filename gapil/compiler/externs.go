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

func (c *C) extern(f *semantic.Function) {
	if _, ok := c.externs[f]; ok {
		panic(fmt.Errorf("Duplicate extern '%v'", f.Name()))
	}
	callParams := f.CallParameters()
	paramTys := make([]codegen.Type, 1, len(callParams)+2)
	paramTys[0] = c.T.CtxPtr
	for _, p := range callParams {
		ty := c.T.Target(p.Type)
		if codegen.IsStruct(ty) {
			// Aggregate types need to be passed by pointer
			ty = c.T.Pointer(ty)
		}
		paramTys = append(paramTys, ty)
	}
	if f.Return.Type != semantic.VoidType {
		resTy := c.T.Target(f.Return.Type)
		paramTys = append(paramTys, c.T.Pointer(resTy))
	}
	name := fmt.Sprintf("%v_%v", c.CurrentAPI().Name(), f.Name())
	c.externs[f] = c.M.Function(c.T.Void, name, paramTys...)
}

func (c *C) callExtern(s *S, e *semantic.Call) *codegen.Value {
	tf := e.Target.Function
	args := make([]*codegen.Value, len(e.Arguments)+1)
	args[0] = s.Ctx
	for i, a := range e.Arguments {
		if codegen.IsStruct(c.T.Target(a.ExpressionType())) {
			// Aggregate types need to be passed by pointer
			args[i+1] = c.expressionAddr(s, a).SetName(tf.FullParameters[i].Name())
		} else {
			args[i+1] = c.expression(s, a).SetName(tf.FullParameters[i].Name())
		}
	}

	f, ok := c.externs[tf]
	if !ok {
		panic(fmt.Errorf("Couldn't resolve extern call target %v", tf.Name()))
	}

	var res *codegen.Value

	// Result is passed back by pointer via the last argument.
	// This is done to avoid complexities of ABI calling conventions of
	// aggregate types.
	if tf.Return.Type != semantic.VoidType {
		resPtr := s.LocalInit("call-res", s.Zero(c.T.Target(tf.Return.Type)))
		args = append(args, resPtr)
		s.Call(f, args...)
		res = resPtr.Load()
	} else {
		s.Call(f, args...)
	}

	if res != nil {
		c.deferRelease(s, res, tf.Return.Type)
	}

	return res
}
