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
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/compiler/mangling/c"
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

	// APIs are the apis that are being compiled.
	APIs []*semantic.API

	// Mangler is the symbol mangler in use.
	Mangler mangling.Mangler

	// Root is the namespace in which generated symbols should be placed.
	// This excludes gapil runtime symbols which have the prefix 'gapil_'.
	Root mangling.Scope

	// Settings are the configuration values used for this compile.
	Settings Settings

	plugins   plugins
	functions map[*semantic.Function]*codegen.Function
	ctx       struct { // Functions that operate on contexts
		create  *codegen.Function
		destroy *codegen.Function
	}
	buf struct { // Functions that operate on buffers
		init   *codegen.Function
		term   *codegen.Function
		append *codegen.Function
	}
	emptyString     codegen.Global
	mappings        *semantic.Mappings
	locationIndex   map[Location]int
	locations       []Location
	refRels         refRels
	currentAPI      *semantic.API
	currentFunc     *semantic.Function
	statementStack  []semantic.Statement
	expressionStack []semantic.Expression
	isFence         bool // If true, a fence should be emitted for the given statement
	callbacks       struct {
		alloc          *codegen.Function
		realloc        *codegen.Function
		free           *codegen.Function
		applyReads     *codegen.Function
		applyWrites    *codegen.Function
		freePool       *codegen.Function
		sliceData      *codegen.Function
		copySlice      *codegen.Function
		makePool       *codegen.Function
		cstringToSlice *codegen.Function
		sliceToString  *codegen.Function
		makeString     *codegen.Function
		freeString     *codegen.Function
		stringToSlice  *codegen.Function
		stringConcat   *codegen.Function
		stringCompare  *codegen.Function
		logf           *codegen.Function
	}
}

// Compile compiles the given API semantic tree to a program using the given
// settings.
func Compile(apis []*semantic.API, mappings *semantic.Mappings, s Settings) (*Program, error) {
	hostABI := host.Instance(context.Background()).Configuration.ABIs[0]
	if s.TargetABI == nil {
		s.TargetABI = hostABI
	}
	if s.CaptureABI == nil {
		s.CaptureABI = hostABI
	}
	if s.Mangler == nil {
		s.Mangler = c.Mangle
	}
	if s.EmitExec {
		s.EmitContext = true
	}

	c := &C{
		M:        codegen.NewModule("api.executor", s.TargetABI),
		APIs:     apis,
		Mangler:  s.Mangler,
		Settings: s,

		plugins:       s.Plugins,
		functions:     map[*semantic.Function]*codegen.Function{},
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
		commands[a.Name()] = &CommandInfo{
			Execute:    f,
			Parameters: c.T.CmdParams[a].(*codegen.Struct),
		}
	}

	globals := &StructInfo{Type: c.T.Globals}

	functions := map[string]*codegen.Function{}
	c.plugins.foreach(func(p FunctionExposerPlugin) {
		for n, f := range p.Functions() {
			functions[n] = f
		}
	})

	structs := map[string]*StructInfo{}
	for _, t := range c.T.target {
		if s, ok := t.(*codegen.Struct); ok {
			structs[s.TypeName()] = &StructInfo{Type: s}
		}
	}

	maps := map[string]*MapInfo{}
	for m, mi := range c.T.Maps {
		maps[m.Name()] = mi
	}

	return &Program{
		Settings:       c.Settings,
		APIs:           c.APIs,
		Commands:       commands,
		Structs:        structs,
		Globals:        globals,
		Functions:      functions,
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
	c.declareBufferFuncs()
	c.declareContextType()

	c.callbacks.alloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_alloc_sig))
	c.callbacks.realloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_realloc_sig))
	c.callbacks.free = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_sig))
	c.callbacks.applyReads = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_reads_sig))
	c.callbacks.applyWrites = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_writes_sig))
	c.callbacks.freePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_pool_sig))
	c.callbacks.sliceData = c.M.ParseFunctionSignature(C.GoString(C.gapil_slice_data_sig))
	c.callbacks.copySlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_copy_slice_sig))
	c.callbacks.makePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_pool_sig))
	c.callbacks.cstringToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_cstring_to_slice_sig))
	c.callbacks.sliceToString = c.M.ParseFunctionSignature(C.GoString(C.gapil_slice_to_string_sig))
	c.callbacks.makeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_string_sig))
	c.callbacks.freeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_string_sig))
	c.callbacks.stringToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_to_slice_sig))
	c.callbacks.stringConcat = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_concat_sig))
	c.callbacks.stringCompare = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_compare_sig))
	c.callbacks.logf = c.M.ParseFunctionSignature(C.GoString(C.gapil_logf_sig))

	c.emptyString = c.M.Global("gapil_empty_string",
		c.M.ConstStruct(
			c.T.Str,
			map[string]interface{}{"ref_count": 1},
		),
	)

	c.buildTypes()
	c.buildBufferFuncs()
	c.buildContextFuncs()

	c.plugins.foreach(func(p Plugin) { p.Build(c) })

	if c.Settings.EmitExec {
		for _, api := range c.APIs {
			c.currentAPI = api
			for _, f := range api.Externs {
				c.extern(f)
			}
			for _, f := range api.Subroutines {
				c.subroutine(f)
			}
			for _, f := range api.Functions {
				c.command(f)
			}
		}
	}
}

