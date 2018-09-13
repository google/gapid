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

func (c *C) callExtern(s *S, e *semantic.Call) *codegen.Value {
	tf := e.Target.Function
	vals := make([]*codegen.Value, len(e.Arguments), len(e.Arguments)+1)
	for i, a := range e.Arguments {
		vals[i] = c.expression(s, a).SetName(tf.FullParameters[i].Name())
	}
	if tf.Return.Type != semantic.VoidType {
		vals = append(vals, s.Zero(c.T.Target(tf.Return.Type)))
	}

	name := fmt.Sprintf("%v.%v", c.CurrentAPI().Name(), tf.Name())
	args := s.LocalInit(tf.Name()+"_params", s.StructOf(name+"-args", vals))

	if tf.Return.Type != semantic.VoidType {
		resPtr := s.LocalInit(tf.Name()+"_res", s.Zero(c.T.Target(tf.Return.Type)))
		s.Call(c.callbacks.callExtern, s.Ctx, s.Scalar(name), args.Cast(c.T.VoidPtr), resPtr.Cast(c.T.VoidPtr))
		res := resPtr.Load()
		c.deferRelease(s, res, tf.Return.Type)
		return res
	}

	s.Call(c.callbacks.callExtern, s.Ctx, s.Scalar(name), args.Cast(c.T.VoidPtr), s.Zero(c.T.VoidPtr))
	return nil
}
