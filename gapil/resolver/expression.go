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
	"strconv"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// entity translates the ast expression to a semantic expression,
// genericSubroutine or imported API.
func entity(rv *resolver, in ast.Node) interface{} {
	var out interface{}
	switch in := in.(type) {
	case *ast.UnaryOp:
		out = unaryOp(rv, in)
	case *ast.BinaryOp:
		out = binaryOp(rv, in)
	case *ast.Call:
		out = call(rv, in)
	case *ast.Definition:
		out = expression(rv, in.Expression)
	case *ast.Switch:
		out = select_(rv, in)
	case *ast.Member:
		out = member(rv, in)
	case *ast.Index:
		out = index(rv, in)
	case *ast.Identifier:
		out = identifier(rv, in)
	case *ast.Generic:
		out = generic(rv, in)
	case *ast.Group:
		out = expression(rv, in.Expression)
	case *ast.Unknown:
		out = &semantic.Unknown{AST: in}
	case *ast.Number:
		out = number(rv, in)
	case *ast.Bool:
		out = semantic.BoolValue(in.Value)
	case *ast.String:
		out = semantic.StringValue(in.Value)
	case *ast.Null:
		out = semantic.Null{AST: in, Type: rv.scope.inferType}
	case *ast.Invalid:
		out = semantic.Invalid{}
	default:
		rv.icef(in, "Unhandled expression type %T found", in)
		out = semantic.Invalid{}
	}

	switch out := out.(type) {
	case *semantic.API:
		rv.mappings.Add(in, out)
		return out
	case semantic.Expression:
		rv.mappings.Add(in, out)
		return out
	}

	rv.errorf(in, "Expected entity, got %T", out)
	node, _ := out.(semantic.Node)
	return semantic.Invalid{Partial: node}
}

// expression translates the ast expression to a semantic expression.
func expression(rv *resolver, in ast.Node) semantic.Expression {
	out := entity(rv, in)

	if out, ok := out.(semantic.Expression); ok {
		rv.mappings.Add(in, out)
		return out
	}

	rv.errorf(in, "Expected expression, got %T", out)
	node, _ := out.(semantic.Node)
	return semantic.Invalid{Partial: node}
}

func call(rv *resolver, in *ast.Call) semantic.Node {
	if b := internalCall(rv, in); b != nil {
		return b
	}
	if a := arrayCall(rv, in); a != nil {
		return a
	}
	if c := classCall(rv, in); c != nil {
		return c
	}
	target := expression(rv, in.Target)
	switch target := target.(type) {
	case *semantic.Callable:
		return functionCall(rv, in, target)
	default:
		rv.errorf(in, "Invalid method call target %T found", target)
		return semantic.Invalid{}
	}
}

func callArguments(rv *resolver, at ast.Node, in []ast.Node, params []*semantic.Parameter, name string) []semantic.Expression {
	var out []semantic.Expression
	if len(params) != len(in) {
		rv.errorf(at, "wrong number of arguments to %s, expected %v got %v", name, len(params), len(in))
		return out
	}
	for i, a := range in {
		p := params[i]
		rv.with(p.Type, func() {
			arg := expression(rv, a)
			at := arg.ExpressionType()
			if paramIsMessage := p.Type == semantic.MessageType; paramIsMessage {
				switch arg := arg.(type) {
				case *semantic.Parameter:
					if arg.Type == semantic.MessageType {
						out = append(out, arg)
						return
					}
				case *semantic.Create:
					// new!Class{foo: bar} -> message
					if argClass, ok := arg.Type.To.(*semantic.Class); ok {
						out = append(out, &semantic.MessageValue{AST: argClass.AST,
							Arguments: arg.Initializer.Fields})
					} else {
						rv.errorf(a, "Message parameters have to reference a class, got: %T -> %T", arg, arg.Type.To)
					}

					// The Create node has been substituted with a MessageValue
					// and is no longer referenced directly.
					rv.mappings.Remove(arg)
					rv.mappings.Remove(arg.Initializer)
					return
				}

				rv.errorf(a, "Message arguments require a new class instance or forwarded message parameter, got: %T", arg)
				return
			}

			if assignable(p.Type, at) {
				out = append(out, arg)
				return
			}

			rv.errorf(a, "argument %d to %s is wrong type, expected %s got %s", i, name, typename(p.Type), typename(at))
		})
	}
	return out
}

