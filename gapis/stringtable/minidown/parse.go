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

// Package minidown is a minimal-feature subset of the markdown language.
//
// Minidown supports headers, emphasis and newlines.
package minidown

import (
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/stringtable/minidown/node"
	"github.com/google/gapid/gapis/stringtable/minidown/parser"
)

// Parse parses a minidown file.
func Parse(filename, source string) (node.Node, parse.ErrorList) {
	return parser.Parse(filename, source)
}
