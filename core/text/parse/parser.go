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

package parse

import (
	"github.com/google/gapid/core/text/parse/cst"
)

// RootParser is a function that is passed to Parse. It is handed the Branch to
// fill in and the Parser to fill it from, and must either succeed or add an
// error to the parser.
type RootParser func(*Parser, *cst.Branch)

// BranchParser is a function that is passed to ParseBranch. It is handed the
// Branch to fill in, and must either succeed or add an error to the parser.
type BranchParser func(*cst.Branch)

// LeafParser is a function that is passed to ParseLeaf. It is handed the
// Leaf to fill in and the Parser to fill it from, and must either succeed or
// add an error to the parser.
type LeafParser func(*cst.Leaf)

// Parse is the main entry point to the parse library.
// Given a root parse function, the input string and the Skip controller, it builds
// and initializes a Parser, runs the root using it, verifies it worked
// correctly and then returns the errors generated if any.
func Parse(filename, data string, skip Skip, parse RootParser) []Error {
	p := &Parser{skip: skip}
	p.setData(filename, data)
	p.parse(parse)
	return p.Errors
}

// Parser contains all the context needed while parsing.
// They are built for you by the Parse function.
type Parser struct {
	Reader               // The token reader for this parser.
	Errors ErrorList     // The set of errors generated during the parse.
	skip   Skip          // The whitespace skipping function.
	prefix cst.Separator // The currently skipped prefix separator.
	suffix cst.Separator // The currently skipped suffix separator.
	last   cst.Node      // The last node fully parsed, potential suffix target
}

func (p *Parser) parse(root RootParser) {
	defer func() {
		err := recover()
		if err != nil && err != AbortParse {
			panic(err)
		}
	}()
	anchor := &cst.Branch{}
	anchor.AddPrefix(p.skip(p, SkipPrefix))
	root(p, anchor)
	if len(p.suffix) > 0 {
		anchor.AddSuffix(p.suffix)
	}
	if len(p.prefix) > 0 {
		// This is a trailing "prefix" for a node that is never going to arrive, so
		// we treat it as a suffix of the cst root instead.
		anchor.AddSuffix(p.prefix)
	}
	if !p.IsEOF() {
		p.Error("Unexpected input at end of parse")
	}
}

func (p *Parser) addChild(in *cst.Branch, child cst.Node) {
	if p.suffix != nil {
		p.last.AddSuffix(p.suffix)
		p.suffix = nil
	}
	child.AddPrefix(p.prefix)
	p.prefix = nil
	in.Children = append(in.Children, child)
	child.SetParent(in)
}

// ParseLeaf adds a new Leaf to b and then calls the do function to
// parse the Leaf.
// If do is nil, a leaf will be built with the current unconsumed input.
func (p *Parser) ParseLeaf(b *cst.Branch, do LeafParser) {
	l := &cst.Leaf{}
	p.addChild(b, l)
	if do != nil {
		do(l)
	}
	if p.offset != p.cursor {
		l.Token = p.Consume()
	}
	p.suffix = p.skip(p, SkipSuffix)
	p.prefix = p.skip(p, SkipPrefix)
	p.last = l
}

// ParseBranch adds a new Branch to b and then calls the do function to
// parse the branch.
// This is called recursively to build the node tree.
func (p *Parser) ParseBranch(b *cst.Branch, do BranchParser) {
	if p.offset != p.cursor {
		p.Error("Starting ParseBranch with parsed but unconsumed tokens")
	}
	n := &cst.Branch{}
	p.addChild(b, n)
	do(n)
	if p.offset != p.cursor {
		p.Error("Finishing ParseBranch with parsed but unconsumed tokens")
	}
	p.last = n
}

// Extend inserts a new branch between n and its parent, and calls do() with
// the newly insterted branch.
//
// Extend will transform:
//
//	n.parent ──> n
//
// to:
//
//	n.parent ──> b ──> n
//
// where b is passed as the second argument to do().
func (p *Parser) Extend(n cst.Node, do BranchParser) {
	if n == nil {
		p.Error("invalid cst node")
		return
	}
	base := n.Parent()
	if base == nil {
		p.Error("Branch did not have a parent")
		return
	}

	g := &cst.Branch{}
	g.SetParent(base)
	g.Children = []cst.Node{n}
	n.SetParent(g)

	// Replace n with g in base.children.
	for i := len(base.Children) - 1; i >= 0; i-- {
		if base.Children[i] == n {
			base.Children[i] = g
			do(g)
			return
		}
	}

	p.Error("Branches parent did not contain branch")
}

// Error adds a new error to the parser error list. It will attempt to consume
// a token from the reader to act as a place holder, and also to ensure
// progress is made in the presence of errors.
func (p *Parser) Error(message string, args ...interface{}) {
	at := cst.Fragment(nil)
	if p.IsEOF() {
		at = p.last
	}
	p.Errors.Add(&p.Reader, at, message, args...)
}

// ErrorAt is like Error, except because it is handed a fragment, it will not
// try to consume anything itself.
func (p *Parser) ErrorAt(loc cst.Fragment, message string, args ...interface{}) {
	p.Errors.Add(&p.Reader, loc, message, args...)
}

// Expected is a wrapper around p.ErrorAt for the very common case of an unexpected
// input. It uses value as the expected input, and parses a token of the stream
// for the unexpected actual input.
func (p *Parser) Expected(value string) {
	invalid := p.GuessNextToken()
	p.Errors.Add(&p.Reader, invalid, "Expected \"%s\" got \"%s\"", value, invalid.String())
}
