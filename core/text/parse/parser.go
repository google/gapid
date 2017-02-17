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

import "sync"

// BranchParser is a function that is passed to ParseBranch. It is handed the
// Branch to fill in and the Parser to fill it from, and must either succeed or
// add an error to the parser.
type BranchParser func(p *Parser, cst *Branch)

// LeafParser is a function that is passed to ParseLeaf. It is handed the
// Leaf to fill in and the Parser to fill it from, and must either succeed or
// add an error to the parser.
type LeafParser func(p *Parser, cst *Leaf)

// CSTMap is the interface to an object into which ast<->cts mappings are stored.
type CSTMap interface {
	// SetCST is called to map an ast node to a CST node.
	SetCST(ast interface{}, cst Node)
	// CST is called to lookup the CST mapping for an AST node.
	CST(ast interface{}) Node
}

// cstMap is a simple implementation of CSTMap that just stores the mappings.
type cstMap struct {
	mappings map[interface{}]Node
	mutex    sync.Mutex
}

func (m *cstMap) SetCST(ast interface{}, cst Node) {
	m.mutex.Lock()
	m.mappings[ast] = cst
	m.mutex.Unlock()
}

func (m *cstMap) CST(ast interface{}) Node {
	m.mutex.Lock()
	out := m.mappings[ast]
	m.mutex.Unlock()
	return out
}

// NewCSTMap returns a new CSTMap instance.
func NewCSTMap() CSTMap {
	return &cstMap{mappings: make(map[interface{}]Node)}
}

// Parse is the main entry point to the parse library.
// Given a root parse function, the input string and the Skip controller, it builds
// and initializes a Parser, runs the root using it, verifies it worked
// correctly and then returns the errors generated if any.
func Parse(root BranchParser, filename, data string, skip Skip, m CSTMap) []Error {
	if m == nil {
		// We have to have a mapping store because of the extend call...
		m = NewCSTMap()
	}
	p := &Parser{
		skip:   skip,
		CSTMap: m,
	}
	p.setData(filename, data)
	p.parse(root)
	return p.Errors
}

// Parser contains all the context needed while parsing.
// They are built for you by the Parse function.
type Parser struct {
	Reader           // The token reader for this parser.
	CSTMap           // A map of cst nodes to ast nodes
	Errors ErrorList // The set of errors generated during the parse.
	skip   Skip      // The whitespace skipping function.
	prefix Separator // The currently skipped prefix separator.
	suffix Separator // The currently skipped suffix separator.
	last   Node      // The last node fully parsed, potential suffix target
}

func (p *Parser) parse(root BranchParser) {
	defer func() {
		err := recover()
		if err != nil && err != AbortParse {
			panic(err)
		}
	}()
	anchor := &Branch{}
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

func (p *Parser) addChild(in *Branch, child Node) {
	if p.suffix != nil {
		p.last.AddSuffix(p.suffix)
		p.suffix = nil
	}
	child.AddPrefix(p.prefix)
	p.prefix = nil
	in.Children = append(in.Children, child)
	p.setParent(child, in)
}

// ParseLeaf adds a new Leaf to cst and then calls the do function to
// parse the Leaf.
// If do is nil, a leaf will be built with the current unconsumed input.
func (p *Parser) ParseLeaf(cst *Branch, do LeafParser) {
	l := &Leaf{}
	p.addChild(cst, l)
	if do != nil {
		do(p, l)
	}
	if p.offset != p.cursor {
		l.SetToken(p.Consume())
	}
	p.suffix = p.skip(p, SkipSuffix)
	p.prefix = p.skip(p, SkipPrefix)
	p.last = l
}

// ParseBranch adds a new Branch to cst and then calls the do function to
// parse the branch.
// This is called recursively to build the node tree.
func (p *Parser) ParseBranch(cst *Branch, do BranchParser) {
	if p.offset != p.cursor {
		p.Error("Starting ParseBranch with parsed but unconsumed tokens")
	}
	b := &Branch{}
	p.addChild(cst, b)
	do(p, b)
	if p.offset != p.cursor {
		p.Error("Finishing ParseBranch with parsed but unconsumed tokens")
	}
	p.last = b
}

// Extend inserts a new branch between ast and its parent, and calls do() with
// the newly insterted branch.
//
// Extend will transform:
//     n.parent ──> n
// to:
//     n.parent ──> b ──> n
// where b is passed as the second argument to do().
func (p *Parser) Extend(ast interface{}, do BranchParser) {
	n := p.CST(ast)
	if n == nil {
		p.Error("ast node has not cst mapping")
		return
	}
	base := n.Parent()
	if base == nil {
		p.Error("Branch did not have a parent")
		return
	}

	g := &Branch{}
	g.parent = base
	g.Children = []Node{n}
	p.setParent(n, g)

	// Replace n with g in base.children.
	for i := len(base.Children) - 1; i >= 0; i-- {
		if base.Children[i] == n {
			base.Children[i] = g
			do(p, g)
			return
		}
	}

	p.Error("Branches parent did not contain branch")
}

func (p *Parser) setParent(child Node, parent *Branch) {
	switch n := child.(type) {
	case *Branch:
		n.parent = parent
	case *Leaf:
		n.parent = parent
	default:
		p.Error("Cannot set parent of node type %T", n)
	}
}

// Error adds a new error to the parser error list. It will attempt to consume
// a token from the reader to act as a place holder, and also to ensure
// progress is made in the presence of errors.
func (p *Parser) Error(message string, args ...interface{}) {
	at := Fragment(nil)
	if p.IsEOF() {
		at = p.last
	}
	p.Errors.Add(&p.Reader, at, message, args...)
}

// ErrorAt is like Error, except because it is handed a fragment, it will not
// try to consume anything itself.
func (p *Parser) ErrorAt(at interface{}, message string, args ...interface{}) {
	loc := p.CST(at)
	p.Errors.Add(&p.Reader, loc, message, args...)
}

// Expected is a wrapper around p.ErrorAt for the very common case of an unexpected
// input. It uses value as the expected input, and parses a token of the stream
// for the unexpected actual input.
func (p *Parser) Expected(value string) {
	invalid := &fragment{}
	invalid.SetToken(p.GuessNextToken())
	p.Errors.Add(&p.Reader, invalid, "Expected \"%s\" got \"%s\"", value, invalid.Token().String())
}
