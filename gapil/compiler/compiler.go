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
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

//#define QUOTE(x) #x
//#define DECL_GAPIL_CALLBACK(RETURN, NAME, ...) \
//	const char* NAME##_sig = QUOTE(RETURN NAME(__VA_ARGS__));
//#include "gapil/runtime/cc/runtime.h"
import "C"

type Settings struct {
	TargetABI              *device.ABI
	StorageABI             *device.ABI
	EmitExec               bool // Should the compiler generate execution functions for each API command?
	EmitEncode             bool // Should the compiler generate encode functions for each API serializable type?
	CodeLocations          bool
	WriteToApplicationPool bool
}

type compiler struct {
	settings      Settings
	module        *codegen.Module
	functions     map[*semantic.Function]codegen.Function
	stateInit     codegen.Function
	emptyString   codegen.Global
	serialization *serialization
	mappings      *resolver.Mappings
	locationIndex map[Location]int
	locations     []Location
	ty            types
	refRels       refRels
	currentFunc   *semantic.Function
	currentStmt   semantic.Node
	currentExpr   semantic.Expression
	callbacks     struct {
		alloc           codegen.Function
		realloc         codegen.Function
		free            codegen.Function
		applyReads      codegen.Function
		applyWrites     codegen.Function
		freePool        codegen.Function
		makeSlice       codegen.Function
		copySlice       codegen.Function
		pointerToSlice  codegen.Function
		pointerToString codegen.Function
		sliceToString   codegen.Function
		makeString      codegen.Function
		freeString      codegen.Function
		stringToSlice   codegen.Function
		stringConcat    codegen.Function
		stringCompare   codegen.Function
		callExtern      codegen.Function
		logf            codegen.Function
	}
}

func Compile(api *semantic.API, mappings *resolver.Mappings, s Settings) (*Program, error) {
	hostABI := host.Instance(context.Background()).Configuration.ABIs[0]
	if s.TargetABI == nil {
		s.TargetABI = hostABI
	}
	if s.StorageABI == nil {
		s.StorageABI = hostABI
	}

	c := &compiler{
		settings:      s,
		module:        codegen.NewModule("api.executor", s.TargetABI),
		functions:     map[*semantic.Function]codegen.Function{},
		mappings:      mappings,
		locationIndex: map[Location]int{},
		locations:     []Location{},
	}

	c.compile(api)

	prog, err := c.program(s)
	if err != nil {
		return nil, err
	}

	return prog, nil
}

func (c *compiler) program(s Settings) (*Program, error) {
	commands := make(map[string]*CommandInfo, len(c.functions))
	for a, f := range c.functions {
		if a.Subroutine || a.Extern {
			continue
		}
		params := f.Type.Signature.Parameters[1].(codegen.Pointer).Element.(*codegen.Struct)
		commands[f.Name] = &CommandInfo{
			Function:   f,
			Parameters: params,
		}
	}

	globals := &StructInfo{Type: c.ty.globals}

	structs := make(map[string]*StructInfo)
	for _, t := range c.ty.target {
		if s, ok := t.(*codegen.Struct); ok {
			structs[s.Name] = &StructInfo{Type: s}
		}
	}

	maps := make(map[string]*MapInfo)
	for m, mi := range c.ty.maps {
		maps[m.Name()] = mi
	}

	return &Program{
		Settings:    c.settings,
		Commands:    commands,
		Structs:     structs,
		Globals:     globals,
		Maps:        maps,
		Locations:   c.locations,
		Module:      c.module,
		Initializer: c.stateInit,
	}, nil
}

func err(err error) {
	if err != nil {
		panic(err)
	}
}

func fail(msg string, args ...interface{}) { err(fmt.Errorf(msg, args...)) }

