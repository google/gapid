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
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// LoadParameters loads the command's parameters from the context's arguments
// field and stores them into s.Parameters.
func (c *C) LoadParameters(s *S, f *semantic.Function) {
	params := s.Ctx.
		Index(0, ContextCmdArgs).
		Load().
		Cast(c.T.Pointer(c.T.CmdParams[f])).
		SetName("params")

	for _, p := range f.FullParameters {
		v := params.Index(0, p.Name()).Load().
			SetName(p.Name()).
			EmitDebug(p.Name())
		s.Parameters[p] = v
	}
}

func (c *C) command(f *semantic.Function) {
	if _, ok := c.commands[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)
	name := fmt.Sprintf("%v_%v", c.CurrentAPI().Name(), f.Name())
	loc := c.SourceLocationFor(f)
	out := c.M.
		Function(c.T.Void, name, c.T.CtxPtr).
		SetLocation(loc.File, loc.Line).
		SetParameterNames("gapil_context").
		LinkInternal()

	c.Build(out, func(s *S) {
		if debugFunctionCalls {
			c.LogI(s, f.Name())
		}

		c.LoadParameters(s, f)

		c.plugins.foreach(func(p OnBeginCommandListener) { p.OnBeginCommand(s, f) })

		c.applyReads(s)

		c.block(s, f.Block)

		c.plugins.foreach(func(p OnEndCommandListener) { p.OnEndCommand(s, f) })
	})
	c.commands[f] = out
	c.setCurrentFunction(old)
}

func (c *C) subroutine(f *semantic.Function) {
	if _, ok := c.subroutines[f]; ok {
		return
	}
	old := c.setCurrentFunction(f)

	params := f.CallParameters()
	paramTys := make([]codegen.Type, len(params)+1)
	paramNames := make([]string, len(params)+1)
	paramTys[0] = c.T.CtxPtr
	paramNames[0] = "ctx"
	for i, p := range params {
		paramTys[i+1] = c.T.Target(p.Type)
		paramNames[i+1] = p.Name()
	}
	resTy := c.T.Target(f.Return.Type)
	name := fmt.Sprintf("%v_%v", c.CurrentAPI().Name(), f.Name())

	loc := c.SourceLocationFor(f)
	out := c.M.
		Function(resTy, name, paramTys...).
		SetLocation(loc.File, loc.Line).
		SetParameterNames(paramNames...).
		LinkInternal()

	c.subroutines[f] = out
	c.Build(out, func(s *S) {
		if debugFunctionCalls {
			c.LogI(s, f.Name())
		}
		for i, p := range params {
			s.Parameters[p] = s.Parameter(i + 1).SetName(p.Name())
		}
		c.block(s, f.Block)
	})
	c.setCurrentFunction(old)
}

func (c *C) block(s *S, n *semantic.Block) {
	s.enter(func(s *S) {
		cst, _ := c.mappings.CST(n).(*cst.Branch)
		// Update source location to the opening brace.
		if l := c.SourceLocationForCST(cst.First()); l.IsValid() {
			s.SetLocation(l.Line, l.Column)
		}
		// Emit statements.
		for _, st := range n.Statements {
			if !c.statement(s, st) {
				break
			}
		}
		// Update source location to the closing brace.
		if l := c.SourceLocationForCST(cst.Last()); l.IsValid() {
			s.SetLocation(l.Line, l.Column)
		}
	})
}

func (c *C) statement(s *S, n semantic.Statement) bool {
	c.pushStatement(s, n)
	defer c.popStatement(s)

	if debugStatements {
		c.LogI(s, fmt.Sprintf("%T %+v", n, n))
	}

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
	case *semantic.MapClear:
		c.mapClear(s, n)
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
	for s := s; s != nil; s = s.parent {
		s.exit()
	}
	s.Throw(s.Scalar(ErrAborted))
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
	c.doAssign(s, n.Operator, n.To, n.Value)
}

