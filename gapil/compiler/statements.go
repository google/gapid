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

package compiler

import (
	"fmt"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

const (
	retError = "error"
	retValue = "value"
)

func (c *compiler) returnType(f *semantic.Function) codegen.Type {
	fields := []codegen.Field{{Name: retError, Type: c.ty.Uint32}}
	if f.Return.Type != semantic.VoidType {
		fields = append(fields, codegen.Field{
			Name: retValue,
			Type: c.targetType(f.Return.Type),
		})
	}
	return c.ty.Struct(f.Name()+"_result", fields...)
}

func (c *compiler) command(f *semantic.Function) {
	if _, ok := c.functions[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)
	resTy := c.returnType(f)
	fields := make([]codegen.Field, len(f.FullParameters))
	for i, p := range f.FullParameters {
		fields[i] = codegen.Field{Name: p.Name(), Type: c.targetType(p.Type)}
	}
	paramTy := c.ty.Pointer(c.ty.Struct(f.Name()+"_params", fields...))
	out := c.module.Function(resTy, f.Name(), c.ty.ctxPtr, paramTy)
	err(out.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		params := jb.Parameter(1).SetName("params")
		for _, p := range f.FullParameters {
			v := params.Index(0, p.Name()).Load()
			v.SetName(p.Name())
			s.parameters[p] = v
		}

		c.applyReads(s)

		c.block(s, f.Block)
	}))
	c.functions[f] = out
	c.setCurrentFunction(old)
}

func (c *compiler) subroutine(f *semantic.Function) {
	if _, ok := c.functions[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)
	resTy := c.returnType(f)

	params := f.CallParameters()
	paramTys := make([]codegen.Type, len(params)+1)
	paramTys[0] = c.ty.ctxPtr
	for i, p := range params {
		paramTys[i+1] = c.targetType(p.Type)
	}
	out := c.module.Function(resTy, f.Name(), paramTys...)
	err(out.Build(func(jb *codegen.Builder) {
		s := c.scope(jb)
		for i, p := range params {
			s.parameters[p] = jb.Parameter(i + 1).SetName(p.Name())
		}
		c.block(s, f.Block)
	}))
	c.functions[f] = out
	c.setCurrentFunction(old)
}

func (c *compiler) block(s *scope, n *semantic.Block) {
	for _, st := range n.Statements {
		if !c.statement(s, st) {
			break
		}
	}
}

func (c *compiler) statement(s *scope, n semantic.Node) bool {
	old := c.setCurrentStatement(s, n)
	switch n := n.(type) {
	case *semantic.Abort:
		c.abort(s, n)
		return false
	case *semantic.ArrayAssign:
		c.arrayAssign(s, n)
	case *semantic.Assign:
		c.assign(s, n)
	case *semantic.Branch:
		c.branch(s, n)
	case *semantic.Call:
		c.call(s, n)
	case *semantic.Copy:
		c.copy(s, n)
	case *semantic.DeclareLocal:
		c.declareLocal(s, n)
	case *semantic.Fence:
		c.fence(s, n)
	case *semantic.Iteration:
		c.iteration(s, n)
	case *semantic.MapAssign:
		c.mapAssign(s, n)
	case *semantic.MapIteration:
		c.mapIteration(s, n)
	case *semantic.MapRemove:
		c.mapRemove(s, n)
	case *semantic.Read:
		c.read(s, n)
	case *semantic.Return:
		c.return_(s, n)
		return false
	case *semantic.SliceAssign:
		c.sliceAssign(s, n)
	case *semantic.Switch:
		c.switch_(s, n)
	case *semantic.Write:
		c.write(s, n)
	default:
		panic(fmt.Errorf("Unexpected semantic type %T", n))
	}
	c.setCurrentStatement(s, old)
	return true
}

func (c *compiler) abort(s *scope, n *semantic.Abort) {
	retTy := c.returnType(c.currentFunc)
	s.Return(s.Zero(retTy).Insert(retError, s.Scalar(ErrAborted)))
}

func (c *compiler) applyReads(s *scope) {
	s.Call(c.callbacks.applyReads, s.ctx)
}

func (c *compiler) applyWrites(s *scope) {
	s.Call(c.callbacks.applyWrites, s.ctx)
}

