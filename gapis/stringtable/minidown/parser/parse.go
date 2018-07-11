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

// Package parser implements the minidown parser.
package parser

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/stringtable/minidown/node"
	"github.com/google/gapid/gapis/stringtable/minidown/scanner"
	"github.com/google/gapid/gapis/stringtable/minidown/token"
)

// Parse parses a minidown file.
func Parse(filename, source string) (node.Node, parse.ErrorList) {
	tokens, errs := scanner.Scan(filename, source)
	if len(errs) > 0 {
		return nil, errs
	}
	p := parser{
		tokens: tokens,
		errors: errs,
	}
	root := compact(p.parse())
	return root, p.errors
}

type parser struct {
	tokens []token.Token
	prev   token.Token
	curr   token.Token
	errors parse.ErrorList
	stack  stack
}

// next gets the next token from the token list, updating p.prev and p.curr.
// If there are no more tokens then next returns false.
func (p *parser) next() bool {
	if len(p.tokens) == 0 {
		return false
	}
	p.prev, p.curr = p.curr, p.tokens[0]
	p.tokens = p.tokens[1:]
	return true
}

// peek returns the next token without changing the current token.
func (p *parser) peek() node.Node {
	if len(p.tokens) == 0 {
		return nil
	}
	return p.tokens[0]
}

// add inserts n into the node at the top of the stack.
func (p *parser) add(n node.Node) {
	addTo(p.stack.head(), n)
}

// stack stores the node hierarchy while building the graph.
type stack []node.Node

// head returns the most recently pushed node on the stack.
func (s *stack) head() node.Node {
	return (*s)[len(*s)-1]
}

// peek returns the nth node from the head of the stack.
func (s *stack) peek(n int) node.Node {
	if n >= len(*s) {
		return nil
	}
	return (*s)[len(*s)-n-1]
}

// push pushes n to the top of the stack.
func (s *stack) push(n node.Node) {
	*s = append(*s, n)
}

// pop pops and returns the node at the top of the stack.
func (s *stack) pop() node.Node {
	n := s.head()
	*s = (*s)[:len(*s)-1]
	return n
}

// parse parses the token list into a node tree.
func (p *parser) parse() *node.Block {
	root := &node.Block{}
	p.stack.push(root)
	for p.next() {
		if len(p.curr.CST().Prefix()) > 0 {
			p.add(&node.Whitespace{})
		}
		switch t := p.curr.(type) {
		case token.Text:
			p.add(&node.Text{Text: t.String()})

		case token.Heading:
			if _, newline := p.prev.(token.NewLine); newline || p.prev == nil {
				h := &node.Heading{Scale: len(t.CST().Tok().String())}
				p.add(h)
				p.stack.push(h)
				continue // Don't add whitespace suffix.
			} else {
				// Treat as text
				p.add(&node.Text{Text: t.CST().Tok().String()})
			}

		case token.Emphasis:
			style := t.CST().Tok().String()
			if emphasis, ok := p.stack.head().(*node.Emphasis); ok && emphasis.Style == style {
				// closing emphasis
				p.stack.pop()
				p.add(emphasis)
			} else {
				// opening emphasis
				p.stack.push(&node.Emphasis{Style: style})
			}

		case token.Bullet: // TODO: Lists
			p.add(&node.Text{Text: "*"})

		case token.Tag:
			str := t.CST().Tok().String()
			str = str[2 : len(str)-2] // trim {{ and }}
			if t.Typed {
				// Split name:type around ':'
				splitted := strings.SplitN(str, ":", 2)
				tagName, tagType := splitted[0], splitted[1]
				p.add(&node.Tag{Identifier: tagName, Type: tagType})
			} else {
				// Token contains only name
				p.add(&node.Tag{Identifier: str, Type: "string"})
			}

		case token.OpenBracket:
			switch {
			case t.Is('['):
				// Opening link body
				l := &node.Link{Body: &node.Block{}}
				p.stack.push(l)
				p.stack.push(l.Body)

			default:
				p.add(&node.Text{Text: t.CST().Tok().String()})
			}

		case token.CloseBracket:
			switch {
			case t.Is(']'):
				if l, ok := p.stack.peek(1).(*node.Link); ok {
					// closing link body
					p.stack.pop() // body
					p.stack.pop() // link
					p.add(l)

					if bracket, ok := p.peek().(token.OpenBracket); ok && bracket.Is('(') {
						l.Target = &node.Block{}
						p.stack.push(l)
						p.stack.push(l.Target)
						p.next() // consume '('
					}
				}

			case t.Is(')'):
				if _, ok := p.stack.peek(1).(*node.Link); ok {
					// closing link target
					p.stack.pop() // body
					p.stack.pop() // target
				} else {
					p.add(&node.Text{Text: t.CST().Tok().String()})
				}

			default:
				p.add(&node.Text{Text: t.CST().Tok().String()})
			}

		case token.NewLine:
			p.handleNewLine()
			if _, newline := p.peek().(token.NewLine); newline {
				p.add(&node.NewLine{})
				for newline {
					p.next()
					_, newline = p.peek().(token.NewLine)
				}
			} else {
				p.add(&node.Whitespace{})
			}
		}
		if len(p.curr.CST().Suffix()) > 0 {
			p.add(&node.Whitespace{})
		}
	}
	p.handleNewLine()
	return root
}

