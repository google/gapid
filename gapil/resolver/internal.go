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

var (
	internals = map[string]func(*resolver, *ast.Call, *ast.Generic) semantic.Node{}
)

func init() {
	internals["assert"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return assert(rv, in, g) }
	internals["as"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return cast(rv, in, g) }
	internals["new"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return new_(rv, in, g) }
	internals["make"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return make_(rv, in, g) }
	internals["clone"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return clone(rv, in, g) }
	internals["read"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return read(rv, in, g) }
	internals["write"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return write(rv, in, g) }
	internals["copy"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return copy_(rv, in, g) }
	internals["len"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return length(rv, in, g) }
	internals["print"] = func(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Node { return print(rv, in, g) }
}

func internalCall(rv *resolver, in *ast.Call) semantic.Node {
	g, ok := in.Target.(*ast.Generic)
	if !ok {
		return nil
	}
	p, ok := internals[g.Name.Value]
	if !ok {
		return nil
	}
	return p(rv, in, g)
}

func checkInternalFunc(rv *resolver, in *ast.Call, g *ast.Generic, typeCount, paramCount int) bool {
	if typeCount >= 0 && len(g.Arguments) != typeCount {
		rv.errorf(in, "wrong number of types to %s, expected %d got %v", g.Name.Value, typeCount, len(g.Arguments))
		return false
	}
	if paramCount >= 0 && len(in.Arguments) != paramCount {
		rv.errorf(in, "wrong number of arguments to %s, expected %d got %v", g.Name.Value, paramCount, len(in.Arguments))
		return false
	}
	return true
}

func assert(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Statement {
	if !checkInternalFunc(rv, in, g, 0, 1) {
		return semantic.Invalid{}
	}
	condition := expression(rv, in.Arguments[0])
	t := condition.ExpressionType()
	if !equal(t, semantic.BoolType) {
		rv.errorf(in, "assert expression must be a bool, got %s", typename(t))
	}
	msg := rv.mappings.AST.CST(in).Tok().String()
	out := &semantic.Assert{AST: in, Condition: condition, Message: msg}
	rv.mappings.Add(in, out)
	return out
}

func cast(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Expression {
	if !checkInternalFunc(rv, in, g, 1, 1) {
		return semantic.Invalid{}
	}
	t := type_(rv, g.Arguments[0])
	var obj semantic.Expression
	rv.with(t, func() {
		obj = expression(rv, in.Arguments[0])
	})
	if equal(t, obj.ExpressionType()) {
		rv.mappings.Add(in, obj)
		return obj
	}
	if !castable(obj.ExpressionType(), t) {
		rv.errorf(in, "cannot cast from %s to %s", typename(obj.ExpressionType()), typename(t))
	}
	out := &semantic.Cast{AST: in, Object: obj, Type: t}
	rv.mappings.Add(in, out)
	return out
}

func new_(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Expression {
	if !checkInternalFunc(rv, in, g, 1, -1) {
		return semantic.Invalid{}
	}
	t := type_(rv, g.Arguments[0])
	rt := getRefType(rv, in, t)
	if c, isclass := t.(*semantic.Class); isclass {
		i := classInitializer(rv, c, in)
		out := &semantic.Create{AST: in, Type: rt, Initializer: i}
		rv.mappings.Add(in, out)
		return out
	}
	out := &semantic.New{AST: in, Type: rt}
	rv.mappings.Add(in, out)
	return out
}

func make_(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Expression {
	if !checkInternalFunc(rv, in, g, 1, 1) {
		return semantic.Invalid{}
	}
	t := type_(rv, g.Arguments[0])

	var size semantic.Expression
	rv.with(semantic.Uint64Type, func() {
		size = expression(rv, in.Arguments[0])
	})
	size = castTo(rv, in.Arguments[0], size, semantic.Uint64Type)
	out := &semantic.Make{AST: in, Type: getSliceType(rv, in, t), Size: size}
	rv.mappings.Add(in, out)
	return out
}

func clone(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Expression {
	if !checkInternalFunc(rv, in, g, 0, 1) {
		return semantic.Invalid{}
	}
	slice := expression(rv, in.Arguments[0])
	st, ok := slice.ExpressionType().(*semantic.Slice)
	if !ok {
		rv.errorf(in, "%s only works on slice types, got type %v", g.Name.Value, typename(slice.ExpressionType()))
		return semantic.Invalid{}
	}
	out := &semantic.Clone{AST: in, Slice: slice, Type: st}
	rv.mappings.Add(in, out)
	return out
}

func read(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Statement {
	if !checkInternalFunc(rv, in, g, 0, 1) {
		return semantic.Invalid{}
	}
	slice := expression(rv, in.Arguments[0])
	if _, ok := slice.ExpressionType().(*semantic.Slice); !ok {
		rv.errorf(in, "%s only works on slice types, got type %v", g.Name.Value, typename(slice.ExpressionType()))
		return semantic.Invalid{}
	}
	out := &semantic.Read{AST: in, Slice: slice}
	rv.mappings.Add(in, out)
	return out
}

func write(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Statement {
	if !checkInternalFunc(rv, in, g, 0, 1) {
		return semantic.Invalid{}
	}
	slice := expression(rv, in.Arguments[0])
	if _, ok := slice.ExpressionType().(*semantic.Slice); !ok {
		rv.errorf(in, "%s only works on slice types, got type %v", g.Name.Value, typename(slice.ExpressionType()))
		return semantic.Invalid{}
	}
	out := &semantic.Write{AST: in, Slice: slice}
	rv.mappings.Add(in, out)
	return out
}

func copy_(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Statement {
	if !checkInternalFunc(rv, in, g, 0, 2) {
		return semantic.Invalid{}
	}
	src := expression(rv, in.Arguments[1])
	srct, ok := src.ExpressionType().(*semantic.Slice)
	if !ok {
		rv.errorf(in, "%s only works on slice types, got type %v", g.Name.Value, typename(src.ExpressionType()))
		return semantic.Invalid{}
	}
	dst := expression(rv, in.Arguments[0])
	if !equal(srct, dst.ExpressionType()) {
		rv.errorf(in, "%s slice types do't match, got %v and %v", g.Name.Value, typename(srct), typename(dst.ExpressionType()))
		return semantic.Invalid{}
	}
	out := &semantic.Copy{AST: in, Src: src, Dst: dst}
	rv.mappings.Add(in, out)
	return out
}

func length(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Expression {
	if !checkInternalFunc(rv, in, g, 0, 1) {
		return semantic.Invalid{}
	}
	obj := expression(rv, in.Arguments[0])
	at := obj.ExpressionType()
	if at == nil {
		rv.errorf(in, "object was not valid")
		return semantic.Invalid{}
	}
	ok := false
	ty := semantic.Underlying(at)
	switch ty := ty.(type) {
	case *semantic.Slice:
		ok = true
	case *semantic.Map:
		ok = true
	case *semantic.Builtin:
		if ty == semantic.StringType {
			ok = true
		}
	}
	if !ok {
		rv.errorf(in, "len cannot work out length of type %s", typename(at))
		return semantic.Invalid{}
	}
	infer := semantic.Underlying(rv.scope.inferType)
	var t semantic.Type = semantic.Int32Type
	switch infer {
	case semantic.Int32Type, semantic.Uint32Type, semantic.Int64Type, semantic.Uint64Type:
		t = infer
	default:
		t = semantic.Int32Type
	}
	out := &semantic.Length{AST: in, Object: obj, Type: t}
	rv.mappings.Add(in, out)
	return out
}

func print(rv *resolver, in *ast.Call, g *ast.Generic) semantic.Statement {
	out := &semantic.Print{AST: in}

	for i, a := range in.Arguments {
		arg := expression(rv, a)
		if isVoid(arg.ExpressionType()) {
			rv.errorf(a, "argument %d to print is void", i)
			return semantic.Invalid{}
		}
		out.Arguments = append(out.Arguments, arg)
	}
	return out
}
