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
	"fmt"
	"reflect"
	"strings"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// Options customize the final output of the resolve.
type Options struct {
	// ExtractCalls moves all call expressions to subroutines out to locals.
	// This is done so that there is an oppotunity to test for abort() before
	// executing the rest of the expression.
	ExtractCalls bool

	// RemoveDeadCode removes if statement blocks when the conditional is a
	// literal false. The primary use of this code is to eliminate conditional
	// blocks inside macros that are directly dependent on a boolean parameter.
	RemoveDeadCode bool
}

type resolver struct {
	errors             parse.ErrorList
	api                *semantic.API
	scope              *scope // The current scope
	globals            *scope // The global scope
	nextID             uint64
	mappings           *semantic.Mappings
	genericSubroutines map[string]genericSubroutine
	aliasStack         stack // Currently resolving aliases.
	defStack           stack // Currently resolving definitions.
	options            Options
}

type scope struct {
	semantic.Symbols
	types     map[string]semantic.Type
	outer     *scope
	inferType semantic.Type
	block     *semantic.Statements
	function  *semantic.Function
	nextID    uint64
}

type named interface {
	Name() string
}

func name(n interface{}) string {
	switch n := n.(type) {
	case semantic.Type:
		return typename(n)
	case named:
		return n.Name()
	case *ast.Identifier:
		return n.Value
	default:
		return fmt.Sprintf("%v %T", n, n)
	}
}

// stack is a stack of objects.
// It is used to detect circular references in type and define declarations.
type stack []interface{}

func (s stack) String() string {
	path := make([]string, len(s))
	for i, o := range s {
		path[i] = name(o)
	}
	return strings.Join(path, " -> ")
}
func (s *stack) push(o interface{}) {
	*s = append(*s, o)
}
func (s *stack) pop() {
	*s = (*s)[:len(*s)-1]
}
func (s stack) contains(o interface{}) bool {
	for _, t := range s {
		if t == o {
			return true
		}
	}
	return false
}

func (rv *resolver) errorf(at interface{}, message string, args ...interface{}) {
	if at != nil {
		n, ok := at.(ast.Node)
		if !ok {
			v := reflect.ValueOf(at)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.Kind() == reflect.Struct {
				if a := v.FieldByName("AST"); a.IsValid() {
					n, ok = a.Interface().(ast.Node)
				}
			}
		}
		if ok && n != nil && !reflect.ValueOf(n).IsNil() {
			rv.errors.Add(nil, rv.mappings.AST.CST(n), message, args...)
			return
		}
	}
	rv.errors.Add(nil, nil, message, args...)
}

func (rv *resolver) icef(at interface{}, message string, args ...interface{}) {
	rv.errorf(at, "INTERNAL ERROR: "+message, args...)
}

// with evaluates the action within a new nested scope.
// The scope is available as rv.scope, and the original scope is
// restored before this function returns.
// Type t is used for type inference within the new scope (for
// instance when resolving untyped numeric constants)
func (rv *resolver) with(t semantic.Type, action func()) {
	original := rv.scope
	rv.scope = &scope{
		types:     map[string]semantic.Type{},
		outer:     rv.scope,
		block:     rv.scope.block,
		function:  rv.scope.function,
		inferType: t,
	}
	defer func() { rv.scope = original }()
	action()
}

// addNamed binds a named node within the current and nested scopes.
func (rv *resolver) addNamed(value semantic.NamedNode) {
	rv.scope.AddNamed(value)
}

// alias maps a name to a node without the node knowing the name.
func (rv *resolver) add(name string, value semantic.Node) {
	rv.scope.Add(name, value)
}

// addMembers adds all the members of owner directly to the curent scope.
func (rv *resolver) addMembers(owner semantic.Owner) {
	owner.VisitMembers(func(m semantic.Owned) {
		rv.scope.AddNamed(m)
	})
}

// addSymbols adds all the entries of symbols directly to the curent scope.
func (rv *resolver) addSymbols(symbols *semantic.Symbols) {
	symbols.Visit(func(name string, node semantic.Node) {
		rv.scope.Add(name, node)
	})
}

func (rv *resolver) ensureResolved(n semantic.Node) {
	if g, ok := n.(*semantic.Global); ok {
		// Globals can refer to other globals in their default initializer.
		// This means some globals may need to be resolved out of order.
		// Ensure that this global is resolved before returning.
		global(rv, g)
	}
}

// find searches the scope stack for a bindings that matches the name.
func (rv *resolver) find(name string) []interface{} {
	result := []interface{}{}
	for search := rv.scope; search != nil; search = search.outer {
		list := search.FindAll(name)
		for _, n := range list {
			rv.ensureResolved(n)
			result = append(result, n)
		}
	}
	if gs, ok := rv.genericSubroutines[name]; ok {
		result = append(result, gs)
	}
	return result
}

