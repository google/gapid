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
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func block(rv *resolver, in *ast.Block, owner semantic.Node) *semantic.Block {
	out := &semantic.Block{AST: in}
	if in == nil {
		return out
	}
	rv.with(semantic.VoidType, func() {
		rv.scope.block = &out.Statements
		r := body(rv, in.Statements, owner)
		if r != nil {
			rv.addStatement(r)
		}
	})
	rv.mappings.Add(in, out)
	return out
}

// body is a resolve function that processes a list of statements and injects them
// into the context's current block.
// the final return statement, if present, is not injected, but returned from the
// function, as it often needs special handling depending on the owner of the
// statements
func body(rv *resolver, in []ast.Node, owner semantic.Node) *semantic.Return {
	f, isFunction := owner.(*semantic.Function)
	var returnStatement *ast.Return
	// we need to check and strip the "return" if the function is supposed to have one
	if isFunction && !isVoid(f.Return.Type) {
		if len(in) == 0 {
			rv.errorf(f.AST, "Missing return statement")
		} else if r, ok := in[len(in)-1].(*ast.Return); !ok {
			rv.errorf(f.AST, "Last statement must be a return")
		} else {
			in = in[0 : len(in)-1]
			returnStatement = r
		}
	}
	// now process the non return statements
	for _, s := range in {
		rv.addStatement(statement(rv, s))
	}
	// and special case the return statement allowing access to the return parameter
	if returnStatement != nil {
		out := return_(rv, returnStatement, f)
		rv.mappings.Add(returnStatement, out)
		return out
	}
	return nil
}

func statement(rv *resolver, in ast.Node) semantic.Statement {
	var out semantic.Statement
	switch in := in.(type) {
	case *ast.Assign:
		out = assign(rv, in)
	case *ast.DeclareLocal:
		out = declareLocal(rv, in)
	case *ast.Delete:
		out = delete_(rv, in)
	case *ast.Clear:
		out = clear_(rv, in)
	case *ast.Branch:
		out = branch(rv, in)
	case *ast.Switch:
		out = switch_(rv, in)
	case *ast.Iteration:
		out = iteration(rv, in)
	case *ast.MapIteration:
		out = mapIteration(rv, in)
	case *ast.Call:
		c := call(rv, in)
		if e, ok := c.(semantic.Expression); ok {
			if ty := e.ExpressionType(); !isVoid(ty) && !isInvalid(ty) {
				rv.errorf(in, "function with return type as statement not allowed")
				return semantic.Invalid{}
			}
		}
		s, ok := c.(semantic.Statement)
		if !ok {
			rv.errorf(in, "expected statement, got %T", c)
			return semantic.Invalid{}
		}
		out = s
	case *ast.Return:
		rv.errorf(in, "unexpected return")
		return semantic.Invalid{}
	case *ast.Abort:
		out = &semantic.Abort{AST: in, Function: rv.scope.function}
	case *ast.Fence:
		out = &semantic.Fence{AST: in, Explicit: true}
	case *ast.Generic, *ast.Member:
		rv.errorf(in, "incomplete statement")
		out = semantic.Invalid{Partial: expression(rv, in)}
	case *ast.Invalid:
		out = semantic.Invalid{}
	default:
		rv.errorf(in, "not a statement (%T)", in)
		out = semantic.Invalid{}
	}
	rv.mappings.Add(in, out)
	return out
}