func functionCall(rv *resolver, in *ast.Call, target *semantic.Callable) *semantic.Call {
	out := &semantic.Call{AST: in, Target: target, Type: semantic.VoidType}
	params := target.Function.FullParameters
	if target.Object != nil {
		if target.Function.This == nil {
			rv.errorf(in, "method call on non method %s of %T", target.Function.Name(), target.Object)
			return out
		}
		params = params[1:]
	}
	if f := target.Function; !f.Extern && !f.Subroutine {
		rv.errorf(in, "Commands cannot call other commands. Did you mean to make '%s' a subroutine?", f.Name())
		return out
	}
	if target.Function.Return != nil && !isVoid(target.Function.Return.Type) {
		params = params[0 : len(params)-1]
	}
	out.Arguments = callArguments(rv, in, in.Arguments, params, target.Function.Name())
	if ret := target.Function.Return; ret != nil {
		out.Type = ret.Type
	} else {
		out.Type = semantic.VoidType
	}
	rv.mappings.Add(in, out)
	return out
}

func select_(rv *resolver, in *ast.Switch) *semantic.Select {
	out := &semantic.Select{AST: in}
	out.Type = nil
	out.Value = expression(rv, in.Value)
	vt := out.Value.ExpressionType()
	for _, c := range in.Cases {
		e := choice(rv, c, vt)
		out.Choices = append(out.Choices, e)
		if out.Type == nil {
			out.Type = e.Expression.ExpressionType()
		} else if !equal(out.Type, e.Expression.ExpressionType()) {
			// TODO: This could be a common ancestor type instead?
			out.Type = semantic.VoidType
		}
	}
	if d := in.Default; d != nil {
		if len(in.Default.Block.Statements) != 1 {
			rv.errorf(in, "switch default is not a single expression")
		}
		e := expression(rv, in.Default.Block.Statements[0])
		out.Default = e
		if out.Type == nil {
			out.Type = e.ExpressionType()
		} else if !equal(out.Type, e.ExpressionType()) {
			// TODO: This could be a common ancestor type instead?
			out.Type = semantic.VoidType
		}
	}
	if out.Type == nil {
		rv.errorf(in, "could not determine type of switch")
		out.Type = semantic.VoidType
	}
	rv.mappings.Add(in, out)
	return out
}

// choice translates Case in to a select Choice.
// vt is the resolved type of the select value being compared against, and can
// be used to infer the choice condition type.
func choice(rv *resolver, in *ast.Case, vt semantic.Type) *semantic.Choice {
	out := &semantic.Choice{AST: in}
	rv.with(vt, func() {
		for _, cond := range in.Conditions {
			exp := expression(rv, cond)
			out.Conditions = append(out.Conditions, exp)
			ct := exp.ExpressionType()
			if !comparable(vt, ct) {
				rv.errorf(cond, "select value %s is not comparable with choice condition %s", typename(vt), typename(ct))
			}
		}
	})
	if len(in.Block.Statements) != 1 {
		rv.errorf(in, "switch case is not a single expression")
		out.Expression = semantic.Invalid{}
		return out
	}
	out.Annotations = annotations(rv, in.Annotations)
	out.Expression = expression(rv, in.Block.Statements[0])
	rv.mappings.Add(in, out)
	return out
}

