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

// Package format registers and implements the "format" apic command.
//
// The format command re-formats an API file to a consistent style.
package format

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

const tabwriterFlags = 0 // | tabwriter.Debug

// Format prints the full re-formatted AST tree to w.
func Format(api *ast.API, m *ast.Mappings, w io.Writer) {
	p := printer{Mappings: m}
	// traverse the AST, applying markup for the CST nodes.
	p.markup(api)

	// Writers are chained like so:
	//    indenter -> [tabwriter -> tabwriter -> ...] -> wsTrimmer -> w
	// initially there are no tabwriters, so the initial chain is:
	//    indenter -> wsTrimmer -> w
	trimmer := &wsTrimmer{out: w}
	p.indenter.out = trimmer
	p.out = trimmer

	// print the CST using the markup generated from the AST.
	p.print(m.CST(api))
}

type printer struct {
	*ast.Mappings
	tabbers    []*tabwriter.Writer
	indenter   indenter
	out        io.Writer
	injections map[injectKey]string
	aligns     map[cst.Node]struct{}
}

// isNewline returns true if n starts on a new line.
func isNewline(n cst.Node) bool {
	for _, s := range n.Prefix() {
		if strings.Contains(s.Tok().String(), "\n") {
			return true
		}
	}
	if b, ok := n.(*cst.Branch); ok && len(b.Children) > 0 {
		return isNewline(b.Children[0])
	}
	return false
}

