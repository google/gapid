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

	plugins     plugins
	commands    map[*semantic.Function]*codegen.Function
	externs     map[*semantic.Function]*codegen.Function
	subroutines map[*semantic.Function]*codegen.Function
	functions   map[string]*codegen.Function
	ctx         struct { // Functions that operate on contexts
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
	refRels         refRels
	currentAPI      *semantic.API
	currentFunc     *semantic.Function
	statementStack  []semantic.Statement
	expressionStack []semantic.Expression
	isFence         bool // If true, a fence should be emitted for the given statement
	callbacks       struct {
		alloc          *codegen.Function
		anyReference   *codegen.Function
		anyRelease     *codegen.Function
		applyReads     *codegen.Function
		applyWrites    *codegen.Function
		callExtern     *codegen.Function
		copySlice      *codegen.Function
		cstringToSlice *codegen.Function
		free           *codegen.Function
		freePool       *codegen.Function
		freeString     *codegen.Function
		logf           *codegen.Function
		makePool       *codegen.Function
		makeString     *codegen.Function
		msgReference   *codegen.Function
		msgRelease     *codegen.Function
		realloc        *codegen.Function
		sliceData      *codegen.Function
		sliceToString  *codegen.Function
		stringCompare  *codegen.Function
		stringConcat   *codegen.Function
		stringToSlice  *codegen.Function
	}
	module codegen.Global
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

	c := &C{
		M:        codegen.NewModule("api.executor", s.TargetABI),
		APIs:     apis,
		Mangler:  s.Mangler,
		Settings: s,

		plugins:     s.Plugins,
		commands:    map[*semantic.Function]*codegen.Function{},
		externs:     map[*semantic.Function]*codegen.Function{},
		subroutines: map[*semantic.Function]*codegen.Function{},
		functions:   map[string]*codegen.Function{},
		mappings:    mappings,
	}

	if s.EmitDebug {
		c.M.EmitDebug()
	}

	for _, n := range s.Namespaces {
		c.Root = &mangling.Namespace{Name: n, Parent: c.Root}
	}

	c.compile()

	prog, err := c.program(s)
	if err != nil {
		return nil, err
	}

	if err := c.M.Verify(); err != nil {
		return nil, err
	}

	return prog, nil
}

func (c *C) program(s Settings) (*Program, error) {
	commands := make(map[string]*CommandInfo, len(c.commands))
	for a, f := range c.commands {
		commands[a.Name()] = &CommandInfo{
			Execute:    f,
			Parameters: c.T.CmdParams[a],
		}
	}

	globals := &StructInfo{Type: c.T.Globals}

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
		Settings:  c.Settings,
		APIs:      c.APIs,
		Commands:  commands,
		Structs:   structs,
		Globals:   globals,
		Functions: c.functions,
		Maps:      maps,
		Codegen:   c.M,
		Module:    c.module,
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
	c.declareMangling()
	c.declareCallbacks()
	c.declareBufferFuncs()
	c.declareContextType()
	c.declareRefRels()

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

	c.plugins.foreach(func(p FunctionExposerPlugin) {
		for n, f := range p.Functions() {
			c.functions[n] = f
		}
	})

	c.buildModule()
}

