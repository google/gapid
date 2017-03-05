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

package annotate

import (
	"fmt"

	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/snippets"
)

// path returns a Pathway for the given stmt. If hasPath(stmt) is
// false then it will panic.
func path(stmt semantic.Node) snippets.Pathway {
	switch expr := stmt.(type) {
	case *semantic.Global:
		return snippets.Variable(snippets.SymbolCategory_Global, expr.Name())
	case *semantic.Parameter:
		return snippets.Variable(snippets.SymbolCategory_Parameter, expr.Name())
	case *semantic.Local:
		return snippets.Variable(snippets.SymbolCategory_Local, expr.Name())
	case *semantic.Member:
		return snippets.Field(path(expr.Object), expr.Field.Name())
	case *semantic.PointerRange:
		return snippets.Range(path(expr.Pointer))
	case *semantic.SliceRange:
		// Don't change the path, since range of slice is still a slice.
		return path(expr.Slice)
	case *semantic.ArrayIndex:
		return snippets.Elem(path(expr.Array))
	case *semantic.SliceIndex:
		return snippets.Elem(path(expr.Slice))
	case *semantic.MapIndex:
		return snippets.Elem(path(expr.Map))
	case *semantic.Observed:
		return path(expr.Parameter)
	case *semantic.Cast:
		return path(expr.Object)
	default:
		panic(fmt.Errorf("Unexpect path expression %T:%v", stmt, stmt))
	}
}

// hasPath returns true if stmt is will return a valid path if given to
// path(stmt). Any expression which represents a variable or a sub-component
// of a variable should have a path.
func hasPath(stmt semantic.Node, api *semantic.API) bool {
	switch expr := stmt.(type) {
	case *semantic.Parameter, *semantic.Local:
		return true
	case *semantic.Global:
		return expr.Owner() == api
	case *semantic.Member:
		return hasPath(expr.Object, api)
	case *semantic.PointerRange:
		return hasPath(expr.Pointer, api)
	case *semantic.SliceRange:
		return hasPath(expr.Slice, api)
	case *semantic.ArrayIndex:
		return hasPath(expr.Array, api)
	case *semantic.SliceIndex:
		return hasPath(expr.Slice, api)
	case *semantic.MapIndex:
		return hasPath(expr.Map, api)
	case *semantic.Observed:
		return hasPath(expr.Parameter, api)
	case *semantic.Cast:
		return hasPath(expr.Object, api)
	}
	return false
}