// markup populates the cst.Node maps with information based on the ast tree.
func (p *printer) markup(n ast.Node) {
	switch n := n.(type) {
	case *ast.Annotation:
		p.inject(n, beforeSuffix, "•")
		if len(n.Arguments) > 0 {
			for _, a := range n.Arguments[1:] {
				p.inject(a, beforePrefix, "•")
			}
		}

	case *ast.API:
		p.align(n)
		if n.Index != nil {
			p.inject(n.Index, afterPrefix, "•")
		}

	case *ast.Assign:
		p.inject(n.LHS, beforeSuffix, "•")
		p.inject(n.RHS, afterPrefix, "•")

	case *ast.BinaryOp:
		if n.Operator != ":" {
			p.inject(n.LHS, beforeSuffix, "•")
			p.inject(n.RHS, afterPrefix, "•")
		}

	case *ast.Block:
		p.align(n)

		// •{}•
		p.inject(n, beforePrefix, "•")
		p.inject(n, afterSuffix, "•")

		cst := p.CST(n).(*cst.Branch)
		if c := len(cst.Children); c > 0 {
			if cst.Children[0].Tok().String() == ast.OpBlockStart {
				// {•» statements... «•}
				p.inject(cst.Children[0], beforeSuffix, "•»")
				p.inject(cst.Children[c-1], afterPrefix, "«•")
			} else {
				// »statement«
				p.inject(n, beforePrefix, "»")
				p.inject(n, afterSuffix, "«")
			}
		}

	case *ast.Branch:
		// indent multi-line conditionals
		p.inject(n.Condition, beforePrefix, "»»")
		p.inject(n.Condition, afterSuffix, "««")

		p.inject(n.Condition, afterPrefix, "•")

	case *ast.Call:
		p.align(n)
		if c := len(n.Arguments); c > 0 {
			for i, v := range n.Arguments {
				if i == 0 {
					p.inject(v, afterPrefix, "\t")
				} else {
					p.inject(v, afterPrefix, "•\t")
				}
			}
			p.inject(n.Target, beforeSuffix, "»")
			p.inject(n.Arguments[c-1], beforeSuffix, "«")
		}

	case *ast.Case:
		p.inject(n, afterPrefix, "•")
		if c := len(n.Conditions); c > 0 {
			for _, c := range n.Conditions {
				p.inject(c, afterPrefix, "•")
			}
			p.inject(n.Conditions[0], afterPrefix, "»»")
			p.inject(n.Conditions[c-1], beforeSuffix, "««")
		}
		blockCST := p.CST(n.Block)
		if !isNewline(blockCST) {
			// align:
			// case Foo: |•{ ... }•
			// case Blah:|•{ ... }•
			p.inject(n.Block, afterPrefix, "\t")
		}

	case *ast.Class:
		p.align(n)
		p.inject(n.Name, afterPrefix, "•")
		p.inject(n.Name, beforeSuffix, "•")
		if c := len(n.Fields); c > 0 {
			p.inject(n.Fields[0], beforePrefix, "•»")
			p.inject(n.Fields[c-1], beforeSuffix, "«•")
		}
	case *ast.Clear:

	case *ast.Delete:
		p.inject(n.Key, beforePrefix, "•")

	case *ast.Default:
		p.inject(n, afterPrefix, "•")
		blockCST := p.CST(n.Block)
		if !isNewline(blockCST) {
			// align:
			// case Foo: |•{ ... }•
			// case Blah:|•{ ... }•
			p.inject(n.Block, afterPrefix, "\t")
		}

	case *ast.Definition:
		p.inject(n.Name, beforePrefix, "•\t")
		p.inject(n.Name, afterSuffix, "•\t")

	case *ast.DeclareLocal:
		p.inject(n.Name, beforeSuffix, "•")
		p.inject(n.RHS, afterPrefix, "•")

	case *ast.Enum:
		p.align(n)
		p.inject(n.Name, afterPrefix, "•")
		p.inject(n.Name, beforeSuffix, "•")
		if c := len(n.Entries); c > 0 {
			p.inject(n.Name, beforeSuffix, "»")
			p.inject(n.Entries[c-1], beforeSuffix, "«")
		}

	case *ast.EnumEntry:
		//name[A•   ]=[B•]value[C    ]•// comment
		p.inject(n.Name, beforeSuffix, "\t•") // A
		p.inject(n.Value, afterPrefix, "\t•") // B
		p.inject(n, beforeSuffix, "\t•")      // C

	case *ast.Field:
		//•type[A•   ]name[B•   ]=[C•]default[D    ]•// comment
		p.inject(n.Type, beforePrefix, "•")
		p.inject(n.Name, afterPrefix, "\t•") // A
		if n.Default != nil {
			p.inject(n.Name, beforeSuffix, "\t•")   // B
			p.inject(n.Default, afterPrefix, "•")   // C
			p.inject(n.Default, beforeSuffix, "\t") // D
		} else {
			p.inject(n.Name, beforeSuffix, "\t\t")
		}

	case *ast.Function:
		p.align(n)
		c := len(n.Parameters) - 1
		ret := n.Parameters[c]
		p.inject(n.Generic, afterPrefix, "•")
		p.inject(ret, afterPrefix, "•")
		for i, v := range n.Parameters[:c] {
			if i == 0 {
				p.inject(v.Type, afterPrefix, "\t")
			} else {
				p.inject(v.Type, afterPrefix, "•\t")
			}
			p.inject(v.Name, afterPrefix, "•\t")
		}
		if c > 0 {
			p.inject(n.Generic, afterPrefix, "»»")
			p.inject(n.Parameters[c-1], afterSuffix, "««")
		}

	case *ast.Generic:
		if len(n.Arguments) > 0 {
			for _, a := range n.Arguments[1:] {
				p.inject(a, afterPrefix, "•")
			}
		}

	case *ast.Iteration:
		p.inject(n.Variable, afterPrefix, "•")
		p.inject(n.Variable, beforeSuffix, "•")
		p.inject(n.Iterable, afterPrefix, "•")

	case *ast.MapIteration:
		p.inject(n.IndexVariable, afterPrefix, "•")
		p.inject(n.IndexVariable, beforeSuffix, "•")
		p.inject(n.KeyVariable, afterPrefix, "•")
		p.inject(n.KeyVariable, beforeSuffix, "•")
		p.inject(n.ValueVariable, afterPrefix, "•")
		p.inject(n.ValueVariable, beforeSuffix, "•")
		p.inject(n.Map, afterPrefix, "•")

	case *ast.Import:
		p.inject(n.Path, afterPrefix, "•")

	case *ast.NamedArg:
		p.inject(n.Value, afterPrefix, "•\t")

	case *ast.PointerType:
		if n.Const {
			p.inject(n.To, beforeSuffix, "•")
		}

	case *ast.PreConst:
		p.inject(n.Type, afterPrefix, "•")

	case *ast.Pseudonym:
		p.inject(n.To, afterPrefix, "•")
		p.inject(n.Name, afterPrefix, "\t•")

	case *ast.Return:
		p.inject(n.Value, afterPrefix, "•")

	case *ast.Switch:
		cst := p.CST(n).(*cst.Branch)
		p.align(n)
		p.inject(n.Value, afterPrefix, "•")
		p.inject(n.Value, beforeSuffix, "•")
		p.inject(n.Value, afterSuffix, "»")
		p.inject(cst.Children[len(cst.Children)-1], beforePrefix, "«")
	}

	ast.Visit(n, p.markup)
}