func member(rv *resolver, in *ast.Member) semantic.Expression {
	obj := entity(rv, in.Object)
	switch obj := obj.(type) {
	case semantic.Expression:
		ot := obj.ExpressionType()
		owned := ot.Member(in.Name.Value)
		var out semantic.Expression
		switch owned := owned.(type) {
		case *semantic.Field:
			out = &semantic.Member{AST: in, Object: obj, Field: owned}
		case *semantic.Function:
			out = &semantic.Callable{Object: obj, Function: owned}
		case nil:
			if in.Name != ast.InvalidIdentifier {
				rv.errorf(in, "%s is not a member of %s", in.Name.Value, typename(ot))
			}
			out = &semantic.Field{Type: semantic.InvalidType, Named: semantic.Named(in.Name.Value)}
		}
		if out != nil {
			rv.mappings.Add(in.Name, owned)
			rv.mappings.Add(in, out)
			return out
		}
		rv.errorf(in, "Invalid member lookup type %T found", owned)
		return semantic.Invalid{Partial: out}
	case *semantic.API:
		owned := obj.Member(in.Name.Value)
		if expr, ok := owned.(semantic.Expression); ok {
			rv.mappings.Add(in.Name, expr)
			rv.mappings.Add(in.Object, owned)
			return expr
		}
		rv.errorf(in, "Expected expression, got: %T", owned)
		return semantic.Invalid{Partial: obj}
	}
	rv.errorf(in, "%T does not have members", obj)
	node, _ := obj.(semantic.Node)
	return semantic.Invalid{Partial: node}
}

func castTo(rv *resolver, in ast.Node, expr semantic.Expression, to semantic.Type) semantic.Expression {
	ty := expr.ExpressionType()
	if equal(ty, to) {
		return expr
	}
	if !castable(ty, to) {
		rv.errorf(in, "cannot cast %s to %s", typename(ty), typename(to))
	}
	return &semantic.Cast{Object: expr, Type: to}
}

func index(rv *resolver, in *ast.Index) semantic.Expression {
	object := expression(rv, in.Object)
	at := semantic.Underlying(object.ExpressionType())
	var index semantic.Expression

	switch at := at.(type) {
	case *semantic.StaticArray:
		rv.with(semantic.Uint64Type, func() {
			index = expression(rv, in.Index)
		})
		if bop, ok := index.(*semantic.BinaryOp); ok && bop.Operator == ast.OpSlice {
			rv.errorf(in, "cannot slice static arrays")
			return semantic.Invalid{}
		}

		it := index.ExpressionType()
		if isNumber(semantic.Underlying(it)) {
			if v, ok := index.(semantic.Uint64Value); ok && uint32(v) >= at.Size {
				rv.errorf(in, "array index %d is out of bounds for %s", index, typename(at))
			}
		} else {
			rv.errorf(in, "array index must be a number, got %s", typename(it))
		}
		out := &semantic.ArrayIndex{AST: in, Array: object, Type: at, Index: index}
		rv.mappings.Add(in, out)
		return out
	case *semantic.Pointer:
		rv.with(semantic.Uint64Type, func() {
			index = expression(rv, in.Index)
		})
		if bop, ok := index.(*semantic.BinaryOp); ok && bop.Operator == ast.OpSlice {
			// pointer[a:b]
			if bop.LHS != nil {
				bop.LHS = castTo(rv, bop.AST.LHS, bop.LHS, semantic.Uint64Type)
			}
			if bop.RHS != nil {
				bop.RHS = castTo(rv, bop.AST.RHS, bop.RHS, semantic.Uint64Type)
			}
			out := &semantic.PointerRange{AST: in, Pointer: object, Type: at.Slice, Range: bop}
			rv.mappings.Add(in, out)
			return out
		}
		if n, ok := index.(semantic.Uint64Value); ok {
			// pointer[n]
			r := &semantic.BinaryOp{LHS: n, Operator: ast.OpSlice, RHS: n + 1}
			slice := &semantic.PointerRange{AST: in, Pointer: object, Type: at.Slice, Range: r}
			out := &semantic.SliceIndex{AST: in, Slice: slice, Type: at.Slice, Index: semantic.Uint64Value(0)}
			rv.mappings.Add(in, out)
			return out
		}
		rv.errorf(in, "type %s not valid slicing pointer", typename(index.ExpressionType()))
		return semantic.Invalid{}
	case *semantic.Slice:
		rv.with(semantic.Uint64Type, func() {
			index = expression(rv, in.Index)
		})
		if bop, ok := index.(*semantic.BinaryOp); ok && bop.Operator == ast.OpSlice {
			// slice[a:b]
			if bop.LHS != nil {
				bop.LHS = castTo(rv, bop.AST.LHS, bop.LHS, semantic.Uint64Type)
			}
			if bop.RHS != nil {
				bop.RHS = castTo(rv, bop.AST.RHS, bop.RHS, semantic.Uint64Type)
			}
			out := &semantic.SliceRange{AST: in, Slice: object, Type: at, Range: bop}
			rv.mappings.Add(in, out)
			return out
		}
		// slice[a]
		index = castTo(rv, in, index, semantic.Uint64Type)
		out := &semantic.SliceIndex{AST: in, Slice: object, Type: at, Index: index}
		rv.mappings.Add(in, out)
		return out
	case *semantic.Map:
		// map[k]
		rv.with(at.KeyType, func() {
			index = expression(rv, in.Index)
		})
		it := index.ExpressionType()
		if !comparable(it, at.KeyType) {
			rv.errorf(in, "type %s not valid indexing map", typename(it))
		}
		out := &semantic.MapIndex{AST: in, Map: object, Type: at, Index: index}
		rv.mappings.Add(in, out)
		return out
	}
	rv.errorf(in, "index operation on non indexable type %s", typename(at))
	return semantic.Invalid{}
}

