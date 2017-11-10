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

package generator

import (
	"context"
	"go/ast"
	"go/token"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/log"
)

type introspection struct {
	fset    *token.FileSet
	entries []*entry
	byName  map[string]*entry
}

type entry struct {
	// Source is the source file the declaration came from.
	Source *ast.File
	// Name is the function name being wrapped.
	Name string
	// IsPointer is true if the return type is a pointer.
	// This is used to control whether nil or a default value is used for error returns.
	IsPointer bool
	// ResultType is the stringified type declaration for the primary return type of the function.
	ResultType string
	// ParsedType is the underlying return type. This is only different from the result type in the
	//  presenced of type aliases, when a type cast may have to be inserted in the generated functions.
	ParsedType string
	// Params is the extracted parameters to the funciton.
	Params []*ast.Field
	// Public is true if this function is exposed outside the package.
	Public bool
	// Parser is true if this is detected to be a parsing funciton that wrapping.
	Parser bool
	// Called is true if the function is actually used.
	Called bool
	// DefaultName is name of the package global that holds the default value for the function return type
	DefaultName string
	// Func is the AST for the function being rewritten.
	Func *ast.FuncDecl
	// Value is the value declaration when constant as function rewrites are occuring.
	Value *ast.BasicLit
	// Method is the scanner method to invoke when constant as parser rewrites are occuring.
	Method string
}

func (info *introspection) collectEntryPoints(ctx context.Context, f *ast.File) {
	// Find all the functions and constants.
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			info.entries = append(info.entries, &entry{
				Source: f,
				Func:   decl,
				Name:   decl.Name.Name,
			})
		case *ast.GenDecl:
			if decl.Tok != token.CONST {
				break
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.ValueSpec)
				for i, name := range spec.Names {
					e := buildConstEntry(spec.Type, spec.Values[i])
					if e != nil {
						e.Source = f
						e.Name = name.Name
						info.entries = append(info.entries, e)
					}
				}
			}
		}
	}
}

func buildConstEntry(t ast.Expr, v ast.Expr) *entry {
	e := &entry{}
	e.Parser = true
	switch v := v.(type) {
	case *ast.BasicLit:
		e.Value = v
	case *ast.CallExpr:
		if len(v.Args) != 1 {
			return nil
		}
		e.ResultType = v.Fun.(*ast.Ident).Name
		litArg, ok := v.Args[0].(*ast.BasicLit)
		if !ok {
			return nil
		}
		e.Value = litArg
	default:
		return nil
	}
	switch e.Value.Kind {
	case token.CHAR:
		e.ParsedType = "rune"
		e.Method = runeMethod
	case token.STRING:
		switch e.Value.Value[0] {
		case '"':
			e.Method = stringMethod
		case '`':
			e.Method = patternMethod
		}
	default:
		return nil
	}
	if e.ResultType == "" {
		e.ResultType = e.ParsedType
	}
	return e
}

func (info *introspection) prepare(ctx context.Context) {
	params := []*ast.Field{}
	// Detect the rewritable functions
	for _, e := range info.entries {
		info.byName[e.Name] = e
		if e.Func == nil {
			// not a function type
			continue
		}
		e.Parser = true
		e.Params = e.Func.Type.Params.List
		if r, _ := utf8.DecodeRuneInString(e.Name); unicode.IsUpper(r) {
			e.Parser = false
		}
		if !collectReturnType(ctx, e) {
			e.Parser = false
		}
		if len(e.Params) < 1 || len(e.Params[0].Names) < 1 {
			e.Parser = false
		}
		if len(params) == 0 && e.Parser && len(e.Params) == 1 && len(e.Params[0].Names) == 1 {
			params = e.Params
		}
	}
	// Set the default params list now we have found it
	for _, e := range info.entries {
		if e.Params == nil {
			e.Params = params
		}
	}
}

func collectReturnType(ctx context.Context, e *entry) bool {
	if e.Func.Type.Results == nil {
		return false
	}
	if len(e.Func.Type.Results.List) != 2 {
		return false
	}
	if !collectTypeName(ctx, e, e.Func.Type.Results.List[0].Type, true) {
		return false
	}
	e.ResultType = e.ParsedType
	return true
}

func collectTypeName(ctx context.Context, e *entry, expr ast.Expr, top bool) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		e.ParsedType = t.Name
		return true
	case *ast.SelectorExpr:
		if !collectTypeName(ctx, e, t.X, false) {
			return false
		}
		e.ParsedType = e.ParsedType + "." + t.Sel.Name
		return true
	case *ast.StarExpr:
		if !top {
			log.F(ctx, true, "Can't cope with non simple pointers (%v)", expr)
			return false
		}
		if !collectTypeName(ctx, e, t.X, false) {
			return false
		}
		e.ParsedType = "*" + e.ParsedType
		return true
	default:
		log.F(ctx, true, "Unknown type %T", t)
		return false
	}
}
