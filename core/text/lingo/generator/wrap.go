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
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

const (
	runeMethod    = `Rune`
	stringMethod  = `Literal`
	patternMethod = `Pattern`
	parserName    = `Parser`
	wrapper       = `
		package exemplar
		func _name_Parser(_params_) (_type_, error) {
			_scanner_.Skip()
			mark := _scanner_.Mark()
			result, err := _call_(_args_)
			if err != nil {
				err = _scanner_.Error(err, "_name_")
				_scanner_.Reset(mark)
			} else {
				_scanner_.Register(mark, _result_)
			}
			return _result_, err
		}
	`
)

func generateFor(ctx context.Context, fset *token.FileSet, wrap *entry) {
	if !wrap.Called {
		return
	}
	if wrap.Method == patternMethod {
		wrap.Source.Decls = append(wrap.Source.Decls, &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{{Name: wrap.Name + patternMethod}},
				Values: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "regexp"},
						Sel: &ast.Ident{Name: "MustCompile"},
					},
					Args: []ast.Expr{&ast.BasicLit{
						Kind:  token.STRING,
						Value: string(wrap.Value.Value[0]) + "^" + wrap.Value.Value[1:],
					}},
				}},
			}},
		})
	}
	scanner := wrap.Params[0].Names[0].Name
	result := "result"
	if wrap.ParsedType != wrap.ResultType {
		result = wrap.ResultType + "(result)"
	}
	call := ""
	args := ""
	buf := &bytes.Buffer{}
	for _, p := range wrap.Params {
		for _, name := range p.Names {
			if args != "" {
				buf.WriteString(" ,")
				args += ", "
			}
			buf.WriteString(name.Name)
			buf.WriteString(" ")
			format.Node(buf, fset, p.Type)
			args += name.Name
		}
	}
	params := buf.String()
	if wrap.Func != nil {
		call = "_name_"
	} else {
		call = "_scanner_." + wrap.Method
		if wrap.Method == patternMethod {
			args = wrap.Name + patternMethod
		} else {
			args = wrap.Value.Value
		}
	}
	text := wrapper
	text = strings.Replace(text, "_result_", result, -1)
	text = strings.Replace(text, "_call_", call, -1)
	text = strings.Replace(text, "_args_", args, -1)
	text = strings.Replace(text, "_params_", params, -1)
	text = strings.Replace(text, "_type_", wrap.ResultType, -1)
	text = strings.Replace(text, "_scanner_", scanner, -1)
	text = strings.Replace(text, "_name_", wrap.Name, -1)
	wrapped, err := parser.ParseFile(fset, "", text, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	wrap.Source.Decls = append(wrap.Source.Decls, wrapped.Decls...)
}

func generateDefault(f *ast.File, name string, typ string) {
	f.Decls = append(f.Decls, &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{{Name: name}},
			Type:  &ast.Ident{Name: typ},
		}},
	})
}