func identifier(rv *resolver, in *ast.Identifier) interface{} {
	if in == ast.InvalidIdentifier {
		out := semantic.Invalid{}
		rv.mappings.Add(in, out)
		return out
	}
	out := rv.get(in, in.Value)
	switch out := out.(type) {
	case *semantic.Definition:
		cyclic := rv.defStack.contains(out)
		rv.defStack.push(out)
		defer rv.defStack.pop()

		if cyclic {
			if !isInvalid(out.Expression) { // Don't repeat errors.
				rv.errorf(in, "cyclic define declaration: %v", rv.defStack)
				out.Expression = semantic.Invalid{}
			}
			return semantic.Invalid{}
		}
		s := &semantic.DefinitionUsage{
			Definition: out,
			Expression: expression(rv, out.AST),
		}
		rv.mappings.Add(in, s)
		return s
	case *semantic.Function:
		rv.mappings.Add(in, out)
		return &semantic.Callable{Function: out}
	case *semantic.API:
		rv.mappings.Add(in, out)
		return out
	case genericSubroutine:
		return out
	case semantic.Expression:
		rv.mappings.Add(in, out)
		return out
	case nil:
		return semantic.Invalid{} // rv.get() already created error.
	default:
		rv.errorf(in, "Symbol %s was non expression %T", in.Value, out)
		return semantic.Invalid{}
	}
}

func generic(rv *resolver, in *ast.Generic) semantic.Node {
	id := identifier(rv, in.Name)
	if gs, ok := id.(genericSubroutine); ok {
		f := gs.resolve(rv, in)
		rv.mappings.Add(in, f)
		return &semantic.Callable{Function: f}
	}
	if len(in.Arguments) > 0 {
		rv.errorf(in, "identifier %s does not support type arguments", in.Name.Value)
	}
	if n, ok := id.(semantic.Node); ok {
		return n
	}
	rv.errorf(in, "unexpected identifier type %T", id)
	n, _ := id.(semantic.Node)
	return semantic.Invalid{Partial: n}
}

func arrayCall(rv *resolver, in *ast.Call) semantic.Expression {
	g, ok := in.Target.(*ast.Generic)
	if !ok {
		return nil
	}
	t := rv.findType(in, g.Name.Value)
	array, ok := semantic.Underlying(t).(*semantic.StaticArray)
	if !ok {
		return nil
	}

	out := &semantic.ArrayInitializer{AST: in, Array: t}
	rv.mappings.Add(in, out)
	for _, a := range in.Arguments {
		rv.with(array.ValueType, func() {
			v := expression(rv, a)
			if vt := v.ExpressionType(); !assignable(array.ValueType, vt) {
				rv.errorf(a, "cannot assign %s to array element type %s", typename(vt), array.ValueType)
			}
			out.Values = append(out.Values, v)
		})
	}
	if len(out.Values) != int(array.Size) {
		rv.errorf(in, "expected %d values, got %d", array.Size, len(out.Values))
	}
	return out
}

