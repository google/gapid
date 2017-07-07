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

package preprocessor

import "github.com/google/gapid/gapis/api/gles/glsl/ast"

// Token hide sets. Used for preventing infinite recursion during macro expansion, as per the spec.
type hideSet map[string]struct{}

func (h hideSet) Clone() hideSet {
	ret := make(hideSet, len(h)+1)
	for k, v := range h {
		ret[k] = v
	}
	return ret
}

func (a hideSet) AddAll(b hideSet) {
	for k := range b {
		a[k] = struct{}{}
	}
}

func intersect(a hideSet, b hideSet) hideSet {
	ret := make(hideSet)
	for k := range a {
		if _, present := b[k]; present {
			ret[k] = struct{}{}
		}
	}
	return ret
}

// tokenExpansion represents an token produced as a result of macro expansion.
// It stores the tokens hide set. Used internally in the preprocessor.
type tokenExpansion struct {
	Info    TokenInfo // The token.
	HideSet hideSet   // Which tokens should not be expanded when this token is processed.
}

func newTokenExpansion(info TokenInfo) tokenExpansion {
	return tokenExpansion{Info: info, HideSet: make(hideSet, 0)}
}

// The function used for expanding tokens in macro definitions.  The args argument is a list of
// macro arguments.
type macroExpander func(args [][]tokenExpansion) []tokenExpansion

// argumentExpander expands to the n-th macro argument.
type argumentExpander int

func (e argumentExpander) expand(args [][]tokenExpansion) []tokenExpansion {
	ret := make([]tokenExpansion, len(args[e]))
	for i, arg := range args[e] {
		ret[i].Info = arg.Info
		ret[i].HideSet = arg.HideSet.Clone()
	}
	return ret
}

// TokenInfo expands to itself.
func (e TokenInfo) expand([][]tokenExpansion) []tokenExpansion {
	return []tokenExpansion{{e, make(hideSet, 1)}}
}

// __LINE__ expands to the current line number
func (p *preprocessorImpl) expandLine(args [][]tokenExpansion) []tokenExpansion {
	ti := TokenInfo{Token: ast.IntValue(p.line)}
	return ti.expand(args)
}