// print traverses and prints the CST, applying modifications based on the
// markup pass.
func (p *printer) print(n cst.Node) {
	// emit any beforePrefix injections.
	if s, ok := p.injections[injectKey{n, beforePrefix}]; ok {
		p.write(s)
	}

	// print the prefix comments.
	p.separator(n.Prefix())

	// emit any afterPrefix injections.
	if s, ok := p.injections[injectKey{n, afterPrefix}]; ok {
		p.write(s)
	}

	switch n := n.(type) {
	case *cst.Branch:
		// if this node should align the children, push a new tabber.
		_, align := p.aligns[n]
		if align {
			p.pushTabber()
		}
		// print the child CST nodes.
		for _, c := range n.Children {
			p.print(c)
		}
		if align {
			p.popTabber()
		}
	case *cst.Leaf:
		p.write(n.Token.String())

	default:
		panic("Unknown parse node type")
	}

	// emit any beforeSuffix injections.
	if s, ok := p.injections[injectKey{n, beforeSuffix}]; ok {
		p.write(s)
	}

	// print the suffix comments.
	p.separator(n.Suffix())

	// emit any afterSuffix injections.
	if s, ok := p.injections[injectKey{n, afterSuffix}]; ok {
		p.write(s)
	}
}

// write prints the string s to the indenter which is always the head of the
// writer chain.
func (p *printer) write(s string) {
	p.indenter.Write([]byte(s))
}

// pushTabber injects a new tabwriter after the indenter in the writer chain.
func (p *printer) pushTabber() {
	t := tabwriter.NewWriter(p.indenter.out, 0, 2, 0, ' ', tabwriterFlags)
	p.tabbers = append(p.tabbers, t)
	p.indenter.out = t
}

// popTabber removes the tabwriter after the indenter in the writer chain.
func (p *printer) popTabber() {
	c := len(p.tabbers)
	p.tabbers[c-1].Flush()
	p.tabbers = p.tabbers[:c-1]
	if c := len(p.tabbers); c > 0 {
		p.indenter.out = p.tabbers[c-1]
	} else {
		p.indenter.out = p.out
	}
}

type position int

const (
	beforePrefix = position(iota)
	afterPrefix
	beforeSuffix
	afterSuffix
)

type injectKey struct {
	n cst.Node
	p position
}

// inject adds s using r relative to the AST or CST node n.
func (p *printer) inject(n interface{}, r position, s string) {
	if p.injections == nil {
		p.injections = make(map[injectKey]string)
	}
	var key injectKey
	key.p = r
	switch n := n.(type) {
	case cst.Node:
		key.n = n
	case ast.Node:
		key.n = p.CST(n)
	default:
		panic(fmt.Errorf("n must be a cst.Node or ast.Node. Got %T", n))
	}
	p.injections[key] = p.injections[key] + s
}

// align marks up n's children to be printed with a new tabwriter.
func (p *printer) align(n ast.Node) {
	if p.aligns == nil {
		p.aligns = make(map[cst.Node]struct{})
	}
	p.aligns[p.CST(n)] = struct{}{}
}

// separator writes sep to the indenter iff it is a comment.
// All comments are preceeded with a soft whitespace.
func (p *printer) separator(sep cst.Separator) {
	for _, sep := range sep {
		s := sep.Tok().String()
		switch {
		case strings.HasPrefix(s, "//"), strings.HasPrefix(s, "/*"):
			p.write("•")
			// Write the '/' to the indenter to get new lines indented.
			p.write(s[:1])
			// Write the rest of the comment skipping the indenter, as we don't want
			// to change new-line indentation within the comments.
			p.indenter.out.Write([]byte(s[1:]))

		case strings.HasPrefix(s, "\n"):
			p.write(s)
		}
	}
}
