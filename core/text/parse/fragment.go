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
	"io"

	"github.com/google/gapid/core/data/compare"
)

// Fragment is a component of a cst that is backed by a token.
// This includes Nodes and all forms of space and comment.
type Fragment interface {
	// Token returns the underlying token of this node.
	Token() Token
	// Write is used to write the underlying token out to the writer.
	WriteTo(io.Writer) error
}

// Separator is a list type to manage fragments that were skipped.
type Separator []Fragment

type fragment struct {
	token Token
}

func (s Separator) WriteTo(w io.Writer) error {
	for _, n := range s {
		if err := n.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}

func (n *fragment) Token() Token {
	return n.token
}

func (n *fragment) SetToken(token Token) {
	n.token = token
}

func (n *fragment) WriteTo(w io.Writer) error {
	_, err := io.WriteString(w, n.Token().String())
	return err
}

func NewFragment(token Token) Fragment {
	return &fragment{token}
}

func compareFragments(c compare.Comparator, reference, value fragment) {
	c.With(c.Path.Member("token", reference, value)).Compare(reference.token, value.token)
}

func init() {
	compare.Register(compareFragments)
}