func (c *compiler) arrayAssign(s *scope, n *semantic.ArrayAssign) {
	if n.Operator != ast.OpAssign {
		fail("Unsupported ArrayAssign operator '%s'", n.Operator)
	}
	val := c.expression(s, n.Value)
	c.assignTo(s, n.Operator, n.To, val)
}

func (c *compiler) assign(s *scope, n *semantic.Assign) {
	c.assignTo(s, n.Operator, n.LHS, c.expression(s, n.RHS))
}

func (c *compiler) assignTo(s *scope, op string, target semantic.Expression, val *codegen.Value) {
	if _, isIgnore := target.(*semantic.Ignore); isIgnore {
		return
	}

	dst := c.expressionAddr(s, target)
	switch op {
	case ast.OpAssign:
		if ty := target.ExpressionType(); c.isRefCounted(ty) {
			c.reference(s, val, ty)
			c.release(s, dst.Load(), ty)
		}
		dst.Store(val)
	case ast.OpAssignPlus:
		// TODO: Ref counting for strings?!
		dst.Store(c.doBinaryOp(s, ast.OpPlus, dst.Load(), val))
	case ast.OpAssignMinus:
		dst.Store(c.doBinaryOp(s, ast.OpMinus, dst.Load(), val))
	default:
		fail("Unsupported composite assignment operator '%s'", op)
	}
}

func (c *compiler) expressionAddr(s *scope, target semantic.Expression) *codegen.Value {
	path := []codegen.ValueIndexOrName{}
	revPath := func() {
		for i, c, m := 0, len(path), len(path)/2; i < m; i++ {
			j := c - i - 1
			path[i], path[j] = path[j], path[i]
		}
	}
	for {
		switch n := target.(type) {
		case *semantic.Global:
			path = append(path, n.Name(), 0)
			revPath()
			return s.globals.Index(path...)
		case *semantic.Local:
			path = append(path, 0)
			revPath()
			return s.locals[n].Index(path...)
		case *semantic.Member:
			path = append(path, n.Field.Name())
			target = n.Object
			if semantic.IsReference(target.ExpressionType()) {
				path = append(path, refValue, 0)
				revPath()
				return c.expression(s, target).Index(path...)
			}
		case *semantic.ArrayIndex:
			path = append(path, c.expression(s, n.Index))
			target = n.Array
		case *semantic.MapIndex:
			path = append(path, 0)
			revPath()
			m := c.expression(s, n.Map)
			k := c.expression(s, n.Index)
			v := s.Call(c.ty.maps[n.Type].Index, s.ctx, m, k, s.Scalar(false)).SetName("map_get")
			return v.Index(path...)
		case *semantic.Unknown:
			target = n.Inferred
		case *semantic.Ignore:
			return nil
		default:
			path = append(path, 0)
			revPath()
			return s.LocalInit("tmp", c.expression(s, n)).Index(path...)
		}
	}
}

func (c *compiler) branch(s *scope, n *semantic.Branch) {
	cond := c.expression(s, n.Condition)
	onTrue := func() { c.block(s, n.True) }
	onFalse := func() { c.block(s, n.False) }
	if n.False == nil {
		onFalse = nil
	}
	s.IfElse(cond, onTrue, onFalse)
}

func (c *compiler) copy(s *scope, n *semantic.Copy) {
	src := c.expression(s, n.Src)
	dst := c.expression(s, n.Dst)
	c.doCopy(s, dst, src, n.Src.ExpressionType().(*semantic.Slice).To)
}

func (c *compiler) doCopy(s *scope, dst, src *codegen.Value, elTy semantic.Type) {
	s.Call(c.callbacks.copySlice, s.ctx, s.LocalInit("dstPtr", dst), s.LocalInit("srcPtr", src))
}

func (c *compiler) declareLocal(s *scope, n *semantic.DeclareLocal) {
	var def *codegen.Value
	if n.Local.Value != nil {
		def = c.expression(s, n.Local.Value)
	} else {
		def = c.initialValue(s, n.Local.Type)
	}
	c.reference(s, def, n.Local.Type)
	local := s.LocalInit(n.Local.Name(), def)
	s.locals[n.Local] = local
	defer func() { c.release(s, local.Load(), n.Local.Type) }()
}

func (c *compiler) fence(s *scope, n *semantic.Fence) {
	c.applyWrites(s)
	if n.Statement != nil {
		c.statement(s, n.Statement)
	}
}