func (c *C) assert(s *S, e *semantic.Assert) {
	cond := c.expression(s, e.Condition).SetName("assert_cond")
	s.If(s.Not(cond), func(s *S) {
		// Gather all the named expressions that we can print.
		type namedVal struct {
			name string
			val  semantic.Expression
		}
		vals := []namedVal{}
		var gather func(semantic.Node)
		gather = func(n semantic.Node) {
			switch n := n.(type) {
			case *semantic.BinaryOp, *semantic.UnaryOp:
				semantic.Visit(n, gather)
			case *semantic.Member:
				vals = append(vals, namedVal{n.Field.Name(), n})
			case *semantic.Field:
				vals = append(vals, namedVal{n.Name(), n})
			case *semantic.Global:
				vals = append(vals, namedVal{n.Name(), n})
			case *semantic.Local:
				vals = append(vals, namedVal{n.Name(), n})
			}
		}
		semantic.Visit(e.Condition, gather)

		msg := strings.Builder{}
		msg.WriteString(e.Message)
		msg.WriteString(" ")

		args := []interface{}{}
		for _, nv := range vals {
			f, v := s.PrintfSpecifier(c.expression(s, nv.val))
			msg.WriteString(nv.name)
			msg.WriteString(": ")
			msg.WriteString(f)
			for _, v := range v {
				args = append(args, v)
			}
		}
		c.Log(s, log.Fatal, msg.String(), args...)
	})
}

func (c *C) assign(s *S, n *semantic.Assign) {
	c.doAssign(s, n.Operator, n.LHS, n.RHS)
}

func (c *C) doAssign(s *S, op string, lhs, rhs semantic.Expression) {
	val := c.expression(s, rhs)

	if _, isIgnore := lhs.(*semantic.Ignore); isIgnore {
		return
	}

	val = c.doCast(s, lhs.ExpressionType(), rhs.ExpressionType(), val)

	dst := c.expressionAddr(s, lhs)
	switch op {
	case ast.OpAssign:
		c.reference(s, val, lhs.ExpressionType())
		c.deferRelease(s, dst.Load(), lhs.ExpressionType())
		dst.Store(val)
	case ast.OpAssignPlus:
		val := c.doBinaryOp(s, ast.OpPlus, dst.Load(), val)
		c.reference(s, val, lhs.ExpressionType())
		c.deferRelease(s, dst.Load(), lhs.ExpressionType())
		dst.Store(val)
	case ast.OpAssignMinus:
		val := c.doBinaryOp(s, ast.OpMinus, dst.Load(), val)
		c.reference(s, val, lhs.ExpressionType())
		c.deferRelease(s, dst.Load(), lhs.ExpressionType())
		dst.Store(val)
	default:
		fail("Unsupported composite assignment operator '%s'", op)
	}
}

func (c *C) expressionAddr(s *S, target semantic.Expression) *codegen.Value {
	path := []codegen.ValueIndexOrName{}
	revPath := func() []codegen.ValueIndexOrName {
		for i, c, m := 0, len(path), len(path)/2; i < m; i++ {
			j := c - i - 1
			path[i], path[j] = path[j], path[i]
		}
		return path
	}
	for {
		switch n := target.(type) {
		case *semantic.Global:
			path = append(path, n.Name(), c.CurrentAPI().Name(), 0)
			return s.Globals.Index(revPath()...)
		case *semantic.Local:
			if isLocalImmutable(n) {
				fail("Cannot take the address of an immutable local")
			}
			path = append(path, 0)
			return s.locals[n].val.Index(revPath()...)
		case *semantic.Member:
			path = append(path, n.Field.Name())
			target = n.Object
			if semantic.IsReference(target.ExpressionType()) {
				path = append(path, RefValue, 0)
				return c.expression(s, target).Index(revPath()...)
			}
		case *semantic.ArrayIndex:
			path = append(path, c.expression(s, n.Index))
			target = n.Array
		case *semantic.MapIndex:
			path = append(path, 0)
			m := c.expression(s, n.Map)
			k := c.expression(s, n.Index)
			v := s.Call(c.T.Maps[n.Type].Index, m, s.Ctx, k, s.Scalar(false)).SetName("map_get")
			return v.Index(revPath()...)
		case *semantic.Unknown:
			target = n.Inferred
		case *semantic.Ignore:
			return nil
		default:
			path = append(path, 0)
			return s.LocalInit("tmp", c.expression(s, n)).Index(revPath()...)
		}
	}
}

func (c *C) branch(s *S, n *semantic.Branch) {
	cond := c.expression(s, n.Condition)
	onTrue := func(s *S) { c.block(s, n.True) }
	if n.False == nil {
		s.If(cond, onTrue)
	} else {
		onFalse := func(s *S) { c.block(s, n.False) }
		s.IfElse(cond, onTrue, onFalse)
	}
}