// Build implements the function f by creating a new scope and calling do to
// emit the function body.
// If the function has a parameter of type context_t* then the Ctx, Location,
// Globals and Arena scope fields are automatically assigned.
func (c *C) Build(f *codegen.Function, do func(*S)) {
	err(f.Build(func(jb *codegen.Builder) {
		s := &S{
			Builder:    jb,
			Parameters: map[*semantic.Parameter]*codegen.Value{},
			locals:     map[*semantic.Local]*codegen.Value{},
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
	dstPtr.Index(0, SlicePool).Store(pool)
	dstPtr.Index(0, SliceRoot).Store(s.Scalar(uint64(0)))
	dstPtr.Index(0, SliceBase).Store(s.Scalar(uint64(0)))
	dstPtr.Index(0, SliceSize).Store(size)
	dstPtr.Index(0, SliceCount).Store(count)
}

// CopySlice copies the contents of slice src to dst.
func (c *C) CopySlice(s *S, dst, src *codegen.Value) {
	s.Call(c.callbacks.copySlice, s.Ctx, s.LocalInit("dstPtr", dst), s.LocalInit("srcPtr", src))
}

// SliceDataForRead returns a pointer to an array of slice elements.
// This pointer should be used to read (not write) from the slice.
// The pointer is only valid until the slice is touched again.
func (c *C) SliceDataForRead(s *S, slicePtr *codegen.Value, elType codegen.Type) *codegen.Value {
	access := s.Scalar(Read).Cast(c.T.DataAccess)
	return s.Call(c.callbacks.sliceData, s.Ctx, slicePtr, access).Cast(c.T.Pointer(elType))
}

// SliceDataForWrite returns a pointer to an array of slice elements.
// This pointer should be used to write (not read) to the slice.
// The pointer is only valid until the slice is touched again.
func (c *C) SliceDataForWrite(s *S, slicePtr *codegen.Value, elType codegen.Type) *codegen.Value {
	access := s.Scalar(Write).Cast(c.T.DataAccess)
	return s.Call(c.callbacks.sliceData, s.Ctx, slicePtr, access).Cast(c.T.Pointer(elType))
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
	loc := c.SourceLocation()
	fullArgs := []*codegen.Value{
		s.Scalar(uint8(severity)),
		s.Scalar(loc.File),
		s.Scalar(uint32(loc.Line)),
		s.Scalar(msg),
	}
	for _, arg := range args {
		if val, ok := arg.(*codegen.Value); ok {
			fullArgs = append(fullArgs, val)
		} else {
			fullArgs = append(fullArgs, s.Scalar(arg))
		}
	}
	s.Call(c.callbacks.logf, fullArgs...)
}

// LogI is short hand for Log(s, log.Info, msg, args...)
func (c *C) LogI(s *S, msg string, args ...interface{}) {
	c.Log(s, log.Info, msg, args...)
}

// Fail is used to immediately terminate compilation due to an internal
// compiler error.
func (c *C) Fail(msg string, args ...interface{}) { fail(msg, args...) }

// Delegate builds the function from with a simple body that calls to, with
// implicit casts for each of the parameters. If the function to returns a
// value, this is cast to the from return type and returned.
// Delegate can be used to produce stub functions that have equivalent
// signatures when lowered to LLVM types.
func (c *C) Delegate(from, to *codegen.Function) {
	c.Build(from, func(s *S) {
		args := make([]*codegen.Value, len(from.Type.Signature.Parameters))
		for i := range args {
			args[i] = s.Parameter(i).Cast(to.Type.Signature.Parameters[i])
		}
		res := s.Call(to, args...)
		if ty := from.Type.Signature.Result; ty != c.T.Void {
			s.Return(res.Cast(ty))
		}
	})
}

// StatementStack returns the current build stack of statements.
func (c *C) StatementStack() []semantic.Statement {
	return append([]semantic.Statement{}, c.statementStack...)
}

// ExpressionStack returns the current build stack of expressions.
func (c *C) ExpressionStack() []semantic.Expression {
	return append([]semantic.Expression{}, c.expressionStack...)
}

// CurrentAPI returns the API that is currently being built.
func (c *C) CurrentAPI() *semantic.API {
	return c.currentAPI
}

// CurrentStatement returns the statement that is currently being built.
func (c *C) CurrentStatement() semantic.Statement {
	if len(c.statementStack) == 0 {
		return nil
	}
	return c.statementStack[len(c.statementStack)-1]
}

// CurrentExpression returns the expression that is currently being built.
func (c *C) CurrentExpression() semantic.Expression {
	if len(c.expressionStack) == 0 {
		return nil
	}
	return c.expressionStack[len(c.expressionStack)-1]
}

// SourceLocation associates a semantic node with its location in a source file.
type SourceLocation struct {
	Node   semantic.Node
	File   string
	Line   int
	Column int
}

func (l SourceLocation) String() string {
	parts := []string{}
	if l.Node != nil {
		parts = append(parts, fmt.Sprintf("%T", l.Node))
	}
	if l.File != "" {
		parts = append(parts, l.File)
	} else {
		parts = append(parts, "<unknown>")
	}
	if l.Line != 0 {
		parts = append(parts, fmt.Sprint(l.Line))
	}
	if l.Column != 0 {
		parts = append(parts, fmt.Sprint(l.Column))
	}
	return strings.Join(parts, ":")
}

// SourceLocationFor returns a string describing the source location of the
// given semantic node.
func (c *C) SourceLocationFor(n semantic.Node) SourceLocation {
	if cst := c.mappings.CST(n); cst != nil {
		tok := cst.Tok()
		line, col := tok.Cursor()
		file := tok.Source.Filename
		if i := strings.LastIndex(file, "gapid/"); i > 0 {
			file = file[i+6:]
		}
		return SourceLocation{n, file, line, col}
	}
	return SourceLocation{Node: n}
}

// SourceLocation returns the SoureLocation for the currently built expression,
// statement or function.
func (c *C) SourceLocation() SourceLocation {
	if e := c.CurrentExpression(); e != nil {
		return c.SourceLocationFor(e)
	} else if s := c.CurrentStatement(); s != nil {
		return c.SourceLocationFor(s)
	} else if f := c.currentFunc; f != nil {
		return c.SourceLocationFor(f)
	}
	return SourceLocation{}
}

func (c *C) setCodeLocation(s *S, t cst.Token) {
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
	if c.Settings.CodeLocations && s.Location != nil {
		s.Location.Store(s.Scalar(uint32(s.locationIdx)))
	}
}

func (c *C) setCurrentFunction(f *semantic.Function) *semantic.Function {
	old := c.currentFunc
	c.currentFunc = f
	return old
}

func (c *C) pushStatement(s *S, n semantic.Statement) {
	c.statementStack = append(c.statementStack, n)
	c.onChangeStatement(s)
}

func (c *C) popStatement(s *S) {
	c.statementStack = c.statementStack[:len(c.statementStack)-1]
	c.onChangeStatement(s)
}

func (c *C) pushExpression(s *S, n semantic.Expression) {
	c.expressionStack = append(c.expressionStack, n)
}

func (c *C) popExpression(s *S) {
	c.expressionStack = c.expressionStack[:len(c.expressionStack)-1]
}

func (c *C) onChangeStatement(s *S) {
	n := c.CurrentStatement()
	if n == nil {
		return
	}
	if cst := c.mappings.CST(n); cst != nil {
		c.setCodeLocation(s, cst.Tok())
	}
}

func (c *C) augmentPanics() {
	r := recover()
	if r == nil {
		return
	}
	panic(fmt.Errorf("Internal compiler error processing %v\n%v", c.SourceLocation(), r))
}
