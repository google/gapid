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

package validate

import (
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/semantic"
)

type fieldUsage struct{ read, written bool }

const annoUnused = "unused"

// noUnused verifies that all declared types and fields are used.
func noUnused(api *semantic.API, mappings *semantic.Mappings) Issues {
	types := map[semantic.Type]bool{}
	fields := map[*semantic.Field]fieldUsage{}
	tokens := map[semantic.Node]cst.Token{}

	// Gather all declared types
	for _, t := range api.Classes {
		types[t] = false
		tokens[t] = mappings.AST.CST(t.AST).Tok()
		for _, f := range t.Fields {
			fields[f] = fieldUsage{}
			tokens[f] = mappings.AST.CST(f.AST).Tok()
			if _, ok := f.Type.(*semantic.Class); ok {
				fields[f] = fieldUsage{false, true}
			}
		}
	}
	for _, t := range api.Enums {
		types[t] = false
		tokens[t] = mappings.AST.CST(t.AST).Tok()
	}
	for _, t := range api.Pseudonyms {
		types[t] = false
		tokens[t] = mappings.AST.CST(t.AST).Tok()
	}

	var markClassFieldsUsed func(ty semantic.Type, read, written bool)

	markFieldUsed := func(f *semantic.Field, read, written bool) {
		if usage, ok := fields[f]; ok {
			usage.written = usage.written || written
			usage.read = usage.read || read
			fields[f] = usage
		}
	}

	markClassFieldsUsed = func(ty semantic.Type, read, written bool) {
		switch ty := ty.(type) {
		case *semantic.Class:
			for _, f := range ty.Fields {
				markFieldUsed(f, read, written)
				markClassFieldsUsed(f.Type, read, written)
			}
		case *semantic.StaticArray:
			markClassFieldsUsed(ty.ValueType, read, written)
		case *semantic.Reference:
			markClassFieldsUsed(ty.To, read, written)
		case *semantic.Pointer:
			markClassFieldsUsed(ty.To, read, written)
		case *semantic.Pseudonym:
			markClassFieldsUsed(ty.To, read, written)
		}
	}

	// Functions for marking types as used
	var markTypeUsed func(t semantic.Type)
	markTypeUsed = func(t semantic.Type) {
		if types[t] {
			return
		}
		types[t] = true
		switch t := t.(type) {
		case *semantic.Reference:
			markTypeUsed(t.To)
		case *semantic.Slice:
			markTypeUsed(t.To)
		case *semantic.StaticArray:
			markTypeUsed(t.ValueType)
		case *semantic.Map:
			markTypeUsed(t.ValueType)
			markTypeUsed(t.KeyType)
		case *semantic.Pointer:
			markTypeUsed(t.To)
		case *semantic.Pseudonym:
			markTypeUsed(t.To)
		case *semantic.Class:
			for _, f := range t.Fields {
				markTypeUsed(f.Type)
			}
		}
	}

	var traverse func(n semantic.Node)
	traverse = func(n semantic.Node) {
		if e, ok := n.(semantic.Expression); ok {
			markTypeUsed(e.ExpressionType())
		}

		switch n := n.(type) {
		case *semantic.Assign:
			if m, ok := n.LHS.(*semantic.Member); ok {
				markFieldUsed(m.Field, false, true)
				traverse(m.Object)
				traverse(n.RHS)
				return
			}
		case *semantic.ArrayAssign:
			if m, ok := n.To.Array.(*semantic.Member); ok {
				markFieldUsed(m.Field, false, true)
				traverse(m.Object)
				traverse(n.Value)
				return
			}
		case *semantic.MapAssign:
			if m, ok := n.To.Map.(*semantic.Member); ok {
				markFieldUsed(m.Field, false, true)
				traverse(m.Object)
				traverse(n.Value)
				return
			}
		case *semantic.Member:
			markFieldUsed(n.Field, true, false)
		case *semantic.SliceIndex:
			markClassFieldsUsed(n.Type.To, true, false)
		case *semantic.Field:
			markFieldUsed(n, true, false)
		case *semantic.FieldInitializer:
			markFieldUsed(n.Field, false, true)
		case *semantic.Parameter:
			if !n.Function.Subroutine && !n.Function.Extern {
				markClassFieldsUsed(n.Type, true, true)
			}
		case semantic.Type:
			markTypeUsed(n)
			return // Don't traverse into the type.
		case *semantic.Callable:
			return // Don't traverse into these.
		}
		semantic.Visit(n, traverse)
	}

	// Traverse the API finding all used types
	for _, g := range api.Globals {
		markTypeUsed(g.Type)
	}
	for _, f := range api.Subroutines {
		traverse(f)
	}
	for _, f := range api.Functions {
		traverse(f)
	}
	for _, c := range api.Classes {
		for _, m := range c.Methods {
			traverse(m)
		}
	}
	for _, c := range api.Pseudonyms {
		for _, m := range c.Methods {
			traverse(m)
		}
	}
	for _, e := range api.Externs {
		if e.Return != nil {
			markClassFieldsUsed(e.Return.Type, false, true)
		}
	}

	// Report all types declared but not used as issues
	issues := Issues{}
	for t, used := range types {
		if a, ok := t.(semantic.Annotated); ok {
			if anno := a.GetAnnotation(annoUnused); anno != nil {
				if used {
					issues.addf(mappings.AST.CST(anno.AST), "Redundant annotation")
				}
				continue
			}
		}
		if !used {
			issues.addf(mappings.CST(t), "Type %s declared but never used", t.Name())
		}
	}
	for f, usage := range fields {
		var msg string
		switch {
		case !usage.read && usage.written:
			msg = "Field %s.%s assigned but never read"
		case usage.read && !usage.written:
			msg = "Field %s.%s read but never assigned"
		case !usage.read && !usage.written:
			msg = "Field %s.%s never used"
		}
		class := f.Owner().(*semantic.Class)
		unused := len(msg) > 0
		fiu, ciu := f.GetAnnotation(annoUnused), class.GetAnnotation(annoUnused)
		if unused && fiu == nil && ciu == nil {
			issues.addf(mappings.AST.CST(f.AST), msg, f.Owner().Name(), f.Name())
		}
		if !unused && fiu != nil && ciu == nil {
			issues.addf(mappings.AST.CST(fiu.AST), "Redundant annotation")
		}
	}
	return issues
}
