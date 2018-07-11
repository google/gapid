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

package cst

import (
	"io"
)

// Fragment is a component of a cst that is backed by a token.
// This includes Nodes and all forms of space and comment.
type Fragment interface {
	// Tok returns the underlying token of this fragment.
	Tok() Token
	// Write is used to write the underlying token out to the writer.
	Write(io.Writer) error
}

// Separator is a list type to manage fragments that were skipped.
type Separator []Fragment

// Write is used to write the list of separators to the writer.
func (s Separator) Write(w io.Writer) error {
	for _, n := range s {
		if err := n.Write(w); err != nil {
			return err
		}
	}
	return nil
}