func (c *compiler) compile(api *semantic.API) {
	defer c.augmentPanics()

	c.declareTypes(api)

	{
		c.callbacks.alloc = c.module.ParseFunctionSignature(C.GoString(C.gapil_alloc_sig))
		c.callbacks.realloc = c.module.ParseFunctionSignature(C.GoString(C.gapil_realloc_sig))
		c.callbacks.free = c.module.ParseFunctionSignature(C.GoString(C.gapil_free_sig))
		c.callbacks.applyReads = c.module.ParseFunctionSignature(C.GoString(C.gapil_apply_reads_sig))
		c.callbacks.applyWrites = c.module.ParseFunctionSignature(C.GoString(C.gapil_apply_writes_sig))
		c.callbacks.freePool = c.module.ParseFunctionSignature(C.GoString(C.gapil_free_pool_sig))
		c.callbacks.makeSlice = c.module.ParseFunctionSignature(C.GoString(C.gapil_make_slice_sig))
		c.callbacks.copySlice = c.module.ParseFunctionSignature(C.GoString(C.gapil_copy_slice_sig))
		c.callbacks.pointerToSlice = c.module.ParseFunctionSignature(C.GoString(C.gapil_pointer_to_slice_sig))
		c.callbacks.pointerToString = c.module.ParseFunctionSignature(C.GoString(C.gapil_pointer_to_string_sig))
		c.callbacks.sliceToString = c.module.ParseFunctionSignature(C.GoString(C.gapil_slice_to_string_sig))
		c.callbacks.makeString = c.module.ParseFunctionSignature(C.GoString(C.gapil_make_string_sig))
		c.callbacks.freeString = c.module.ParseFunctionSignature(C.GoString(C.gapil_free_string_sig))
		c.callbacks.stringToSlice = c.module.ParseFunctionSignature(C.GoString(C.gapil_string_to_slice_sig))
		c.callbacks.stringConcat = c.module.ParseFunctionSignature(C.GoString(C.gapil_string_concat_sig))
		c.callbacks.stringCompare = c.module.ParseFunctionSignature(C.GoString(C.gapil_string_compare_sig))
		c.callbacks.callExtern = c.module.ParseFunctionSignature(C.GoString(C.gapil_call_extern_sig))
		c.callbacks.logf = c.module.ParseFunctionSignature(C.GoString(C.gapil_logf_sig))
	}

	c.emptyString = c.module.Global("gapil_empty_string",
		c.module.ConstStruct(
			c.ty.str,
			map[string]interface{}{"ref_count": 1},
		),
	)

	c.buildTypes(api)

	if c.settings.EmitExec {
		for _, f := range api.Externs {
			c.extern(f)
		}
		for _, f := range api.Subroutines {
			c.subroutine(f)
		}
		for _, f := range api.Functions {
			c.command(f)
		}
		c.buildStateInit(api)
	}

	if c.settings.EmitEncode {
		c.buildSerialization()
	}
}

func (c *compiler) build(f codegen.Function, do func(*scope)) {
	err(f.Build(func(jb *codegen.Builder) {
		s := &scope{
			Builder:    jb,
			locals:     map[*semantic.Local]*codegen.Value{},
			parameters: map[*semantic.Parameter]*codegen.Value{},
		}
		for i, p := range f.Type.Signature.Parameters {
			if p == c.ty.ctxPtr {
				ctx := jb.Parameter(i).SetName("ctx")
				globals := ctx.Index(0, contextGlobals).Load().SetName("globals")
				arena := ctx.Index(0, contextArena).Load().SetName("arena")
				location := ctx.Index(0, contextLocation)
				s.ctx = ctx
				s.location = location
				s.globals = globals
				s.arena = arena
				break
			}
		}

		s.enter(do)
	}))
}

func (c *compiler) buildStateInit(api *semantic.API) {
	c.stateInit = c.module.Function(c.ty.Void, "init", c.ty.Pointer(c.ty.ctx))
	c.build(c.stateInit, func(s *scope) {
		for _, g := range api.Globals {
			var val *codegen.Value
			if g.Default != nil {
				val = c.expression(s, g.Default)
			} else {
				val = c.initialValue(s, g.Type)
			}
			c.reference(s, val, g.Type)
			s.globals.Index(0, g.Name()).Store(val)
		}
	})
}

