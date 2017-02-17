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

// Package node holds the syntax tree nodes for a parse minidown document.
package node

// Node represents a node in a minidown document graph.
type Node interface{}

// Block represents a collection of nodes in sequential order.
type Block struct {
	Children []Node
}

// Heading represents a heading node.
type Heading struct {
	Scale int // {1: H1, 2: H2} and so on.
	Body  Node
}

// NewLine represents a blank new line.
type NewLine struct{}

// Whitespace represent a whitespace.
type Whitespace struct{}

// Emphasis represents a emphasis node.
type Emphasis struct {
	Style string // "**", "_" etc
	Body  Node
}

// Text represent a block of text.
type Text struct {
	Text string
}

// Link represents a link in the form: [body](target).
type Link struct {
	Body   Node
	Target Node
}

// Tag represents a tag in the form: {{identifier:type}}, where "type" is
// optional and is "string" by default.
type Tag struct {
	Identifier string
	Type       string
}