// handleNewLine adjusts nodes on the stack for a newline.
func (p *parser) handleNewLine() {
	for {
		switch parent := p.stack.head().(type) {
		case *node.Emphasis:
			// Unpaired emphasis. Treat as text.
			p.stack.pop()
			p.add(&node.Text{Text: parent.Style})
			p.add(parent.Body)

		case *node.Block:
			if link, ok := p.stack.peek(1).(*node.Link); ok {
				switch p.stack.head() {
				case link.Body:
					// Unclosed link body. Treat as text.
					p.stack.pop() // body
					p.stack.pop() // link
					p.add(&node.Text{Text: "["})
					p.add(link.Body)

				case link.Target:
					// Unclosed link target. Treat as text.
					p.stack.pop() // body
					p.stack.pop() // target
					p.add(&node.Text{Text: "("})
					p.add(link.Target)
					link.Target = nil

				default:
					panic("Unexpected nested node of a link")
				}
			} else {
				// Remove any whitespace at end of line.
				if c := len(parent.Children); c > 0 {
					if _, ws := parent.Children[c-1].(*node.Whitespace); ws {
						parent.Children = parent.Children[:c-1]
					}
				}
				return
			}

		case *node.Heading:
			p.stack.pop()

		default:
			return
		}
	}
}

// addTo adds n to the end of the list of children for parent.
func addTo(parent, n node.Node) {
	switch parent := parent.(type) {
	case *node.Block:
		parent.Children = append(parent.Children, n)

	case *node.Emphasis:
		if parent.Body == nil {
			parent.Body = &node.Block{}
		}
		addTo(parent.Body, n)

	case *node.Heading:
		if parent.Body == nil {
			parent.Body = &node.Block{}
		}
		addTo(parent.Body, n)

	default:
		panic(fmt.Errorf("Node %T cannot have children", parent))
	}
}

// compact traverses the graph of nodes starting from n, removing
// redundant nodes and merging together adjacent text nodes.
func compact(n node.Node) node.Node {
	switch n := n.(type) {
	case *node.Block:
		children := []node.Node{}

		var add func(node.Node)
		add = func(n node.Node) {
			switch n := compact(n).(type) {
			case *node.Block:
				for _, c := range n.Children {
					add(c)
				}
				return
			case *node.Whitespace:
				if len(children) > 0 {
					switch children[len(children)-1].(type) {
					case *node.NewLine, *node.Whitespace, *node.Heading:
						// <newline> <whitespace> -> <newline>
						// <whitespace> <whitespace> -> <whitespace>
						// <heading> <whitespace> -> <heading>
						return
					}
				}
			case *node.Text:
				if len(children) > 0 {
					switch prev := children[len(children)-1].(type) {
					case *node.Text:
						// <text a> <text b> -> <text a+text b>
						prev.Text = prev.Text + n.Text
						return
					case *node.Whitespace:
						// <text a> <ws> <text b> -> <text a+' '+text b>
						if len(children) > 1 {
							if a, ok := children[len(children)-2].(*node.Text); ok {
								a.Text = a.Text + " " + n.Text
								children = children[:len(children)-1] // kill ws.
								return
							}
						}
					}
				}
			case nil:
				return
			}
			children = append(children, n)
		}

		for _, c := range n.Children {
			add(c)
		}

		switch len(children) {
		case 0:
			return nil
		case 1:
			return children[0]
		default:
			n.Children = children
		}
	case *node.Emphasis:
		n.Body = compact(n.Body)
	case *node.Heading:
		n.Body = compact(n.Body)
	case *node.Link:
		n.Body = compact(n.Body)
		n.Target = compact(n.Target)
	case *node.Text, *node.NewLine, *node.Whitespace, *node.Tag, nil:
	default:
		panic(fmt.Errorf("Unknown node type %T", n))
	}
	return n
}
