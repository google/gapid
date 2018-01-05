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

func (c *compiler) extern(f *semantic.Function) {
	if _, ok := c.functions[f]; ok {
		panic(fmt.Errorf("Duplicate extern '%v'", f.Name()))
	}
	old := c.setCurrentFunction(f)
	resTy := c.targetType(f.Return.Type)
	params := f.CallParameters()
	fields := make([]codegen.Field, len(params))
	for i, p := range params {
		fields[i] = codegen.Field{
			Name: p.Name(),
			Type: c.targetType(p.Type),
		}
	}
	paramsTy := c.ty.Struct(f.Name()+"_params", fields...)
	c.externs[f] = &ExternInfo{
		Name:       f.Name(),
		Parameters: paramsTy,
		Result:     resTy,
	}
	c.setCurrentFunction(old)
}

func (c *compiler) callExtern(s *scope, e *ExternInfo, call *semantic.Call) *codegen.Value {
	panic("TODO")
	/*
		args := s.Local(e.Name+"_args", e.Parameters)
		for i, f := range e.Parameters.Fields {
			arg := c.expression(s, call.Arguments[i]).SetName(f.Name)
			args.Index(0, f.Name).Store(arg)
		}

		id := c.addString(s, e.Name)

		if e.Result != c.ty.Void {
			res := s.Local(e.Name+"_res", e.Result)
			s.Call(c.callbacks.callExtern, s.ctx, id, args.Cast(c.u8PtrTy), res.Cast(c.u8PtrTy))
			return res.Load()
		}

		s.Call(c.callbacks.callExtern, s.ctx, id, args.Cast(c.u8PtrTy), s.Zero(c.u8PtrTy))
		return nil
	*/
}