func assign(rv *resolver, in *ast.Assign) semantic.Statement {
	lhs := expression(rv, in.LHS)
	var rhs semantic.Expression
	rv.with(lhs.ExpressionType(), func() {
		rhs = expression(rv, in.RHS)
	})
	var out semantic.Statement
	inferUnknown(rv, lhs, rhs)
	lt := lhs.ExpressionType()
	rt := rhs.ExpressionType()
	if !assignable(lt, rt) {
		rv.errorf(in, "cannot assign %s to %s", typename(rt), typename(lt))
	}
	switch lhs := lhs.(type) {
	case semantic.Invalid:
		out = semantic.Invalid{}
	case *semantic.ArrayIndex:
		out = &semantic.ArrayAssign{AST: in, To: lhs, Value: rhs, Operator: in.Operator}
	case *semantic.MapIndex:
		out = &semantic.MapAssign{AST: in, To: lhs, Value: rhs, Operator: in.Operator}
	case *semantic.SliceIndex:
		out = &semantic.SliceAssign{AST: in, To: lhs, Value: rhs, Operator: in.Operator}
	case *semantic.Global, *semantic.Ignore, *semantic.Member:
		out = &semantic.Assign{AST: in, LHS: lhs, Operator: in.Operator, RHS: rhs}
	case *semantic.Local:
		rv.errorf(in, "Cannot assign to '%v' - locals are immutable", lhs.Name())
	default:
		rv.icef(in, "Unexpected LHS type for assignment: %T", lhs)
	}
	if out == nil {
		out = &semantic.Assign{AST: in, LHS: lhs, Operator: in.Operator, RHS: rhs}
	}
	rv.mappings.Add(in, out)
	return out
}

func delete_(rv *resolver, in *ast.Delete) *semantic.MapRemove {
	k := semantic.Expression(semantic.Invalid{})
	m := expression(rv, in.Map)
	mty, ok := m.ExpressionType().(*semantic.Map)
	if ok {
		rv.with(mty.KeyType, func() {
			k = expression(rv, in.Key)
		})
		if !comparable(k.ExpressionType(), mty.KeyType) {
			rv.errorf(in.Key, "Cannot use %s as key to %s",
				typename(k.ExpressionType()), typename(m.ExpressionType()))
		}
	} else {
		rv.errorf(in.Map, "delete's first argument must be a map, got %s", typename(m.ExpressionType()))
	}
	return &semantic.MapRemove{AST: in, Type: mty, Map: m, Key: k}
}

func clear_(rv *resolver, in *ast.Clear) *semantic.MapClear {
	m := expression(rv, in.Map)
	mty, ok := m.ExpressionType().(*semantic.Map)
	if !ok {
		rv.errorf(in.Map, "clear's argument must be a map, got %s", typename(m.ExpressionType()))
	}
	out := &semantic.MapClear{AST: in, Type: mty, Map: m}
	rv.mappings.Add(in, out)
	return out
}

func addLocal(rv *resolver, in *ast.DeclareLocal, name string, value semantic.Expression) *semantic.DeclareLocal {
	out := &semantic.DeclareLocal{AST: in}
	out.Local = &semantic.Local{
		Declaration: out,
		Named:       semantic.Named(name),
		Value:       value,
		Type:        value.ExpressionType(),
	}
	if isVoid(out.Local.Type) {
		rv.errorf(in, "void in local declaration")
	}
	rv.addNamed(out.Local)
	if in != nil {
		rv.mappings.Add(in, out)
		rv.mappings.Add(in.Name, out.Local)
	}
	return out
}

func declareLocal(rv *resolver, in *ast.DeclareLocal) *semantic.DeclareLocal {
	out := addLocal(rv, in, in.Name.Value, expression(rv, in.RHS))
	rv.mappings.Add(in, out)
	return out
}

func branch(rv *resolver, in *ast.Branch) *semantic.Branch {
	out := &semantic.Branch{AST: in}
	out.Condition = expression(rv, in.Condition)
	ct := out.Condition.ExpressionType()
	if ct == nil {
		rv.errorf(in, "condition was not valid")
		return out
	}
	if !equal(ct, semantic.BoolType) {
		rv.errorf(in, "if condition must be boolean (got %s)", typename(ct))
	}
	out.True = block(rv, in.True, out)
	if in.False != nil {
		out.False = block(rv, in.False, out)
	}
	rv.mappings.Add(in, out)
	return out
}

