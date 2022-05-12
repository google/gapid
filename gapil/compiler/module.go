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

// Package compiler implements the core gapil language compiler.
//
// The compiler will generate types and command execution functions using LLVM
// for a resolved API. The compiler can be extended with Plugins for additional
// functionality.
package compiler

import (
	"sort"

	"github.com/google/gapid/core/codegen"
)

// buildModule builds the output module definition.
func (c *C) buildModule() {
	apiModuleTy := c.T.Struct("module_api",
		codegen.Field{Name: "globals_offset", Type: c.T.Uint64},
		codegen.Field{Name: "globals_size", Type: c.T.Uint64},
		codegen.Field{Name: "num_cmds", Type: c.T.Uint32},
		codegen.Field{Name: "cmds", Type: c.T.VoidPtrPtr},
	)
	symbolTy := c.T.Struct("symbol",
		codegen.Field{Name: "name", Type: c.T.Pointer(c.T.Uint8)},
		codegen.Field{Name: "addr", Type: c.T.VoidPtr},
	)
	moduleTy := c.T.Struct("module",
		codegen.Field{Name: "create_context", Type: c.T.Pointer(c.ctx.create.Type)},
		codegen.Field{Name: "destroy_context", Type: c.T.Pointer(c.ctx.destroy.Type)},
		codegen.Field{Name: "globals_size", Type: c.T.Uint64},
		codegen.Field{Name: "num_apis", Type: c.T.Uint32},
		codegen.Field{Name: "apis", Type: c.T.Pointer(apiModuleTy)},
		codegen.Field{Name: "num_symbols", Type: c.T.Uint32},
		codegen.Field{Name: "symbols", Type: c.T.Pointer(symbolTy)},
	)

	apiMaxIndex := 0
	for _, api := range c.APIs {
		if apiMaxIndex < int(api.Index) {
			apiMaxIndex = int(api.Index)
		}
	}

	apis := make([]codegen.Const, apiMaxIndex+1)
	for i := range apis {
		apis[i] = c.M.Zero(apiModuleTy)
	}

	for _, api := range c.APIs {
		commands := make([]*codegen.Function, len(api.Functions))

		for i, f := range api.Functions {
			commands[i] = c.commands[f]
		}

		cmdTbl := c.M.
			Global(api.Name()+"-cmd-tbl", c.M.Array(commands, c.T.VoidPtr)).
			SetConstant(true).
			Cast(c.T.VoidPtr)

		apis[api.Index] = c.M.ConstStruct(
			apiModuleTy, map[string]interface{}{
				"globals_offset": c.M.OffsetOf(c.T.Globals, api.Name()),
				"globals_size":   c.M.SizeOf(c.T.Globals.Field(api.Name()).Type),
				"num_cmds":       len(commands),
				"cmds":           cmdTbl.Cast(c.T.VoidPtrPtr),
			},
		)
	}

	apiTbl := c.M.
		Global("api-tbl", c.M.Array(apis, apiModuleTy)).
		SetConstant(true)

	functions := make([]*codegen.Function, 0, len(c.functions))
	for _, f := range c.functions {
		functions = append(functions, f)
	}
	sort.Slice(functions, func(i, j int) bool { return functions[i].Name < functions[j].Name })

	symbols := make([]codegen.Const, len(functions))
	for i, f := range functions {
		symbols[i] = c.M.ConstStruct(
			symbolTy, map[string]interface{}{
				"name": c.M.Scalar(f.Name),
				"addr": c.M.Scalar(f).Cast(c.T.VoidPtr),
			},
		)
	}

	symbolTbl := c.M.
		Global("symbols", c.M.Array(symbols, symbolTy)).
		SetConstant(true)

	fields := map[string]interface{}{
		"num_apis":     c.M.Scalar(uint32(len(apis))),
		"apis":         apiTbl.Cast(c.T.Pointer(apiModuleTy)),
		"num_symbols":  c.M.Scalar(uint32(len(symbols))),
		"symbols":      symbolTbl.Cast(c.T.Pointer(symbolTy)),
		"globals_size": c.M.SizeOf(c.T.Globals),
	}

	c.module = c.M.Global(
		c.Settings.Module,
		c.M.ConstStruct(moduleTy, fields),
	)

	if c.Settings.Module != "" {
		c.module.LinkPublic()
	}
}
