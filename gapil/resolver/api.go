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

package resolver

import (
	"sort"
	"strconv"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func apiNames(rv *resolver, in *ast.API) {
	if in.Index != nil {
		if n, e := strconv.ParseUint(in.Index.Value, 10, 4); e == nil {
			rv.api.Index = semantic.Uint8Value(n)
		} else {
			rv.errorf(in.Index, "cannot convert API index %q into 4-bit unsigned integer", in.Index.Value)
		}
	}
	// Build and register the high level semantic objects
	for _, e := range in.Enums {
		n := &semantic.Enum{AST: e, Named: semantic.Named(e.Name.Value)}
		rv.api.Enums = append(rv.api.Enums, n)
		semantic.Add(rv.api, n)
		rv.addType(n)
	}
	for _, c := range in.Classes {
		n := &semantic.Class{AST: c, Named: semantic.Named(c.Name.Value)}
		rv.api.Classes = append(rv.api.Classes, n)
		semantic.Add(rv.api, n)
		rv.addType(n)
	}
	for _, p := range in.Pseudonyms {
		n := &semantic.Pseudonym{AST: p, Named: semantic.Named(p.Name.Value)}
		rv.api.Pseudonyms = append(rv.api.Pseudonyms, n)
		semantic.Add(rv.api, n)
		rv.addType(n)
	}
	for _, e := range in.Externs {
		n := &semantic.Function{AST: e, Named: semantic.Named(e.Generic.Name.Value), Extern: true}
		rv.api.Externs = append(rv.api.Externs, n)
		semantic.Add(rv.api, n)
	}
	for _, m := range in.Commands {
		f := &semantic.Function{AST: m, Named: semantic.Named(m.Generic.Name.Value)}
		if !m.Parameters[0].This {
			rv.api.Functions = append(rv.api.Functions, f)
			semantic.Add(rv.api, f)
		} else {
			rv.api.Methods = append(rv.api.Methods, f)
		}
	}
	for _, m := range in.Subroutines {
		if m.Parameters[0].This {
			rv.errorf(m.Parameters[0], "cannot use this on subroutines")
			continue
		}
		if len(m.Generic.Arguments) > 0 {
			rv.genericSubroutines[m.Generic.Name.Value] = newGenericSubroutine(rv, m)
		} else {
			f := &semantic.Function{AST: m, Named: semantic.Named(m.Generic.Name.Value), Subroutine: true}
			rv.api.Subroutines = append(rv.api.Subroutines, f)
			semantic.Add(rv.api, f)
		}
	}
	for _, f := range in.Fields {
		n := &semantic.Global{AST: f, Named: semantic.Named(f.Name.Value)}
		rv.api.Globals = append(rv.api.Globals, n)
		semantic.Add(rv.api, n)
	}
	for _, c := range in.Definitions {
		n := &semantic.Definition{AST: c, Named: semantic.Named(c.Name.Value)}
		rv.api.Definitions = append(rv.api.Definitions, n)
		rv.addNamed(n)
	}
}

func resolve(rv *resolver) {
	rv.globals = rv.scope

	rv.addMembers(rv.api)
	for _, e := range rv.api.Enums {
		enum(rv, e)
	}
	for _, p := range rv.api.Pseudonyms {
		pseudonym(rv, p)
	}

	for _, c := range rv.api.Definitions {
		definition(rv, c)
	}
	for _, c := range rv.api.Classes {
		class(rv, c)
	}
	for _, g := range rv.api.Globals {
		global(rv, g)
	}
	for _, e := range rv.api.Externs {
		functionSignature(rv, e)
		functionBody(rv, nil, e)
	}
	for _, s := range rv.api.Subroutines {
		functionSignature(rv, s)
		for _, p := range s.FullParameters {
			if p.Type == semantic.AnyType {
				rv.errorf(p, "cannot use any as parameter type on subroutines")
			}
		}
	}
	for _, s := range rv.api.Subroutines {
		functionBody(rv, nil, s)
	}
	for _, f := range rv.api.Functions {
		functionSignature(rv, f)
		functionBody(rv, nil, f)
	}
	if rv.options.ExtractCalls {
		for _, f := range rv.api.Subroutines {
			extractCalls(rv, f.Block)
		}
		for _, f := range rv.api.Functions {
			extractCalls(rv, f.Block)
		}
	}
	if rv.options.RemoveDeadCode {
		for _, f := range rv.api.Subroutines {
			removeDeadCode(rv, f.Block)
		}
		for _, f := range rv.api.Functions {
			removeDeadCode(rv, f.Block)
		}
	}
	for _, f := range rv.api.Functions {
		resolveFenceOrder(rv, f, visitedFuncs{})
	}
	for _, m := range rv.api.Methods {
		method(rv, m)
	}
	sort.Sort(functionsByName(rv.api.Externs))
	sort.Sort(slicesByName(rv.api.Slices))
	sort.Sort(mapsByName(rv.api.Maps))

	for _, f := range rv.api.Functions {
		if f.Recursive && (f.Order.Pre() && f.Order.Post()) {
			rv.errorf(f, "Fence in recursive function")
		}
	}
	for _, f := range rv.api.Subroutines {
		if f.Recursive && (f.Order.Pre() && f.Order.Post()) {
			rv.errorf(f, "Fence in recursive function")
		}
	}
}

func annotations(rv *resolver, in ast.Annotations) semantic.Annotations {
	if len(in) == 0 {
		return nil
	}
	var out semantic.Annotations
	for _, a := range in {
		entry := &semantic.Annotation{AST: a, Named: semantic.Named(a.Name.Value)}
		for _, arg := range a.Arguments {
			entry.Arguments = append(entry.Arguments, expression(rv, arg))
		}
		out = append(out, entry)
		rv.mappings.Add(a, entry)
	}
	return out
}

func global(rv *resolver, out *semantic.Global) {
	if out.Type != nil {
		return // already resolved.
	}

	// Begin by assigning a void type to the global.
	// This is done to avoid a stack overflow if the global's type resolves to
	// the same global (for example 'x.y x').
	out.Type = semantic.VoidType

	in := out.AST
	out.Annotations = annotations(rv, in.Annotations)
	out.Type = type_(rv, in.Type)
	if isVoid(out.Type) {
		rv.errorf(in, "void typed global variable %s", out.Name())
	}
	if in.Default != nil {
		rv.with(out.Type, func() {
			out.Default = expression(rv, in.Default)
		})
		dt := out.Default.ExpressionType()
		if !assignable(out.Type, dt) {
			rv.errorf(in, "cannot assign %s to %s", typename(dt), typename(out.Type))
		}
	}
	rv.mappings.Add(in, out)
	rv.mappings.Add(in.Name, out)
}

// slicesByName is used to sort the slice list by name for generated code stability
type slicesByName []*semantic.Slice

func (a slicesByName) Len() int           { return len(a) }
func (a slicesByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a slicesByName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }

// mapsByName is used to sort the map list by name for generated code stability
type mapsByName []*semantic.Map

func (a mapsByName) Len() int           { return len(a) }
func (a mapsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a mapsByName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }

// mapsByName is used to sort the map list by name for generated code stability
type functionsByName []*semantic.Function

func (a functionsByName) Len() int           { return len(a) }
func (a functionsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a functionsByName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
