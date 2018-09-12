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

package codegen

import (
	"debug/dwarf"
	"fmt"
	"llvm/bindings/go/llvm"
	"path/filepath"
)

// dbg holds debug info generation data.
type dbg struct {
	m     *Module
	cu    llvm.Metadata
	llvm  *llvm.DIBuilder
	files map[string]file
	tys   map[Type]llvm.Metadata
}

// builder constructs and returns a new llvm debug info builder for the module.
func (d *dbg) builder() *llvm.DIBuilder {
	if d.llvm != nil {
		return d.llvm
	}

	b := llvm.NewDIBuilder(d.m.llvm)
	d.cu = b.CreateCompileUnit(llvm.DICompileUnit{
		Language:       0x0001, // DW_LANG_C,
		File:           "apis",
		Dir:            "/ssd/src/gapid", // TODO: HACK
		Producer:       "gapid",
		RuntimeVersion: 1,
	})
	d.llvm = b

	// Build the basic types.
	tys := d.m.Types
	d.tys[tys.Void] = b.CreateBasicType(llvm.DIBasicType{Name: "void"})
	for _, ty := range []Type{tys.Int, tys.Int8, tys.Int16, tys.Int32, tys.Int64} {
		d.tys[ty] = b.CreateBasicType(llvm.DIBasicType{
			Name:       ty.TypeName(),
			SizeInBits: uint64(ty.SizeInBits()),
			Encoding:   llvm.DW_ATE_signed,
		})
	}
	for _, ty := range []Type{tys.Bool, tys.Uint, tys.Uint8, tys.Uint16, tys.Uint32, tys.Uint64, tys.Uintptr, tys.Size} {
		d.tys[ty] = b.CreateBasicType(llvm.DIBasicType{
			Name:       ty.TypeName(),
			SizeInBits: uint64(ty.SizeInBits()),
			Encoding:   llvm.DW_ATE_unsigned,
		})
	}
	for _, ty := range []Type{tys.Float32, tys.Float64} {
		d.tys[ty] = b.CreateBasicType(llvm.DIBasicType{
			Name:       ty.TypeName(),
			SizeInBits: uint64(ty.SizeInBits()),
			Encoding:   llvm.DW_ATE_float,
		})
	}
	return b
}

func (d *dbg) finalize() {
	if d != nil && d.llvm != nil {
		d.llvm.Finalize()
	}
}

// file returns a debug scope for the given file path.
func (d *dbg) file(path string) file {
	if existing, ok := d.files[path]; ok {
		return existing
	}
	dir, name := filepath.Split(path)
	file := file{path, d.builder().CreateFile(name, dir)}
	d.files[path] = file
	return file
}

// ty returns the llvm debug type for the given codegen type t.
func (d *dbg) ty(t Type) (out llvm.Metadata) {
	if existing, ok := d.tys[t]; ok {
		return existing
	}
	b := d.builder()

	defer func() {
		d.tys[t] = out
	}()

	switch t := t.(type) {
	case Pointer:
		return b.CreatePointerType(llvm.DIPointerType{
			Pointee:     d.ty(t.Element),
			SizeInBits:  uint64(t.SizeInBits()),
			AlignInBits: uint32(t.AlignInBits()),
		})
	case *Array:
		return b.CreateArrayType(llvm.DIArrayType{
			SizeInBits:  uint64(t.SizeInBits()),
			AlignInBits: uint32(t.AlignInBits()),
			ElementType: d.ty(t.Element),
			Subscripts:  []llvm.DISubrange{{Count: int64(t.Size)}},
		})
	case *FunctionType:
		ty := llvm.DISubroutineType{
			// TODO: File
			Parameters: make([]llvm.Metadata, len(t.Signature.Parameters)+1),
		}
		if t.Signature.Result != d.m.Types.Void {
			ty.Parameters[0] = d.ty(t.Signature.Result)
		}
		for i, t := range t.Signature.Parameters {
			ty.Parameters[i+1] = d.ty(t)
		}
		return b.CreateSubroutineType(ty)
	case *Struct:
		placeholder := b.CreateReplaceableCompositeType(d.cu, llvm.DIReplaceableCompositeType{
			Tag:         dwarf.TagStructType,
			SizeInBits:  uint64(t.SizeInBits()),
			AlignInBits: uint32(t.AlignInBits()),
			Name:        t.TypeName(),
			File:        d.cu,
		})
		d.tys[t] = placeholder
		defer func() { placeholder.ReplaceAllUsesWith(out) }()

		fields := t.Fields()
		members := make([]llvm.Metadata, len(fields))
		for i, f := range fields {
			members[i] = b.CreateMemberType(d.cu, llvm.DIMemberType{
				Name:         f.Name,
				Type:         d.ty(f.Type),
				SizeInBits:   uint64(f.Type.SizeInBits()),
				AlignInBits:  uint32(f.Type.AlignInBits()),
				OffsetInBits: uint64(t.FieldOffsetInBits(i)),
			})
		}
		return b.CreateStructType(d.cu, llvm.DIStructType{
			Name:        t.TypeName(),
			SizeInBits:  uint64(t.SizeInBits()),
			AlignInBits: uint32(t.AlignInBits()),
			Elements:    members,
		})
	default:
		fail("Unhandled type %T", t)
		return llvm.Metadata{}
	}
}