func switch_(rv *resolver, in *ast.Switch) *semantic.Switch {
	out := &semantic.Switch{AST: in}
	out.Value = expression(rv, in.Value)
	vt := out.Value.ExpressionType()
	for _, c := range in.Cases {
		out.Cases = append(out.Cases, case_(rv, c, vt))
	}
	if in.Default != nil {
		out.Default = block(rv, in.Default.Block, out)
	}
	rv.mappings.Add(in, out)
	return out
}

// case_ translates Case in to a switch Case.
// vt is the resolved type of the switch value being compared against, and can
// be used to infer the case condition type.
func case_(rv *resolver, in *ast.Case, vt semantic.Type) *semantic.Case {
	out := &semantic.Case{AST: in}
	rv.with(vt, func() {
		for _, cond := range in.Conditions {
			exp := expression(rv, cond)
			out.Conditions = append(out.Conditions, exp)
			ct := exp.ExpressionType()
			if !comparable(vt, ct) {
				rv.errorf(cond, "switch value %s is not comparable with case condition %s", typename(vt), typename(ct))
			}
		}
	})
	out.Annotations = annotations(rv, in.Annotations)
	out.Block = block(rv, in.Block, out)
	rv.mappings.Add(in, out)
	return out
}

func iteration(rv *resolver, in *ast.Iteration) semantic.Statement {
	v := &semantic.Local{Named: semantic.Named(in.Variable.Value)}
	rv.mappings.Add(in.Variable, v)
	iterable := expression(rv, in.Iterable)
	b, ok := iterable.(*semantic.BinaryOp)
	if !ok {
		rv.errorf(in, "iterable can only be range operator, got %T", b)
		return semantic.Invalid{}
	} else if b.Operator != ast.OpRange {
		rv.errorf(in, "iterable can only be range operator, got %s\n", b.Operator)
	}
	rv.mappings.Remove(b) // The binary op is no longer referenced directly.
	out := &semantic.Iteration{AST: in, Iterator: v, From: b.LHS, To: b.RHS}
	v.Type = iterable.ExpressionType()
	rv.with(semantic.VoidType, func() {
		rv.addNamed(v)
		out.Block = block(rv, in.Block, out)
	})
	rv.mappings.Add(in, out)
	return out
}

func mapIteration(rv *resolver, in *ast.MapIteration) *semantic.MapIteration {
	i := &semantic.Local{Named: semantic.Named(in.IndexVariable.Value)}
	k := &semantic.Local{Named: semantic.Named(in.KeyVariable.Value)}
	v := &semantic.Local{Named: semantic.Named(in.ValueVariable.Value)}
	rv.mappings.Add(in.IndexVariable, i)
	rv.mappings.Add(in.KeyVariable, k)
	rv.mappings.Add(in.ValueVariable, v)
	out := &semantic.MapIteration{AST: in, IndexIterator: i, KeyIterator: k, ValueIterator: v}
	out.Map = expression(rv, in.Map)
	if m, ok := out.Map.ExpressionType().(*semantic.Map); ok {
		i.Type = semantic.Int32Type
		k.Type = m.KeyType
		v.Type = m.ValueType
	} else {
		rv.errorf(in, "key value iteration can only be done on a map, got %T", out.Map.ExpressionType())
		i.Type = semantic.InvalidType
		k.Type = semantic.InvalidType
		v.Type = semantic.InvalidType
	}
	rv.with(semantic.VoidType, func() {
		rv.addNamed(i)
		rv.addNamed(k)
		rv.addNamed(v)
		out.Block = block(rv, in.Block, out)
	})
	rv.mappings.Add(in, out)
	return out
}

func return_(rv *resolver, in *ast.Return, f *semantic.Function) *semantic.Return {
	out := &semantic.Return{AST: in}
	out.Function = f
	rv.with(f.Return.Type, func() {
		out.Value = expression(rv, in.Value)
	})
	inferUnknown(rv, f.Return, out.Value)
	rt := out.Value.ExpressionType()
	if !assignable(f.Return.Type, rt) {
		rv.errorf(in, "cannot assign %s to %s", typename(rt), typename(f.Return.Type))
	}
	rv.mappings.Add(in, out)
	return out
}