func (c *compiler) iteration(s *scope, n *semantic.Iteration) {
	from := c.expression(s, n.From)
	to := c.expression(s, n.To)
	one := s.One(from.Type())
	it := s.LocalInit(n.Iterator.Name(), from)
	s.While(func() *codegen.Value {
		return s.NotEqual(it.Load(), to)
	}, func() {
		s.enter(func(s *scope) {
			s.locals[n.Iterator] = it
			c.block(s, n.Block)
			it.Store(s.Add(it.Load(), one))
		})
	})
}

func (c *compiler) mapAssign(s *scope, n *semantic.MapAssign) {
	ty := n.To.Type
	m := c.expression(s, n.To.Map)
	k := c.expression(s, n.To.Index)
	v := c.expression(s, n.Value)
	s.Call(c.ty.maps[ty].Index, s.ctx, m, k, s.Scalar(true)).Store(v)
}

func (c *compiler) mapIteration(s *scope, n *semantic.MapIteration) {
	mapPtr := c.expression(s, n.Map)
	count := mapPtr.Index(0, mapCount).Load()
	elPtr := mapPtr.Index(0, mapElements).Load()
	iTy := c.targetType(n.IndexIterator.Type)
	i := s.Local("i", iTy)
	s.ForN(count.Cast(iTy), func(it *codegen.Value) *codegen.Value {
		i.Store(it)
		k := elPtr.Index(it, "k")
		v := elPtr.Index(it, "v")
		s.enter(func(s *scope) {
			s.locals[n.IndexIterator] = i
			s.locals[n.KeyIterator] = k
			s.locals[n.ValueIterator] = v
			c.block(s, n.Block)
		})
		return nil
	})
}

func (c *compiler) mapRemove(s *scope, n *semantic.MapRemove) {
	ty := n.Type
	m := c.expression(s, n.Map)
	k := c.expression(s, n.Key)
	s.Call(c.ty.maps[ty].Remove, s.ctx, m, k)
}

func (c *compiler) read(s *scope, n *semantic.Read) {
	// TODO
}

func (c *compiler) return_(s *scope, n *semantic.Return) {
	retTy := c.returnType(c.currentFunc)
	retVal := s.Zero(retTy).Insert(retValue, c.expression(s, n.Value))
	s.Return(retVal)
}

func (c *compiler) sliceAssign(s *scope, n *semantic.SliceAssign) {
	if n.Operator != ast.OpAssign {
		fail("Unsupported slice composite assignment operator '%s'", n.Operator)
	}

	index := c.expression(s, n.To.Index).Cast(c.ty.Uint64).SetName("index")
	slice := c.expression(s, n.To.Slice)

	elTy := n.To.Type.To
	targetTy := c.targetType(elTy)
	storageTy := c.storageType(elTy)

	write := func(el *codegen.Value) {
		base := slice.Extract(sliceBase).Cast(c.ty.Pointer(el.Type()))
		base.Index(index).Store(el)
	}

	if !c.settings.WriteToApplicationPool {
		// Writes to the application pool are disabled by default.
		// This can be overridden with the WriteToApplicationPool setting.
		actuallyWrite := write
		write = func(el *codegen.Value) {
			pool := slice.Extract(slicePool)
			s.If(s.NotEqual(pool, s.appPool), func() {
				actuallyWrite(el)
			})
		}
	}

	el := c.expression(s, n.Value)
	if targetTy == storageTy {
		write(el)
	} else {
		write(c.castTargetToStorage(s, elTy, el))
	}
}

func (c *compiler) switch_(s *scope, n *semantic.Switch) {
	cases := make([]codegen.SwitchCase, len(n.Cases))
	for i, kase := range n.Cases {
		i, kase := i, kase
		cases[i] = codegen.SwitchCase{
			Conditions: func() []*codegen.Value {
				conds := make([]*codegen.Value, len(kase.Conditions))
				for i, cond := range kase.Conditions {
					conds[i] = c.expression(s, cond)
				}
				return conds
			},
			Block: func() { c.block(s, kase.Block) },
		}
	}
	if n.Default != nil {
		s.Switch(c.expression(s, n.Value), cases, func() { c.block(s, n.Default) })
	} else {
		s.Switch(c.expression(s, n.Value), cases, nil)
	}
}

func (c *compiler) write(s *scope, n *semantic.Write) {
	// TODO
}