func (c *C) copy(s *S, n *semantic.Copy) {
	src := c.expression(s, n.Src)
	dst := c.expression(s, n.Dst)
	ty := n.Src.ExpressionType().(*semantic.Slice)

	// Adjust slice lengths to the min of src and dst.
	// This is handled automatically in CopySlice, but we need correct counts
	// for the plugin callbacks.
	srcCnt := src.Extract(SliceCount)
	dstCnt := dst.Extract(SliceCount)
	cnt := s.Select(s.LessThan(srcCnt, dstCnt), srcCnt, dstCnt)
	size := s.Mul(cnt, s.SizeOf(c.T.Capture(ty.To)))
	src = src.Insert(SliceCount, cnt).Insert(SliceSize, size)
	dst = dst.Insert(SliceCount, cnt).Insert(SliceSize, size)

	c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, src, ty) })
	c.CopySlice(s, dst, src)
	if c.isFence {
		c.applyFence(s)
	}
	c.plugins.foreach(func(p OnWriteListener) { p.OnWrite(s, dst, ty) })
}

func (c *C) declareLocal(s *S, n *semantic.DeclareLocal) {
	var def *codegen.Value
	if n.Local.Value != nil {
		def = c.expression(s, n.Local.Value)
	} else {
		def = c.initialValue(s, n.Local.Type)
	}
	var l local
	if isLocalImmutable(n.Local) {
		l = local{def, false}
	} else {
		l = local{s.LocalInit(n.Local.Name(), def), true}
	}
	l.val.EmitDebug(n.Local.Name())
	s.locals[n.Local] = l
}

func isLocalImmutable(l *semantic.Local) bool {
	switch l.Type.(type) {
	case *semantic.Class:
		return false
	default:
		return true
	}
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
			s.locals[n.Iterator] = local{it, true}
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
	c.reference(s, v, ty.ValueType)
	c.release(s, dst.Load(), ty.ValueType)
	dst.Store(v)
}

func (c *C) mapIteration(s *S, n *semantic.MapIteration) {
	mapPtr := c.expression(s, n.Map)
	c.IterateMap(s, mapPtr, n.IndexIterator.Type, func(i, k, v *codegen.Value) {
		s.enter(func(s *S) {
			s.locals[n.IndexIterator] = local{i, true}
			s.locals[n.KeyIterator] = local{k, true}
			s.locals[n.ValueIterator] = local{v, true}
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

func (c *C) mapClear(s *S, n *semantic.MapClear) {
	ty := n.Type
	m := c.expression(s, n.Map)
	s.Call(c.T.Maps[ty].ClearKeep, m)
}

func (c *C) read(s *S, n *semantic.Read) {
	slice := c.expression(s, n.Slice)
	ty := n.Slice.ExpressionType().(*semantic.Slice)
	c.plugins.foreach(func(p OnReadListener) { p.OnRead(s, slice, ty) })
}

func (c *C) return_(s *S, n *semantic.Return) {
	switch {
	case c.currentFunc.Subroutine:
		var val *codegen.Value
		var ty semantic.Type
		switch {
		case n.Value != nil:
			val = c.expression(s, n.Value)
			ty = n.Value.ExpressionType()
		case c.currentFunc.Signature.Return != semantic.VoidType:
			val = c.initialValue(s, c.currentFunc.Signature.Return)
			ty = c.currentFunc.Signature.Return
		default:
			s.Return(nil)
			return
		}

		val = val.Cast(c.T.Target(n.Function.Return.Type))

		c.reference(s, val, ty)
		s.Return(val)

	default:
		// Commands have no return
		s.Return(nil)
	}
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
	captureSize := s.Scalar(uint64(c.T.CaptureTypes.SizeOf(elTy)))
	captureStride := s.Scalar(uint64(c.T.CaptureTypes.StrideOf(elTy)))

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
			s.If(s.NotEqual(pool, s.Zero(pool.Type())), func(s *S) {
				next(el) // Actually perform the write.
			})
		})
	}

	// Regardless of whether we locally update the app pool or not, we always
	// want to inform the plugins that there is a slice write.
	chainWrite(func(el *codegen.Value, next func(el *codegen.Value)) {
		next(el)
		c.plugins.foreach(func(p OnWriteListener) { p.OnWrite(s, subslice, n.To.Type) })
	})

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
	slice := c.expression(s, n.Slice)
	ty := n.Slice.ExpressionType().(*semantic.Slice)
	c.plugins.foreach(func(p OnWriteListener) { p.OnWrite(s, slice, ty) })
}
