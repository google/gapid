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
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/compiler/mangling/c"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

//#define QUOTE(x) #x
//#define DECL_GAPIL_CB(RETURN, NAME, ...) \
//	const char* NAME##_sig = QUOTE(RETURN NAME(__VA_ARGS__));
//#include "gapil/runtime/cc/runtime.h"
import "C"

// C is the compiler context used to build the program.
type C struct {
	// T holds the compiler types.
	T Types

	// M is the codegen module for the program.
	M *codegen.Module

	// API is the api that is being compiled.
	API *semantic.API

	// Mangler is the symbol mangler in use.
	Mangler mangling.Mangler

	// Root is the namespace in which generated symbols should be placed.
	// This excludes gapil runtime symbols which have the prefix 'gapil_'.
	Root mangling.Scope

	settings  Settings
	plugins   plugins
	functions map[*semantic.Function]codegen.Function
	ctx       struct { // Functions that operate on contexts
		create  codegen.Function
		destroy codegen.Function
	}
	buf struct { // Functions that operate on buffers
		init   codegen.Function
		term   codegen.Function
		append codegen.Function
	}
	emptyString   codegen.Global
	mappings      *resolver.Mappings
	locationIndex map[Location]int
	locations     []Location
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
		copySlice       codegen.Function
		makePool        codegen.Function
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

// Compile compiles the given API semantic tree to a program using the given
// settings.
func Compile(api *semantic.API, mappings *resolver.Mappings, s Settings) (*Program, error) {
	hostABI := host.Instance(context.Background()).Configuration.ABIs[0]
	if s.TargetABI == nil {
		s.TargetABI = hostABI
	}
	if s.StorageABI == nil {
		s.StorageABI = hostABI
	}
	if s.Mangler == nil {
		s.Mangler = c.Mangle
	}
	if s.EmitExec {
		s.EmitContext = true
	}

	c := &C{
		M:       codegen.NewModule("api.executor", s.TargetABI),
		API:     api,
		Mangler: s.Mangler,

		plugins:       s.Plugins,
		settings:      s,
		functions:     map[*semantic.Function]codegen.Function{},
		mappings:      mappings,
		locationIndex: map[Location]int{},
		locations:     []Location{},
	}
	for _, n := range s.Namespaces {
		c.Root = &mangling.Namespace{Name: n, Parent: c.Root}
	}

	c.compile()

	prog, err := c.program(s)
	if err != nil {
		return nil, err
	}

	if err := prog.Module.Verify(); err != nil {
		return nil, err
	}

	return prog, nil
}

func (c *C) program(s Settings) (*Program, error) {
	commands := make(map[string]*CommandInfo, len(c.functions))
	for a, f := range c.functions {
		if a.Subroutine || a.Extern {
			continue
		}
		params := f.Type.Signature.Parameters[1].(codegen.Pointer).Element.(*codegen.Struct)
		commands[f.Name] = &CommandInfo{
			Execute:    f,
			Parameters: params,
		}
	}

	globals := &StructInfo{Type: c.T.Globals}

	structs := make(map[string]*StructInfo)
	for _, t := range c.T.target {
		if s, ok := t.(*codegen.Struct); ok {
			structs[s.Name] = &StructInfo{Type: s}
		}
	}

	maps := make(map[string]*MapInfo)
	for m, mi := range c.T.Maps {
		maps[m.Name()] = mi
	}

	return &Program{
		Settings:       c.settings,
		Commands:       commands,
		Structs:        structs,
		Globals:        globals,
		Maps:           maps,
		Locations:      c.locations,
		Module:         c.M,
		CreateContext:  c.ctx.create,
		DestroyContext: c.ctx.destroy,
	}, nil
}

func err(err error) {
	if err != nil {
		panic(err)
	}
}

func fail(msg string, args ...interface{}) { err(fmt.Errorf(msg, args...)) }

func (c *C) compile() {
	defer c.augmentPanics()

	c.declareTypes()

	c.callbacks.alloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_alloc_sig))
	c.callbacks.realloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_realloc_sig))
	c.callbacks.free = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_sig))
	c.callbacks.applyReads = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_reads_sig))
	c.callbacks.applyWrites = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_writes_sig))
	c.callbacks.freePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_pool_sig))
	c.callbacks.copySlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_copy_slice_sig))
	c.callbacks.makePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_pool_sig))
	c.callbacks.pointerToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_pointer_to_slice_sig))
	c.callbacks.pointerToString = c.M.ParseFunctionSignature(C.GoString(C.gapil_pointer_to_string_sig))
	c.callbacks.sliceToString = c.M.ParseFunctionSignature(C.GoString(C.gapil_slice_to_string_sig))
	c.callbacks.makeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_string_sig))
	c.callbacks.freeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_string_sig))
	c.callbacks.stringToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_to_slice_sig))
	c.callbacks.stringConcat = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_concat_sig))
	c.callbacks.stringCompare = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_compare_sig))
	c.callbacks.callExtern = c.M.ParseFunctionSignature(C.GoString(C.gapil_call_extern_sig))
	c.callbacks.logf = c.M.ParseFunctionSignature(C.GoString(C.gapil_logf_sig))

	c.emptyString = c.M.Global("gapil_empty_string",
		c.M.ConstStruct(
			c.T.Str,
			map[string]interface{}{"ref_count": 1},
		),
	)

	c.buildTypes()

	if c.settings.EmitExec {
		for _, f := range c.API.Externs {
			c.extern(f)
		}
		for _, f := range c.API.Subroutines {
			c.subroutine(f)
		}
		for _, f := range c.API.Functions {
			c.command(f)
		}
	}

	c.buildContextFuncs()
	c.buildBufferFuncs()

	c.plugins.foreach(func(p Plugin) { p.Build(c) })
}

