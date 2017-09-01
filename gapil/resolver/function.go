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
	"bytes"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func functionSignature(rv *resolver, out *semantic.Function) {
	in := out.AST
	if !out.Subroutine && len(in.Generic.Arguments) > 0 {
		rv.errorf(in.Generic.Arguments[0], "generic parameters are not supported here")
	}
	args := make([]semantic.Type, 0, len(in.Parameters)-1)
	out.FullParameters = make([]*semantic.Parameter, 0, len(in.Parameters))
	for i, inp := range in.Parameters {
		outp := parameter(rv, out, inp)
		if inp.This {
			if i == 0 {
				out.This = outp
			} else {
				rv.errorf(inp, "this only allowed on arg 0")
			}
		}
		if i < len(in.Parameters)-1 {
			if isVoid(outp.Type) {
				rv.errorf(in, "void typed parameter %s on function %s", outp.Name(), out.Name())
			}
			if !inp.This {
				args = append(args, outp.Type)
			}
			out.FullParameters = append(out.FullParameters, outp)
		} else {
			out.Return = outp
			if !isVoid(outp.ExpressionType()) {
				out.Return.Named = semantic.Named("result")
				out.FullParameters = append(out.FullParameters, outp)
			}
		}
	}
	out.Signature = getSignature(rv, in, out.Return.Type, args)
	out.Annotations = annotations(rv, in.Annotations)
	rv.mappings.add(in, out)
	rv.mappings.add(in.Generic.Name, out)
}

func parameter(rv *resolver, owner *semantic.Function, in *ast.Parameter) *semantic.Parameter {
	out := &semantic.Parameter{
		AST:      in,
		Function: owner,
	}
	if in.Name != nil {
		out.Named = semantic.Named(in.Name.Value)
	}
	out.Docs = rv.findDocumentation(in)
	out.Annotations = annotations(rv, in.Annotations)
	out.Type = type_(rv, in.Type)
	rv.mappings.add(in, out)
	rv.mappings.add(in.Name, out)
	return out
}

func functionBody(rv *resolver, owner semantic.Type, out *semantic.Function) {
	in := out.AST
	if owner != nil {
		semantic.Add(owner, out)
	}
	if in.Block != nil {
		rv.with(semantic.VoidType, func() {
			rv.scope.function = out
			for _, p := range out.FullParameters {
				rv.addNamed(p)
			}
			if out.This != nil {
				rv.add(string(ast.KeywordThis), out.This)
			}
			out.Block = block(rv, in.Block, out)
			if out.Subroutine {
				switch out.Block.Statements.Last().(type) {
				case *semantic.Return, *semantic.Abort:
				default:
					// Subroutines must end with a return statement.
					if out.Return.Type != semantic.VoidType {
						rv.icef(out.Return.AST, "Expected return statement as last statement of subroutine.")
					}
					out.Block.Statements.Append(&semantic.Return{Function: out})
				}
			}
		})
	}
	if len(out.Docs) == 0 {
		out.Docs = rv.findDocumentation(out.AST)
	}
	rv.mappings.add(in, out)
}

func method(rv *resolver, out *semantic.Function) {
	functionSignature(rv, out)
	t := out.This.Type
	switch t := t.(type) {
	case *semantic.Pointer:
		if class, ok := t.To.(*semantic.Class); !ok {
			rv.errorf(out.AST, "expected this as a reference to a class, got %s[%T]", typename(t.To), t.To)
		} else {
			class.Methods = append(class.Methods, out)
			semantic.Add(class, out)
			functionBody(rv, class, out)
		}
	case *semantic.Pseudonym:
		t.Methods = append(t.Methods, out)
		semantic.Add(t, out)
		functionBody(rv, t, out)
	case *semantic.Class:
		t.Methods = append(t.Methods, out)
		semantic.Add(t, out)
		functionBody(rv, t, out)
	default:
		rv.errorf(out.AST, "invalid type for this , got %s[%T]", typename(t), t)
	}
	rv.mappings.add(out.AST, out)
}

func getSignature(rv *resolver, at ast.Node, r semantic.Type, args []semantic.Type) *semantic.Signature {
	buffer := bytes.Buffer{}
	buffer.WriteString("fun_")
	buffer.WriteString(r.Name())
	buffer.WriteString("_")
	for _, a := range args {
		buffer.WriteString("_")
		buffer.WriteString(a.Name())
	}
	name := buffer.String()
	for _, s := range rv.api.Signatures {
		if s.Name() == name {
			if !equal(r, s.Return) {
				rv.icef(at, "Signature %s found with non matching return type, got %s expected %s", name, typename(s.Return), typename(r))
			}
			if len(args) != len(s.Arguments) {
				rv.icef(at, "Signature %s found with %d arguments, expected %s", name, len(s.Arguments), len(args))
			}
			for i, a := range args {
				if !equal(a, s.Arguments[i]) {
					rv.icef(at, "Signature %s found with non matching arg at %d, got %s expected %s", name, i, typename(s.Arguments[i]), typename(a))
				}
			}
			return s
		}
	}
	out := &semantic.Signature{
		Named:     semantic.Named(name),
		Return:    r,
		Arguments: args,
	}
	rv.api.Signatures = append(rv.api.Signatures, out)
	return out
}
