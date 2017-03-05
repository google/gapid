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

package resolver

import (
	"fmt"
	"strings"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

type genericSubroutine struct {
	AST  *ast.Function
	args []string
}

func newGenericSubroutine(rv *resolver, f *ast.Function) genericSubroutine {
	gs := genericSubroutine{AST: f}
	for _, a := range f.Generic.Arguments {
		arg, ok := a.(*ast.Generic)
		if !ok || len(arg.Arguments) > 0 {
			rv.errorf(a, "expected identifier, got %T", a)
		}
		gs.args = append(gs.args, arg.Name.Value)
	}
	return gs
}

func (g genericSubroutine) resolve(rv *resolver, in *ast.Generic) *semantic.Function {
	args := in.Arguments
	if req, got := len(g.AST.Generic.Arguments), len(args); req != got {
		rv.errorf(in, "Incorrect number of generic arguments for %v. Required %v, got %v",
			g.AST.Generic.Name.Value, req, got)
	}
	tys := make(map[string]semantic.Type, len(args))
	tynames := make([]string, len(args))
	for i, a := range args {
		ty := type_(rv, a)
		tys[g.args[i]], tynames[i] = ty, ty.Name()
	}
	name := fmt.Sprintf("%v!%v", g.AST.Generic.Name.Value, strings.Join(tynames, ":"))
	existing := rv.disambiguate(rv.find(name))
	switch len(existing) {
	case 0:
		s := &semantic.Function{AST: g.AST, Named: semantic.Named(name), Subroutine: true}
		rv.api.Subroutines = append(rv.api.Subroutines, s)
		rv.globals.Add(name, s)
		semantic.Add(rv.api, s)

		original := rv.scope
		defer func() { rv.scope = original }()
		rv.scope = &scope{
			types:    map[string]semantic.Type{},
			outer:    rv.globals,
			block:    rv.globals.block,
			function: rv.globals.function,
		}

		rv.with(semantic.VoidType, func() {
			for n, ty := range tys {
				rv.addGenericParameter(n, ty)
			}
			functionSignature(rv, s)
			for _, p := range s.FullParameters {
				if p.Type == semantic.AnyType {
					rv.errorf(p, "cannot use any as parameter type on subroutines")
				}
			}
			functionBody(rv, nil, s)
		})

		extractCalls(rv, s.Block)
		removeDeadCode(rv, s.Block)
		return s

	case 1:
		f, ok := existing[0].(*semantic.Function)
		if !ok {
			rv.icef(in, "%v resolved to %T when expected function", name, existing)
		}
		return f

	default:
		rv.ambiguousIdentifier(in, existing)
		return nil
	}
}