func (c *compiler) alloc(s *scope, arena, count *codegen.Value, ty codegen.Type) *codegen.Value {
	size := s.Mul(count, s.SizeOf(ty).Cast(c.ty.Uint64))
	align := s.AlignOf(ty).Cast(c.ty.Uint64)
	return s.Call(c.callbacks.alloc, s.arena, size, align).Cast(c.ty.Pointer(ty))
}

func (c *compiler) realloc(s *scope, arena, old, count *codegen.Value, ty codegen.Type) *codegen.Value {
	size := s.Mul(count, s.SizeOf(ty).Cast(c.ty.Uint64))
	align := s.AlignOf(ty).Cast(c.ty.Uint64)
	return s.Call(c.callbacks.realloc, s.arena, old.Cast(c.ty.u8Ptr), size, align).Cast(c.ty.Pointer(ty))
}

func (c *compiler) free(s *scope, arena, ptr *codegen.Value) {
	s.Call(c.callbacks.free, arena, ptr.Cast(c.ty.voidPtr))
}

func (c *compiler) logf(s *scope, severity log.Severity, msg string, args ...interface{}) {
	ctx := s.ctx
	if ctx == nil {
		ctx = s.Zero(c.ty.ctxPtr)
	}
	fullArgs := []*codegen.Value{
		ctx,
		s.Scalar(uint8(severity)),
		s.Scalar(msg),
	}
	for _, arg := range args {
		if val, ok := arg.(*codegen.Value); ok {
			fullArgs = append(fullArgs, val)
		} else {
			fullArgs = append(fullArgs, s.Scalar(arg))
		}
	}
	c.updateCodeLocation(s)
	s.Call(c.callbacks.logf, fullArgs...)
}

func (c *compiler) setCodeLocation(s *scope, t parse.Token) {
	_, file := filepath.Split(t.Source.Filename)
	line, col := t.Cursor()
	loc := Location{file, line, col}
	idx, ok := c.locationIndex[loc]
	if !ok {
		idx = len(c.locations)
		c.locations = append(c.locations, loc)
	}
	if idx != s.locationIdx {
		s.locationIdx = idx
		if c.settings.CodeLocations {
			c.updateCodeLocation(s)
		}
	}
}

func (c *compiler) updateCodeLocation(s *scope) {
	if s.location != nil {
		s.location.Store(s.Scalar(uint32(s.locationIdx)))
	}
}

func (c *compiler) setCurrentFunction(f *semantic.Function) *semantic.Function {
	old := c.currentFunc
	c.currentFunc = f
	return old
}

func (c *compiler) setCurrentStatement(s *scope, n semantic.Node) semantic.Node {
	old := c.currentStmt
	c.currentStmt = n
	for _, ast := range c.mappings.SemanticToAST[n] {
		if cst := c.mappings.CST(ast); cst != nil {
			c.setCodeLocation(s, cst.Token())
			break
		}
	}
	return old
}

func (c *compiler) setCurrentExpression(s *scope, e semantic.Expression) semantic.Expression {
	old := c.currentExpr
	c.currentExpr = e
	return old
}

func (c *compiler) augmentPanics() {
	r := recover()
	if r == nil {
		return
	}

	loc := func(n semantic.Node) string {
		for _, ast := range c.mappings.SemanticToAST[n] {
			if cst := c.mappings.CST(ast); cst != nil {
				tok := cst.Token()
				line, col := tok.Cursor()
				file := tok.Source.Filename
				return fmt.Sprintf("%v:%v:%v\n%s", file, line, col, tok.String())
			}
			return "nil node"
		}
		// return "no mapping"
		return fmt.Sprintf("%T %+v", n, n)
	}

	what, where := "<unknown>", "<unknown source location>"
	switch {
	case c.currentExpr != nil:
		what = "expression"
		where = loc(c.currentExpr)
	case c.currentStmt != nil:
		what = "statement"
		where = loc(c.currentStmt)
	}

	if c.currentFunc != nil {
		where = fmt.Sprintf("%v()\n%v", c.currentFunc.Name(), where)
	}

	panic(fmt.Errorf("Internal compiler error processing %v at:\n%v\n%v", what, where, r))
}