func (c *C) declareCallbacks() {
	c.callbacks.alloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_alloc_sig))
	c.callbacks.anyReference = c.M.ParseFunctionSignature(C.GoString(C.gapil_any_reference_sig))
	c.callbacks.anyRelease = c.M.ParseFunctionSignature(C.GoString(C.gapil_any_release_sig))
	c.callbacks.applyReads = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_reads_sig))
	c.callbacks.applyWrites = c.M.ParseFunctionSignature(C.GoString(C.gapil_apply_writes_sig))
	c.callbacks.callExtern = c.M.ParseFunctionSignature(C.GoString(C.gapil_call_extern_sig))
	c.callbacks.copySlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_copy_slice_sig))
	c.callbacks.cstringToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_cstring_to_slice_sig))
	c.callbacks.free = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_sig))
	c.callbacks.freePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_pool_sig))
	c.callbacks.freeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_free_string_sig))
	c.callbacks.logf = c.M.ParseFunctionSignature(C.GoString(C.gapil_logf_sig))
	c.callbacks.makePool = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_pool_sig))
	c.callbacks.makeString = c.M.ParseFunctionSignature(C.GoString(C.gapil_make_string_sig))
	c.callbacks.msgReference = c.M.ParseFunctionSignature(C.GoString(C.gapil_msg_reference_sig))
	c.callbacks.msgRelease = c.M.ParseFunctionSignature(C.GoString(C.gapil_msg_release_sig))
	c.callbacks.realloc = c.M.ParseFunctionSignature(C.GoString(C.gapil_realloc_sig))
	c.callbacks.sliceData = c.M.ParseFunctionSignature(C.GoString(C.gapil_slice_data_sig))
	c.callbacks.sliceToString = c.M.ParseFunctionSignature(C.GoString(C.gapil_slice_to_string_sig))
	c.callbacks.stringCompare = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_compare_sig))
	c.callbacks.stringConcat = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_concat_sig))
	c.callbacks.stringToSlice = c.M.ParseFunctionSignature(C.GoString(C.gapil_string_to_slice_sig))
}

// Build implements the function f by creating a new scope and calling do to
// emit the function body.
// If the function has a parameter of type context_t* then the Ctx, Globals and
// Arena scope fields are automatically assigned.
func (c *C) Build(f *codegen.Function, do func(*S)) {
	err(f.Build(func(b *codegen.Builder) {
		s := &S{
			Builder:    b,
			Parameters: map[*semantic.Parameter]*codegen.Value{},
			locals:     map[*semantic.Local]local{},
		}
		for i, p := range f.Type.Signature.Parameters {
			if p == c.T.CtxPtr {
				s.Ctx = b.Parameter(i).SetName("ctx")
				if debugCtxNotNull {
					s.If(s.Ctx.IsNull(), func(s *S) {
						c.LogF(s, "Context is null")
					})
				}
				s.Globals = s.Ctx.Index(0, ContextGlobals).Load().
					SetName("globals").
					EmitDebug("globals")
				s.Arena = s.Ctx.Index(0, ContextArena).Load().
					SetName("arena").
					EmitDebug("arena")
				s.CurrentThread = s.Ctx.Index(0, ContextThread).Load().
					SetName("thread").
					EmitDebug("$Thread")
				break
			}
		}

		do(s)
		s.exit()
	}))
}

// MakeSlice creates a new slice of the given size in bytes.
func (c *C) MakeSlice(s *S, size, count *codegen.Value) *codegen.Value {
	slice := s.Local("slice", c.T.Sli)
	c.MakeSliceAt(s, size, count, slice)
	return slice.Load()
}

// MakeSliceAt creates a new slice of the given size in bytes at the given
// slice pointer.
func (c *C) MakeSliceAt(s *S, size, count, slice *codegen.Value) {
	pool := s.Call(c.callbacks.makePool, s.Ctx, size)
	slice.Index(0, SlicePool).Store(pool)
	slice.Index(0, SliceRoot).Store(s.Scalar(uint64(0)))
	slice.Index(0, SliceBase).Store(s.Scalar(uint64(0)))
	slice.Index(0, SliceSize).Store(size)
	slice.Index(0, SliceCount).Store(count)
}

// CopySlice copies the contents of slice src to dst.
func (c *C) CopySlice(s *S, dst, src *codegen.Value) {
	pDst := s.LocalInit("dstPtr", dst)
	pSrc := s.LocalInit("srcPtr", src)
	c.CopySliceByPtr(s, pDst, pSrc)
}

// CopySliceByPtr copies the contents of slice at *src to the slice at *dst.
func (c *C) CopySliceByPtr(s *S, dst, src *codegen.Value) {
	s.Call(c.callbacks.copySlice, s.Ctx, dst, src)
}

// SliceDataForRead returns a pointer to an array of slice elements.
// This pointer should be used to read (not write) from the slice.
// The pointer is only valid until the slice is touched again.
func (c *C) SliceDataForRead(s *S, slice *codegen.Value, elType codegen.Type) *codegen.Value {
	access := s.Scalar(Read).Cast(c.T.DataAccess)
	return s.Call(c.callbacks.sliceData, s.Ctx, slice, access).Cast(c.T.Pointer(elType))
}