// disambiguate takes a list of possible scope values and attempts to see if
// one of them is unambiguously the right choice in the current context.
// For instance, if the values are all enum entries but only one of them
// matches the current enum inference.
func (rv *resolver) disambiguate(matches []interface{}) []interface{} {
	if len(matches) <= 1 {
		return matches
	}

	var enum semantic.Owner
	for test := rv.scope; test != nil; test = test.outer {
		if test.inferType != nil {
			if e, ok := test.inferType.(*semantic.Enum); ok {
				enum = e
				break
			}
		}
	}
	if enum == nil {
		// No disambiguating enums present
		return matches
	}
	var res *semantic.EnumEntry
	for _, m := range matches {
		if ev, ok := m.(*semantic.EnumEntry); ok {
			if enum == ev.Owner() {
				// We found a disambiguation match
				res = ev
			}
		} else {
			// Non enum match found
			return matches
		}
	}
	if res == nil {
		return matches
	}
	// Matched exactly once
	return []interface{}{res}
}

// get searches the scope stack for a bindings that matches the name.
// If it cannot find exactly 1 unambiguous match, it reports an error, and
// nil is returned.
func (rv *resolver) get(at ast.Node, name string) interface{} {
	if name == "_" {
		return &semantic.Ignore{AST: at}
	}
	if strings.HasPrefix(name, "$") {
		for _, g := range semantic.BuiltinGlobals {
			if name == g.Name() {
				return g
			}
		}
		rv.errorf(at, "Unknown builtin global %s", name)
	}
	matches := rv.disambiguate(rv.find(name))
	switch len(matches) {
	case 0:
		rv.errorf(at, "Unknown identifier %s", name)
		return nil
	case 1:
		return matches[0]
	default:
		rv.ambiguousIdentifier(at, matches)
		return nil
	}
}

func (rv *resolver) ambiguousIdentifier(at ast.Node, matches []interface{}) {
	possibilities := ""
	for i, m := range matches {
		if i > 0 {
			possibilities += ", "
		}
		switch t := m.(type) {
		case *semantic.EnumEntry:
			possibilities += fmt.Sprintf("%s.%s", t.Owner().Name(), t.Name())
		case *semantic.Parameter:
			possibilities += fmt.Sprintf("parameter %q", t.Name())
		case *semantic.Global:
			possibilities += fmt.Sprintf("global %q", t.Name())
		case semantic.Type:
			possibilities += fmt.Sprintf("type %q [%T]", typename(t), t)
		default:
			possibilities += fmt.Sprintf("[%T]%v", t, t)
		}
	}
	rv.errorf(at, "Ambiguous identifier %q [using %s].\n Could be: %s", name(at), typename(rv.scope.inferType), possibilities)
}

func (rv *resolver) addType(t semantic.Type) {
	name := t.Name()

	withLocation := func(ty semantic.Type) string {
		astBacked, ok := ty.(semantic.ASTBacked)
		if ok {
			tok := rv.mappings.AST.CST(astBacked.ASTNode()).Tok()
			line, col := tok.Cursor()
			return fmt.Sprintf("%s at %s:%d:%d", ty.Name(), tok.Source.Filename, line, col)
		}
		return ty.Name()
	}

	if prev, present := rv.scope.types[name]; present {
		rv.errorf(t, "Duplicate type %s (already seen: %s)", t.Name(), withLocation(prev))
	}
	rv.scope.types[name] = t
}

func (rv *resolver) addGenericParameter(name string, t semantic.Type) {
	if _, present := rv.scope.types[name]; present {
		rv.errorf(t, "Duplicate type %s", name)
	}
	rv.scope.types[name] = t
}

func (rv *resolver) findType(at ast.Node, name string) semantic.Type {
	for search := rv.scope; search != nil; search = search.outer {
		if t, found := search.types[name]; found {
			return t
		}
	}
	return nil
}

func (rv *resolver) addStatement(s semantic.Statement) {
	if !isInvalid(s) {
		*rv.scope.block = append(*rv.scope.block, s)
	}
}

func (rv *resolver) uid() uint64 {
	id := rv.nextID
	rv.nextID++
	return id
}

func (rv *resolver) declareTemporaryLocal(value semantic.Expression) *semantic.DeclareLocal {
	name := fmt.Sprintf("_res_%v", rv.scope.uid())
	decl := &semantic.DeclareLocal{}
	decl.Local = &semantic.Local{
		Declaration: decl,
		Type:        value.ExpressionType(),
		Named:       semantic.Named(name),
		Value:       value,
	}
	return decl
}

func (s *scope) uid() uint64 {
	id := s.nextID
	s.nextID++
	return id
}
