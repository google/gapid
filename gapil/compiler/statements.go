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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

const (
	retError = "error"
	retValue = "value"
)

// LoadParameters loads the command's parameters from the context's arguments
// field and stores them into s.Parameters.
func (c *C) LoadParameters(s *S, f *semantic.Function) {
	params := s.Ctx.
		Index(0, ContextArguments).
		Load().
		Cast(c.T.Pointer(c.T.CmdParams[f])).
		SetName("params")

	for _, p := range f.FullParameters {
		v := params.Index(0, p.Name()).Load()
		v.SetName(p.Name())
		s.Parameters[p] = v
	}
}

func (c *C) returnType(f *semantic.Function) codegen.Type {
	fields := []codegen.Field{{Name: retError, Type: c.T.Uint32}}
	if f.Return.Type != semantic.VoidType {
		fields = append(fields, codegen.Field{
			Name: retValue,
			Type: c.T.Target(f.Return.Type),
		})
	}
	return c.T.Struct(f.Name()+"_result", fields...)
}

func (c *C) command(f *semantic.Function) {
	if _, ok := c.functions[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)
	out := c.M.Function(c.returnType(f), f.Name(), c.T.CtxPtr)
	c.Build(out, func(s *S) {
		c.LoadParameters(s, f)

		c.plugins.foreach(func(p OnBeginCommandListener) { p.OnBeginCommand(s, f) })

		c.applyReads(s)

		c.block(s, f.Block)

		c.plugins.foreach(func(p OnEndCommandListener) { p.OnEndCommand(s, f) })
	})
	c.functions[f] = out
	c.setCurrentFunction(old)
}

func (c *C) subroutine(f *semantic.Function) {
	if _, ok := c.functions[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)
	resTy := c.returnType(f)

	params := f.CallParameters()
	paramTys := make([]codegen.Type, len(params)+1)
	paramTys[0] = c.T.CtxPtr
	for i, p := range params {
		paramTys[i+1] = c.T.Target(p.Type)
	}
	out := c.M.Function(resTy, f.Name(), paramTys...)
	c.functions[f] = out
	c.Build(out, func(s *S) {
		for i, p := range params {
			s.Parameters[p] = s.Parameter(i + 1).SetName(p.Name())
		}
		c.block(s, f.Block)
	})
	c.setCurrentFunction(old)
}

func (c *C) block(s *S, n *semantic.Block) {
	s.enter(func(s *S) {
		for _, st := range n.Statements {
			if !c.statement(s, st) {
				break
			}
		}
	})
}