func classCall(rv *resolver, in *ast.Call) semantic.Expression {
	g, ok := in.Target.(*ast.Generic)
	if !ok {
		return nil
	}
	t := rv.findType(in, g.Name.Value)
	class, ok := t.(*semantic.Class)
	if !ok {
		return nil
	}
	return classInitializer(rv, class, in)
}

func classInitializer(rv *resolver, class *semantic.Class, in *ast.Call) *semantic.ClassInitializer {
	out := &semantic.ClassInitializer{AST: in, Class: class}
	rv.mappings.Add(in, out)
	if g, ok := in.Target.(*ast.Generic); ok {
		rv.mappings.Add(g.Name, out)
	}
	if len(in.Arguments) == 0 {
		return out
	}
	if _, named := in.Arguments[0].(*ast.NamedArg); named {
		for _, a := range in.Arguments {
			n, ok := a.(*ast.NamedArg)
			if !ok {
				rv.errorf(a, "initializer for class %s uses a mix of named and unnamed arguments", class.Name())
				return out
			}
			m := class.Member(n.Name.Value)
			if m == nil {
				rv.errorf(n.Name, "class %s has no field %s", class.Name(), n.Name.Value)
				return out
			}
			f, ok := m.(*semantic.Field)
			if !ok {
				rv.errorf(n.Name, "member %s of class %s is not a field [%T]", n.Name.Value, class.Name(), m)
				return out
			}
			rv.mappings.Add(n.Name, f)
			out.Fields = append(out.Fields, fieldInitializer(rv, class, f, n.Value))
		}
		return out
	}

	// Check if the copy constructor is being used.
	if len(in.Arguments) == 1 {
		switch in.Arguments[0].(type) {
		case *ast.Null:
			// "class(null)"" is ambiguous, but it does not make sense to deref null,
			// so we force it to mean to initialize the first field with null.
		default:
			field := &semantic.FieldInitializer{AST: in.Arguments[0]}
			rv.with(class, func() {
				field.Value = expression(rv, in.Arguments[0])
			})
			if equal(class, field.Value.ExpressionType()) {
				out.Fields = append(out.Fields, field)
				return out
			}
		}
	}

	if len(in.Arguments) > len(class.Fields) {
		rv.errorf(in, "too many arguments to class %s constructor, expected %d got %d", class.Name, len(class.Fields), len(in.Arguments))
		return out
	}
	for i, a := range in.Arguments {
		out.Fields = append(out.Fields, fieldInitializer(rv, class, class.Fields[i], a))
	}
	return out
}

func fieldInitializer(rv *resolver, class *semantic.Class, field *semantic.Field, in ast.Node) *semantic.FieldInitializer {
	out := &semantic.FieldInitializer{AST: in, Field: field}
	rv.with(field.Type, func() {
		out.Value = expression(rv, in)
	})
	ft := field.Type
	vt := out.Value.ExpressionType()
	if !assignable(ft, vt) {
		rv.errorf(in, "cannot assign %s to field '%s' of type %s", typename(vt), field.Name(), typename(ft))
	}
	rv.mappings.Add(in, out)
	return out
}

func number(rv *resolver, in *ast.Number) semantic.Expression {
	infer := semantic.Underlying(rv.scope.inferType)
	out := inferNumber(rv, in, infer)
	if out != nil {
		if infer == rv.scope.inferType {
			rv.mappings.Add(in, out)
			return out
		}
		return &semantic.Cast{Type: rv.scope.inferType, Object: out}
	}
	if v, err := strconv.ParseInt(in.Value, 0, 32); err == nil {
		s := semantic.Int32Value(v)
		rv.mappings.Add(in, s)
		return s
	}
	if v, err := strconv.ParseFloat(in.Value, 64); err == nil {
		s := semantic.Float64Value(v)
		rv.mappings.Add(in, s)
		return s
	}
	rv.errorf(in, "could not parse %s as a number (%s)", in.Value, typename(infer))
	return semantic.Invalid{}
}