// SetLocation sets the source location of the given function.
// SetLocation returns f so it can be called fluently.
func (f *Function) SetLocation(path string, line int) *Function {
	d := f.m.dbg
	if d == nil {
		return f
	}
	file := d.file(path)
	b := d.builder()
	dif := b.CreateFunction(file.llvm, llvm.DIFunction{
		Name:         f.Name,
		LinkageName:  f.Name,
		File:         file.llvm,
		Line:         line,
		Type:         d.ty(f.Type),
		IsDefinition: true,
	})
	f.llvm.SetSubprogram(dif)
	f.dbg = &funcDbg{
		scope:      dif,
		file:       file,
		declLine:   line,
		declColumn: 0,
	}
	return f
}

// SetLocation sets the source location of the next instructions to be built.
func (b *Builder) SetLocation(line, column int) {
	if f := b.function.dbg; f != nil {
		b.llvm.SetCurrentDebugLocation(uint(line), uint(column), f.scope, llvm.Metadata{})
		f.curLine, f.curColumn = line, column
	}
}

func (b *Builder) dbgRestoreLocation() {
	if d := b.function.dbg; d != nil {
		b.SetLocation(d.curLine, d.curColumn)
	}
}

func (b *Builder) dbgEmitParameters(bb llvm.BasicBlock) {
	if b.function.dbg == nil {
		return
	}
	d := b.m.dbg.llvm
	// Create a debug descriptor for the variable.
	for i, param := range b.params {
		name := fmt.Sprintf("param_%d", i)
		if i < len(b.function.paramNames) {
			name = b.function.paramNames[i]
		}
		loc := llvm.DebugLoc{
			Scope: b.function.dbg.scope,
			Line:  uint(b.function.dbg.declLine),
			Col:   uint(b.function.dbg.declColumn),
		}
		v := d.CreateParameterVariable(b.function.dbg.scope,
			llvm.DIParameterVariable{
				Name:           name,
				File:           b.function.dbg.file.llvm,
				Line:           int(loc.Line),
				Type:           b.m.dbg.ty(param.Type()),
				AlwaysPreserve: true,
				ArgNo:          1 + i,
			},
		)
		d.InsertValueAtEnd(param.llvm, v, d.CreateExpression(nil), loc, bb)
	}
}

func (b *Builder) dbgEmitValue(val *Value, name string) {
	if b.function.dbg == nil {
		return
	}
	d := b.m.dbg.llvm
	isAlloca := val.llvm.IsAAllocaInst().C != nil
	loc := llvm.DebugLoc{
		Scope: b.function.dbg.scope,
		Line:  uint(b.function.dbg.curLine),
		Col:   uint(b.function.dbg.curColumn),
	}
	elTy := val.Type()
	if isAlloca {
		elTy = elTy.(Pointer).Element
	}
	v := d.CreateAutoVariable(b.function.dbg.scope,
		llvm.DIAutoVariable{
			Name:        name,
			File:        b.function.dbg.file.llvm,
			Line:        b.function.dbg.curLine,
			Type:        b.m.dbg.ty(elTy),
			AlignInBits: uint32(elTy.AlignInBits()),
		},
	)
	if isAlloca {
		d.InsertDeclareAtEnd(val.llvm, v, d.CreateExpression(nil), loc, b.llvm.GetInsertBlock())
	} else {
		d.InsertValueAtEnd(val.llvm, v, d.CreateExpression(nil), loc, b.llvm.GetInsertBlock())
	}
}

type funcDbg struct {
	scope                llvm.Metadata
	file                 file
	declLine, declColumn int
	curLine, curColumn   int
}

type file struct {
	path string
	llvm llvm.Metadata
}