func (c *C) statement(s *S, n semantic.Statement) bool {
	c.pushStatement(s, n)
	defer c.popStatement(s)

	switch n := n.(type) {
	case *semantic.Assert:
		c.assert(s, n)
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
	return true
}

func (c *C) abort(s *S, n *semantic.Abort) {
	retTy := c.returnType(c.currentFunc)
	s.Return(s.Zero(retTy).Insert(retError, s.Scalar(ErrAborted)))
}

func (c *C) applyReads(s *S) {
	s.Call(c.callbacks.applyReads, s.Ctx)
}

func (c *C) applyWrites(s *S) {
	s.Call(c.callbacks.applyWrites, s.Ctx)
}

func (c *C) arrayAssign(s *S, n *semantic.ArrayAssign) {
	if n.Operator != ast.OpAssign {
		fail("Unsupported ArrayAssign operator '%s'", n.Operator)
	}
	val := c.expression(s, n.Value)
	c.assignTo(s, n.Operator, n.To, val)
}

func (c *C) assert(s *S, e *semantic.Assert) {
	cond := c.expression(s, e.Condition).SetName("assert_cond")
	s.If(cond, func(s *S) {
		c.Log(s, log.Fatal, "assert: "+fmt.Sprint(e.AST.Arguments[0]))
	})
}

func (c *C) assign(s *S, n *semantic.Assign) {
	c.assignTo(s, n.Operator, n.LHS, c.expression(s, n.RHS))
}

func (c *C) assignTo(s *S, op string, target semantic.Expression, val *codegen.Value) {
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

func (c *C) expressionAddr(s *S, target semantic.Expression) *codegen.Value {
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
			return s.Globals.Index(path...)
		case *semantic.Local:
			path = append(path, 0)
			revPath()
			return s.locals[n].Index(path...)
		case *semantic.Member:
			path = append(path, n.Field.Name())
			target = n.Object
			if semantic.IsReference(target.ExpressionType()) {
				path = append(path, RefValue, 0)
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
			v := s.Call(c.T.Maps[n.Type].Index, m, k, s.Scalar(false)).SetName("map_get")
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

func (c *C) branch(s *S, n *semantic.Branch) {
	cond := c.expression(s, n.Condition)
	onTrue := func(s *S) { c.block(s, n.True) }
	onFalse := func(s *S) { c.block(s, n.False) }
	if n.False == nil {
		onFalse = nil
	}
	s.IfElse(cond, onTrue, onFalse)
}

func (c *C) copy(s *S, n *semantic.Copy) {
	src := c.expression(s, n.Src)
	dst := c.expression(s, n.Dst)
	c.CopySlice(s, dst, src)
	if c.isFence {
		c.applyFence(s)
	}
}

func (c *C) declareLocal(s *S, n *semantic.DeclareLocal) {
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

func (c *C) fence(s *S, n *semantic.Fence) {
	if n.Statement != nil {
		c.isFence = true
		c.statement(s, n.Statement)
		if c.isFence {
			c.Fail("%v did not consume the fence", n.Statement)
		}
	} else {
		c.applyFence(s)
	}
}

func (c *C) applyFence(s *S) {
	c.plugins.foreach(func(p OnFenceListener) { p.OnFence(s) })
	c.applyWrites(s)
	c.isFence = false
}

func (c *C) iteration(s *S, n *semantic.Iteration) {
	from := c.expression(s, n.From)
	to := c.expression(s, n.To)
	one := s.One(from.Type())
	it := s.LocalInit(n.Iterator.Name(), from)
	s.While(func() *codegen.Value {
		return s.NotEqual(it.Load(), to)
	}, func() {
		s.enter(func(s *S) {
			s.locals[n.Iterator] = it
			c.block(s, n.Block)
			it.Store(s.Add(it.Load(), one))
		})
	})
}

func (c *C) mapAssign(s *S, n *semantic.MapAssign) {
	ty := n.To.Type
	m := c.expression(s, n.To.Map)
	k := c.expression(s, n.To.Index)
	v := c.expression(s, n.Value)
	dst := s.Call(c.T.Maps[ty].Index, m, k, s.Scalar(true))
	if ty := n.To.Type.ValueType; c.isRefCounted(ty) {
		c.reference(s, v, ty)
		c.release(s, dst.Load(), ty)
	}
	dst.Store(v)
}

func (c *C) mapIteration(s *S, n *semantic.MapIteration) {
	mapPtr := c.expression(s, n.Map)
	c.IterateMap(s, mapPtr, n.IndexIterator.Type, func(i, k, v *codegen.Value) {
		s.enter(func(s *S) {
			s.locals[n.IndexIterator] = i
			s.locals[n.KeyIterator] = k
			s.locals[n.ValueIterator] = v
			c.block(s, n.Block)
		})
	})
}

func (c *C) mapRemove(s *S, n *semantic.MapRemove) {
	ty := n.Type
	m := c.expression(s, n.Map)
	k := c.expression(s, n.Key)
	s.Call(c.T.Maps[ty].Remove, m, k)
}

func (c *C) read(s *S, n *semantic.Read) {
	// TODO
}

func (c *C) return_(s *S, n *semantic.Return) {
	var val *codegen.Value
	var ty semantic.Type
	if n.Value != nil {
		val = c.expression(s, n.Value)
		ty = n.Value.ExpressionType()
	} else if c.currentFunc.Signature.Return != semantic.VoidType {
		val = c.initialValue(s, c.currentFunc.Signature.Return)
		ty = c.currentFunc.Signature.Return
	} else {
		s.Return(nil)
		return
	}
	c.reference(s, val, ty)
	retTy := c.returnType(c.currentFunc) // <error, value>
	ret := s.Zero(retTy).Insert(retValue, val)
	s.Return(ret)
}

func (c *C) sliceAssign(s *S, n *semantic.SliceAssign) {
	if n.Operator != ast.OpAssign {
		fail("Unsupported slice composite assignment operator '%s'", n.Operator)
	}

	index := c.expression(s, n.To.Index).Cast(c.T.Uint64).SetName("index")
	slice := c.expression(s, n.To.Slice)

	elTy := n.To.Type.To
	targetTy := c.T.Target(elTy)
	captureTy := c.T.Capture(elTy)
	captureSize := s.Scalar(uint64(c.T.CaptureSize(elTy)))
	captureStride := s.Scalar(uint64(c.T.CaptureAllocaSize(elTy)))

	base := slice.Extract(SliceBase)
	offset := s.Mul(index, captureStride)
	subslice := slice.
		Insert(SliceBase, s.Add(base, offset)).
		Insert(SliceSize, captureSize).
		Insert(SliceCount, s.Scalar(uint64(1)))
	subslicePtr := s.LocalInit("subslice", subslice)

	write := func(el *codegen.Value) {
		c.SliceDataForWrite(s, subslicePtr, el.Type()).Store(el)
	}

	chainWrite := func(f func(el *codegen.Value, next func(el *codegen.Value))) {
		next := write
		write = func(el *codegen.Value) { f(el, next) }
	}

	if !c.Settings.WriteToApplicationPool {
		// Writes to the application pool are disabled by default.
		// This can be overridden with the WriteToApplicationPool setting.
		chainWrite(func(el *codegen.Value, next func(el *codegen.Value)) {
			pool := slice.Extract(SlicePool)
			appPool := s.Zero(c.T.PoolPtr)
			s.If(s.NotEqual(pool, appPool), func(s *S) {
				next(el) // Actually perform the write.
			})
		})
	}

	el := c.expression(s, n.Value)
	if targetTy == captureTy {
		write(el)
	} else {
		write(c.castTargetToCapture(s, elTy, el))
	}
}

func (c *C) switch_(s *S, n *semantic.Switch) {
	val := c.expression(s, n.Value)

	cases := make([]SwitchCase, len(n.Cases))
	for i, kase := range n.Cases {
		i, kase := i, kase
		cases[i] = SwitchCase{
			Conditions: func(s *S) []*codegen.Value {
				conds := make([]*codegen.Value, len(kase.Conditions))
				for i, cond := range kase.Conditions {
					conds[i] = c.equal(s, val, c.expression(s, cond))
				}
				return conds
			},
			Block: func(s *S) { c.block(s, kase.Block) },
		}
	}
	if n.Default != nil {
		s.Switch(cases, func(s *S) { c.block(s, n.Default) })
	} else {
		s.Switch(cases, nil)
	}
}

func (c *C) write(s *S, n *semantic.Write) {
	// TODO
}
