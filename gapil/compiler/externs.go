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
	if _, ok := c.functions[f]; ok {
		panic(fmt.Errorf("Duplicate extern '%v'", f.Name()))
	}
	callParams := f.CallParameters()
	paramTys := make([]codegen.Type, 1, len(callParams)+2)
	paramTys[0] = c.T.CtxPtr
	for _, p := range callParams {
		paramTys = append(paramTys, c.T.Target(p.Type))
	}
	if f.Return.Type != semantic.VoidType {
		resTy := c.T.Target(f.Return.Type)
		paramTys = append(paramTys, c.T.Pointer(resTy))
	}
	name := fmt.Sprintf("%v_%v", c.CurrentAPI().Name(), f.Name())
	c.functions[f] = c.M.Function(c.T.Void, name, paramTys...)
}