// Build implements the function f by creating a new scope and calling do to
// emit the function body.
// If the function has a parameter of type context_t* then the Ctx, Location,
// Globals and Arena scope fields are automatically assigned.
func (c *C) Build(f codegen.Function, do func(*S)) {
	err(f.Build(func(jb *codegen.Builder) {
		s := &S{
			Builder:    jb,
			locals:     map[*semantic.Local]*codegen.Value{},
			parameters: map[*semantic.Parameter]*codegen.Value{},
		}
		for i, p := range f.Type.Signature.Parameters {
			if p == c.T.CtxPtr {
				s.Ctx = jb.Parameter(i).SetName("ctx")
				s.Globals = s.Ctx.Index(0, ContextGlobals).Load().SetName("globals")
				s.Arena = s.Ctx.Index(0, ContextArena).Load().SetName("arena")
				s.Location = s.Ctx.Index(0, ContextLocation)
				break
			}
		}

		s.enter(do)
	}))
}

// MakeSlice creates a new slice of the given size in bytes.
func (c *C) MakeSlice(s *S, size, count *codegen.Value) *codegen.Value {
	dstPtr := s.Local("dstPtr", c.T.Sli)
	c.MakeSliceAt(s, size, count, dstPtr)
	return dstPtr.Load()
}

// MakeSliceAt creates a new slice of the given size in bytes at the given
// slice pointer.
func (c *C) MakeSliceAt(s *S, size, count, dstPtr *codegen.Value) {
	pool := s.Call(c.callbacks.makePool, s.Ctx, size)
	buf := pool.Index(0, PoolBuffer).Load()
	dstPtr.Index(0, SlicePool).Store(pool)
	dstPtr.Index(0, SliceRoot).Store(buf)
	dstPtr.Index(0, SliceBase).Store(buf)
	dstPtr.Index(0, SliceSize).Store(size)
	dstPtr.Index(0, SliceCount).Store(count)
}

// CopySlice copies the contents of slice src to dst.
func (c *C) CopySlice(s *S, dst, src *codegen.Value) {
	s.Call(c.callbacks.copySlice, s.Ctx, s.LocalInit("dstPtr", dst), s.LocalInit("srcPtr", src))
}

// MakeString creates a new string from the specified data and length in bytes.
func (c *C) MakeString(s *S, length, data *codegen.Value) *codegen.Value {
	return s.Call(c.callbacks.makeString, s.Arena, length, data)
}

// Alloc calls gapil_alloc to allocate a buffer big enough to hold count
// elements of type ty.
func (c *C) Alloc(s *S, count *codegen.Value, ty codegen.Type) *codegen.Value {
	if s.Arena == nil {
		fail("Cannot call Alloc without an arena")
	}
	size := s.Mul(count.Cast(c.T.Uint64), s.SizeOf(ty).Cast(c.T.Uint64))
	align := s.AlignOf(ty).Cast(c.T.Uint64)
	return s.Call(c.callbacks.alloc, s.Arena, size, align).Cast(c.T.Pointer(ty))
}

// Realloc calls gapil_realloc to reallocate a buffer that was previously
// allocated with Alloc or Realloc.
func (c *C) Realloc(s *S, old, count *codegen.Value) *codegen.Value {
	if s.Arena == nil {
		fail("Cannot call Realloc without an arena")
	}
	ty := old.Type().(codegen.Pointer).Element
	size := s.Mul(count.Cast(c.T.Uint64), s.SizeOf(ty).Cast(c.T.Uint64))
	align := s.AlignOf(ty).Cast(c.T.Uint64)
	return s.Call(c.callbacks.realloc, s.Arena, old.Cast(c.T.VoidPtr), size, align).Cast(c.T.Pointer(ty))
}

// Free calls gapil_free to release a buffer that was previously allocated with
// Alloc or Realloc.
func (c *C) Free(s *S, ptr *codegen.Value) {
	if s.Arena == nil {
		fail("Cannot call Realloc without an arena")
	}
	s.Call(c.callbacks.free, s.Arena, ptr.Cast(c.T.VoidPtr))
}

// Log emits a call to gapil_logf with the given printf-style arguments.
// args can be a mix of codegen.Values or simple data types (which are
// automatically passed to codegen.Builder.Scalar).
func (c *C) Log(s *S, severity log.Severity, msg string, args ...interface{}) {
	ctx := s.Ctx
	if ctx == nil {
		ctx = s.Zero(c.T.CtxPtr)
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

// Fail is used to immediately terminate compilation due to an internal
// compiler error.
func (c *C) Fail(msg string, args ...interface{}) { fail(msg, args...) }

func (c *C) setCodeLocation(s *S, t parse.Token) {
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
		c.updateCodeLocation(s)
	}
}

func (c *C) updateCodeLocation(s *S) {
	if c.settings.CodeLocations && s.Location != nil {
		s.Location.Store(s.Scalar(uint32(s.locationIdx)))
	}
}

func (c *C) setCurrentFunction(f *semantic.Function) *semantic.Function {
	old := c.currentFunc
	c.currentFunc = f
	return old
}

func (c *C) setCurrentStatement(s *S, n semantic.Node) semantic.Node {
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

func (c *C) setCurrentExpression(s *S, e semantic.Expression) semantic.Expression {
	old := c.currentExpr
	c.currentExpr = e
	return old
}

func (c *C) augmentPanics() {
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
