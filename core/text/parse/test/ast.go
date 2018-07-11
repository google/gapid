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

package test

import (
	"strconv"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
)

type ListNode []interface{}

type ArrayNode ListNode

type CallNode struct {
	name *ValueNode
	args ListNode
}

type NumberNode uint64
type ValueNode string

const (
	tagComma      = ","
	tagBeginArray = "["
	tagEndArray   = "]"
	tagBeginCall  = "("
	tagEndCall    = ")"
)

var (
	comma      = opParser(tagComma)
	beginArray = opParser(tagBeginArray)
	endArray   = opParser(tagEndArray)
	beginCall  = opParser(tagBeginCall)
	endCall    = opParser(tagEndCall)
)

func opParser(op string) parse.LeafParser {
	return func(p *parse.Parser, _ *cst.Leaf) {
		if !p.String(op) {
			p.Expected(op)
		}
	}
}

func maybeValue(p *parse.Parser, in *cst.Branch) interface{} {
	switch {
	case Peek(&p.Reader, tagBeginArray):
		a := &ArrayNode{}
		p.ParseBranch(in, a.Parse)
		return a
	case p.Numeric() != parse.NotNumeric:
		var n NumberNode
		p.ParseLeaf(in, n.Consume)
		return &n
	case p.AlphaNumeric():
		var n ValueNode
		p.ParseLeaf(in, n.Consume)
		if Peek(&p.Reader, tagBeginCall) {
			c := &CallNode{name: &n}
			p.ParseBranch(in, c.Parse)
			return c
		} else {
			return &n
		}
	}
	return nil
}

func parseValue(p *parse.Parser, in *cst.Branch) interface{} {
	v := maybeValue(p, in)
	if v == nil {
		p.Expected("value")
	}
	return v
}

func (n *ValueNode) Parse(p *parse.Parser, l *cst.Leaf) {
	if !p.AlphaNumeric() {
		p.Expected("value")
	}
	n.Consume(p, l)
}

func (n *ValueNode) Consume(p *parse.Parser, l *cst.Leaf) {
	l.Token = p.Consume()
	*n = ValueNode(l.Token.String())
}

func (n *NumberNode) Consume(p *parse.Parser, l *cst.Leaf) {
	l.Token = p.Consume()
	v, _ := strconv.ParseUint(l.Token.String(), 0, 32)
	*n = NumberNode(v)
}

func (n *ListNode) Parse(p *parse.Parser, cst *cst.Branch) {
	v := maybeValue(p, cst)
	if v == nil {
		return
	}
	*n = append(*n, v)
	for p.String(tagComma) {
		p.ParseLeaf(cst, nil)
		*n = append(*n, parseValue(p, cst))
	}
}

func (n *ArrayNode) Parse(p *parse.Parser, cst *cst.Branch) {
	p.ParseLeaf(cst, beginArray)
	p.ParseBranch(cst, (*ListNode)(n).Parse)
	p.ParseLeaf(cst, endArray)
}

func (n *CallNode) Parse(p *parse.Parser, cst *cst.Branch) {
	p.ParseLeaf(cst, beginCall)
	p.ParseBranch(cst, n.args.Parse)
	p.ParseLeaf(cst, endCall)
}

func Value(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		n := ValueNode(t)
		v = &n
	case int:
		n := NumberNode(t)
		v = &n
	}
	return v
}

func List(values ...interface{}) *ListNode {
	n := &ListNode{}
	for _, v := range values {
		*n = append(*n, Value(v))
	}
	return n
}

func Array(values ...interface{}) *ArrayNode {
	n := &ArrayNode{}
	for _, v := range values {
		*n = append(*n, Value(v))
	}
	return n
}

func Call(name string, values ...interface{}) *CallNode {
	n := &CallNode{}
	nv := ValueNode(name)
	n.name = &nv
	for _, v := range values {
		n.args = append(n.args, Value(v))
	}
	return n
}