// SliceDataForWrite returns a pointer to an array of slice elements.
// This pointer should be used to write (not read) to the slice.
// The pointer is only valid until the slice is touched again.
func (c *C) SliceDataForWrite(s *S, slice *codegen.Value, elType codegen.Type) *codegen.Value {
	access := s.Scalar(Write).Cast(c.T.DataAccess)
	return s.Call(c.callbacks.sliceData, s.Ctx, slice, access).Cast(c.T.Pointer(elType))
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

	// Transform all args into codegen Values (if they're not already)
	vals := make([]*codegen.Value, len(args))
	for i, arg := range args {
		v, ok := arg.(*codegen.Value)
		if !ok {
			v = s.Scalar(arg)
		}
		vals[i] = v
	}

	// Substitute any %v's with their expanded form
	msg, vals = substitutePercentVs(s, msg, vals)

	loc := c.SourceLocation()
	fullArgs := append([]*codegen.Value{
		s.Scalar(uint8(severity)),
		s.Scalar(loc.File),
		s.Scalar(uint32(loc.Line)),
		s.Scalar(msg),
	}, vals...)
	s.Call(c.callbacks.logf, fullArgs...)
}

// substitutePercentVs returns the transformed printf fmt message and values,
// replacing any '%v's with the correct specifier(s) for the given value type.
func substitutePercentVs(s *S, fmt string, vals []*codegen.Value) (string, []*codegen.Value) {
	inRunes := ([]rune)(fmt)
	outRunes := make([]rune, 0, len(inRunes))
	outVals := make([]*codegen.Value, 0, len(vals))
	for i, c := 0, len(inRunes); i < c; i++ {
		r := inRunes[i]
		if r == '%' && i <= len(inRunes)-1 {
			n := inRunes[i+1]
			switch n {
			case 'v':
				f, v := s.PrintfSpecifier(vals[0])
				vals = vals[1:]
				outRunes = append(outRunes, ([]rune)(f)...)
				outVals = append(outVals, v...)
			case '%':
				outRunes = append(outRunes, '%', '%')
			default:
				outRunes = append(outRunes, '%', n)
				outVals = append(outVals, vals[0])
				vals = vals[1:]
			}
			i++ // Skip the consumed character following the %
		} else {
			outRunes = append(outRunes, r)
		}
	}
	if len(vals) != 0 {
		fail("Log message has %v unconsumed values. Message: '%v'", len(vals), fmt)
	}
	return string(outRunes), outVals
}

// LogI is short hand for Log(s, log.Info, msg, args...)
func (c *C) LogI(s *S, msg string, args ...interface{}) {
	c.Log(s, log.Info, msg, args...)
}

// LogF is short hand for Log(s, log.Fatal, msg, args...)
func (c *C) LogF(s *S, msg string, args ...interface{}) {
	c.Log(s, log.Fatal, msg, args...)
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

// IsValid returns true if the source location is valid.
func (l SourceLocation) IsValid() bool {
	return l.File != "" && l.Line > 0
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
		l := c.SourceLocationForCST(cst)
		l.Node = n
		return l
	}
	return SourceLocation{Node: n}
}

// SourceLocationForCST returns a string describing the source location of the
// given CST node.
func (c *C) SourceLocationForCST(n cst.Node) SourceLocation {
	tok := n.Tok()
	line, col := tok.Cursor()
	file := tok.Source.Filename
	if i := strings.LastIndex(file, "gapid/"); i > 0 {
		file = file[i+6:]
	}
	return SourceLocation{nil, file, line, col}
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
	if n := c.CurrentStatement(); n != nil {
		loc := c.SourceLocationFor(n)
		s.SetLocation(loc.Line, loc.Column)
	}
}

func (c *C) augmentPanics() {
	r := recover()
	if r == nil {
		return
	}
	panic(fmt.Errorf("Internal compiler error processing %v\n%v", c.SourceLocation(), r))
}
