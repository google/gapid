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

// removeDeadCode removes if statement blocks when the conditional is a literal
// false. The primary use of this code is to eliminate conditional blocks inside
// macros that are directly dependent on a boolean parameter.
// TODO: Consider deleting this code once macros are removed.
func removeDeadCode(rv *resolver, block *semantic.Block) {
	d := deadCodeStripper{
		locals: map[*semantic.Local]struct{}{},
		decls:  map[*semantic.Local]localDeclInfo{},
	}
	d.doBlock(rv, block)
	d.killUnusedLocals(rv, block)
}

type localDeclInfo struct {
	block *semantic.Block
	decl  *semantic.DeclareLocal
}

type deadCodeStripper struct {
	// locals are all the locals that were used for dead-code stripping
	locals map[*semantic.Local]struct{}
	// all local declarations found during traversal
	decls map[*semantic.Local]localDeclInfo
}

func (d *deadCodeStripper) doBlock(rv *resolver, block *semantic.Block) {
	if block == nil {
		return
	}
	rv.with(semantic.VoidType, func() {
		for i := 0; i < len(block.Statements); i++ {
			switch n := block.Statements[i].(type) {
			case *semantic.Block:
				d.doBlock(rv, n)
			case *semantic.Branch:
				d.doBlock(rv, n.True)
				if n.False != nil {
					d.doBlock(rv, n.False)
				}
				if d.doBranch(n.Condition, n, block, i) {
					i-- // statement may have been removed.
				}
			case *semantic.Switch:
				d.doBlock(rv, n.Default)
				for _, c := range n.Cases {
					d.doBlock(rv, c.Block)
				}
			case *semantic.DeclareLocal:
				// record the local declaration for killUnusedLocals().
				d.decls[n.Local] = localDeclInfo{block, n}
			}
		}
	})
}

func (d *deadCodeStripper) doBranch(cond semantic.Expression, branch *semantic.Branch, block *semantic.Block, index int) bool {
	switch cond := cond.(type) {
	case semantic.BoolValue:
		if cond {
			if branch.False != nil {
				// remove the false block
				foreachLocal(branch.False, d.potentiallyRemove)
				branch.False = nil
			}
		} else {
			// remove the true block
			foreachLocal(branch.True, d.potentiallyRemove)
			// flip condition so that the true block becomes the false block.
			branch.Condition = &semantic.UnaryOp{Operator: ast.OpNot, Expression: branch.Condition}
			branch.True, branch.False = branch.False, nil
		}

		if branch.True != nil {
			// move statements up to parent block
			block.Statements.Replace(index, 1, branch.True)
		} else {
			// remove if block all together
			block.Statements.Replace(index, 1)
		}
		return true
	case *semantic.Local:
		if d.doBranch(cond.Value, branch, block, index) {
			d.potentiallyRemove(cond)
		}
	}
	return false
}

func (d *deadCodeStripper) potentiallyRemove(n *semantic.Local) {
	d.locals[n] = struct{}{}
}

// killUnusedLocals removes all local declarations that were only referenced
// by if statements that have been folded.
func (d *deadCodeStripper) killUnusedLocals(rv *resolver, block *semantic.Block) {
	// remove from d.locals all the locals still in use
	foreachLocal(block, func(n *semantic.Local) { delete(d.locals, n) })
	// d.locals now contains all the locals that were folded, and are not used
	// anywhere else in the block or sub-blocks.
	for unused := range d.locals {
		info, ok := d.decls[unused]
		if !ok {
			rv.icef(unused.Declaration.AST, "Declaration was not picked up!")
			continue
		}
		info.block.Statements.Remove(info.decl)
	}
}

func foreachLocal(n semantic.Node, f func(n *semantic.Local)) {
	var visit func(n semantic.Node)
	visit = func(n semantic.Node) {
		switch n := n.(type) {
		case semantic.Type, *semantic.Call, *semantic.Function:
			// Don't traverse into these.
		case *semantic.DeclareLocal:
			visit(n.Local.Value)
		case *semantic.Local:
			f(n)
		default:
			foreachLocal(n, f)
		}
	}
	semantic.Visit(n, visit)
}
