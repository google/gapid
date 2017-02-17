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

	"github.com/google/gapid/core/gapil/semantic"
	"github.com/google/gapid/core/gapil/snippets"
)

// nodeString returns a textual representation of a semantic node. If the
// node has a name that is used, otherwise its path is used, otherwise it
// is just formatted with %T:%v.
func nodeString(n semantic.Node, api *semantic.API) string {
	if n, ok := n.(semantic.NamedNode); ok {
		return n.Name()
	}
	if hasPath(n, api) {
		return fmt.Sprint(path(n))
	}
	return fmt.Sprintf("%T:%v", n, n)
}

// flow make two locations equivalent.
func flow(left *Location, right *Location) {
	if left == nil || right == nil {
		return
	}

	left.alias(right.leader())
}

// Annotator is used to perform the static analysis.
type Annotator struct {
	symbols SymbolSpace
	api     *semantic.API
}

func (a Annotator) String() string {
	return fmt.Sprintf("annotator(%s)", a.symbols)
}

// New returns a newly initialized Annotator
func New(api *semantic.API) *Annotator {
	return &Annotator{symbols: MakeSymbolSpace(), api: api}
}

// flow makes information flow between an expression right and an location
// leader left. The specified reason is ignored at the moment.
func (a *Annotator) flow(left *Location, right semantic.Expression) {
	flow(left, a.visitExpression(right))
}

// accept used to indicate the expression on the left hand side accepts
// values from the expression on the right hand side. The left hand side
// expression must be assignable (an l-value). The underscore '_' expression
// is ignored on the left hand side.
func (a *Annotator) accept(left semantic.Expression, right semantic.Expression) {
	if _, ignore := left.(*semantic.Ignore); ignore {
		return
	}
	leftHasPath := hasPath(left, a.api)
	if leftHasPath {
		a.duplex(left, right)
	}
}

// isAssign tests to see if node is an assignment
func isAssign(node semantic.Node) bool {
	switch node.(type) {
	case *semantic.ArrayAssign, *semantic.SliceAssign,
		*semantic.Assign, *semantic.MapAssign:
		return true
	default:
		return false
	}
}

// assignSrcDest returns the left and right side of an assignment node.
// If isAssign is false this function may panic.
func assignDestSrc(node semantic.Node) (semantic.Expression, semantic.Expression) {
	switch node := node.(type) {
	case *semantic.ArrayAssign:
		return node.To, node.Value
	case *semantic.SliceAssign:
		return node.To, node.Value
	case *semantic.Assign:
		return node.LHS, node.RHS
	case *semantic.MapAssign:
		return node.To, node.Value
	case *semantic.Copy:
		return node.Src, node.Dst
	}
	panic(fmt.Errorf("Expected assign %T:%v", node, node))
}

// isPrimitiveValue returns true if expr is a simple expressions which just
// represents a value of a particular type.
func isPrimitiveValue(expr semantic.Expression) bool {
	switch expr.(type) {
	case semantic.Null,
		semantic.Int8Value, semantic.Uint8Value,
		semantic.Int16Value, semantic.Uint16Value,
		semantic.Int32Value, semantic.Uint32Value,
		semantic.Int64Value, semantic.Uint64Value,
		semantic.BoolValue,
		semantic.StringValue,
		semantic.Float64Value, semantic.Float32Value,
		*semantic.EnumEntry,
		*semantic.Label:
		return true
	default:
		return false
	}
}

// elem decomposes an expression one level. Returns the location which
// represents an element of the collection.
func (a *Annotator) elem(expr semantic.Expression) *Location {
	nested := a.visitExpression(expr).getNested()
	if nested == nil {
		return &Location{}
	}
	if container, ok := nested.(*container); ok {
		return container.Elem()
	}
	return &Location{}
}

// key decomposes an expression one level. Returns the location which
// represents the key of the collection.
func (a *Annotator) key(expr semantic.Expression) *Location {
	nested := a.visitExpression(expr).getNested()
	if nested == nil {
		return &Location{}
	}
	if container, ok := nested.(*container); ok {
		return container.Key()
	}
	return &Location{}
}

// symbolTable returns a symbol table for expression of the category specified
func (a *Annotator) symbolTable(expr semantic.Expression) *ScopedSymbolTable {
	return a.symbols[category(expr)]
}

// declare makes a new symbol table entry for the specified expression.
// The specified expression must be a NamedNode and it must not have
// previously been declared. The location for the new declaration is returned.
func (a *Annotator) declare(expr semantic.Expression) *Location {
	if named, ok := expr.(semantic.NamedNode); ok {
		return a.symbolTable(expr).declare(named.Name())
	}
	panic(fmt.Errorf("Attempt to declare non-named type: %T:%v:%s",
		expr, expr, nodeString(expr, a.api)))
}

// locate returns a location leader for the expression specified. It
// will panic if the expression does not have a location.
func (a *Annotator) locate(expr semantic.Expression) *Location {
	if !hasPath(expr, a.api) {
		panic(fmt.Errorf("Unexpected lvalue %T:%v", expr, expr))
	}
	switch t := expr.(type) {
	case nil:
		panic("nil expression in locate")
	case semantic.NamedNode:
		return a.lookup(expr, t.Name()).leader()
	case *semantic.Member:
		return a.locateMember(t)
	case *semantic.PointerRange:
		return a.locateRange(t.Pointer)
	case *semantic.SliceRange:
		return a.locate(t.Slice)
	case *semantic.SliceIndex:
		return a.locateElem(t.Slice)
	case *semantic.ArrayIndex:
		return a.locateElem(t.Array)
	case *semantic.MapIndex:
		return a.locateElem(t.Map)
	case *semantic.Cast:
		return a.locate(t.Object)
	case *semantic.Observed:
		return a.locate(t.Parameter)
	default:
		panic(fmt.Errorf("Unexpected type %T in locate %v", expr, expr))
	}
}

// locateElem returns a location leader for an element of the collection
// specified by the expression.
func (a *Annotator) locateElem(expr semantic.Expression) *Location {
	return a.locate(expr).container(expr).Elem()
}

// locateKey returns a location leader for the key of the collection
// specified by the expression.
func (a *Annotator) locateKey(expr semantic.Expression) *Location {
	return a.locate(expr).container(expr).Key()
}

// locateRange returns a location leader for the key of the collection
// specified by the expression.
func (a *Annotator) locateRange(expr semantic.Expression) *Location {
	return a.locate(expr).pointer(expr).Range()
}

// lookup returns a location for an expression in the symbol table. The
// expression and name should match. The name is expected to have been
// previously declared.
func (a *Annotator) lookup(expr semantic.Expression, name string) *Location {
	return a.symbolTable(expr).lookup(name)
}

// boolExpr return a value for a boolean result.
func (a *Annotator) boolExpr() *Location {
	return a.value(semantic.BoolValue(false))
}

// duplex visits the left and right expressions and allows information to flow
// in both directions.
func (a *Annotator) duplex(left semantic.Expression, right semantic.Expression) {
	leftHasPath := hasPath(left, a.api)
	rightHasPath := hasPath(right, a.api)

	r := a.visitExpression(right)
	l := a.visitExpression(left)

	if leftHasPath {
		flow(a.locate(left), r)
	} else if rightHasPath {
		flow(a.locate(right), l)
	}
}

// compare visits the left and right sides of a comparison and returns a result
func (a *Annotator) compare(left semantic.Expression, right semantic.Expression) *Location {
	a.duplex(left, right)
	return a.boolExpr()
}

// value returns a location for a value expression.
func (a *Annotator) value(right semantic.Expression) *Location {
	if isPrimitiveValue(right) {
		return &Location{}
	}
	t := right.ExpressionType()
	switch t := t.(type) {
	case *semantic.Class:
		return newEntity(t).leader()
	case *semantic.Slice, *semantic.Map, *semantic.StaticArray:
		return newContainer().leader()
	case *semantic.Pointer:
		return newPointer().leader()
	case *semantic.Builtin:
		return &Location{}
	default:
		panic(fmt.Errorf("Unexpected value expr=%T:%v type=%T:%v\n", right, right, t, t))
	}
}

// member visits a member expression and returns a location for the field
// specified by the member.
func (a *Annotator) member(right *semantic.Member) *Location {
	nested := a.visitExpression(right.Object).getNested()
	if nested == nil {
		return &Location{}
	}
	if ent, ok := nested.(*entity); ok {
		return ent.Field(right.Field.Name())
	}
	return &Location{}
}

// locateMember returns a leader location for the specified member expression.
func (a *Annotator) locateMember(right *semantic.Member) *Location {
	return a.locate(right.Object).entity(right.Object).Field(right.Field.Name())
}

// TODO is a placeholder for expressions which are not implemented.
func (a *Annotator) TODO(right semantic.Expression) *Location {
	return &Location{}
}

// read associates a Read observation with the location of an expression
func (a *Annotator) read(expr semantic.Expression) *Location {
	if hasPath(expr, a.api) {
		flow(a.locate(expr), newObservation(snippets.ObservationType_Read))
	}
	return a.value(expr)
}

// write associates an Write observation with the location of an expression
func (a *Annotator) write(expr semantic.Expression) *Location {
	if hasPath(expr, a.api) {
		flow(a.locate(expr), newObservation(snippets.ObservationType_Write))
	}
	return a.value(expr)
}

// create makes a new nested entity structure based on a class initializer
func (a *Annotator) create(typ semantic.Type, ini *semantic.ClassInitializer) Nested {
	e := newEntity(typ)
	for _, f := range ini.Fields {
		a.flow(e.Field(f.Field.Name()), f.Value)
	}
	return e
}

// beginScope enter a new scope in the symbol table.
func (a *Annotator) beginScope() {
	a.symbols.enter()
}

// endScope finish the current scope in the symbol table.
func (a *Annotator) endScope() {
	a.symbols.leave()
}

// scoped calls the function f with a symbol space scope.
func (a *Annotator) scoped(f func()) {
	a.beginScope()
	defer a.endScope()
	f()
}

// visitExpression visits a semantic expression and returns a location for
// the value returned by the expression. The return value maybe nil, which
// means nothing of interest.
func (a *Annotator) visitExpression(expr semantic.Expression) *Location {
	switch expr := expr.(type) {
	case nil:
		panic(fmt.Errorf("nil expression passed"))
	case *semantic.Global:
		return a.lookup(expr, expr.Name())
	case *semantic.Local:
		return a.lookup(expr, expr.Name())
	case *semantic.Parameter:
		return a.lookup(expr, expr.Name())
	case *semantic.Select:
		for _, choice := range expr.Choices {
			for _, c := range choice.Conditions {
				a.compare(expr.Value, c)
			}
		}
		return a.boolExpr()
	case *semantic.UnaryOp:
		return a.visitExpression(expr.Expression)
	case *semantic.BinaryOp:
		return a.compare(expr.LHS, expr.RHS)
	case *semantic.BitTest:
		return a.compare(expr.Bitfield, expr.Bits)
	case *semantic.MapContains:
		a.flow(a.locateKey(expr.Map), expr.Key)
		return a.value(expr)
	case *semantic.Member:
		return a.member(expr)
	case *semantic.PointerRange:
		return a.locateRange(expr.Pointer)
	case *semantic.SliceRange:
		return a.visitExpression(expr.Slice)
	case *semantic.ArrayIndex:
		a.flow(a.locateKey(expr.Array), expr.Index)
		return a.elem(expr.Array)
	case *semantic.SliceIndex:
		a.flow(a.locateKey(expr.Slice), expr.Index)
		return a.elem(expr.Slice)
	case *semantic.MapIndex:
		a.flow(a.locateKey(expr.Map), expr.Index)
		return a.elem(expr.Map)
	case *semantic.Callable:
		// TODO: ultimately interested in snippets attached to return value
		return a.TODO(expr)
	case *semantic.Call:
		result := a.TODO(expr.Target)
		if f := expr.Target.Function; f != nil && f.Block != nil {
			args := make([]*Location, len(expr.Arguments))
			for i := range args {
				args[i] = a.visitExpression(expr.Arguments[i])
			}
			a.scoped(func() {
				// Propagate arguments into subroutine scope as parameters.
				for i, p := range f.CallParameters() {
					a.symbols[snippets.SymbolCategory_Parameter].add(p.Name(), args[i])
				}
				// Declare the return 'parameter'.
				if f.Return != nil {
					result = a.declare(f.Return)
				}
				// Process the subroutine.
				a.visitStatement(f.Block)
			})
		}
		return result
	case *semantic.Length:
		return a.value(expr)
	case *semantic.Cast:
		// TODO annotate cast. Needs Type binary.Object and support for Pseudonym.
		return a.visitExpression(expr.Object)
	case *semantic.Create:
		return a.create(expr.Type.To, expr.Initializer).leader()
	case *semantic.Make:
		return a.value(expr)
	case *semantic.Clone:
		return a.read(expr.Slice)
	case *semantic.ClassInitializer:
		return a.create(expr.ExpressionType(), expr).leader()
	case *semantic.Definition:
		a.declare(expr)
		a.accept(expr, expr.Expression)
		return &Location{}
	case *semantic.DefinitionUsage:
		// TODO this type never visited: lookup(expr.Expression, expr.Definition.Name())
		return a.TODO(expr)
	case *semantic.EnumEntry:
		return newLabel(expr.Name())
	case *semantic.Label:
		return newLabel(expr.Name())
	case *semantic.ArrayInitializer:
		c := newContainer()
		e := c.Elem()
		for _, v := range expr.Values {
			a.flow(e, v)
		}
		return c.leader()
	case *semantic.Unknown:
		return a.visitExpression(expr.Inferred)
	case *semantic.Observed:
		return a.visitExpression(expr.Parameter)
	case *semantic.MessageValue:
		return &Location{}
	default:
		if isPrimitiveValue(expr) {
			return a.value(expr)
		}
		panic(fmt.Errorf("Unexpected expression %T %v", expr, expr))
	}
}

// visitStatement visits a semantic node. Information flows between locations
// used in assignment or comparison.
func (a *Annotator) visitStatement(stmt semantic.Node) {
	if isAssign(stmt) {
		dest, src := assignDestSrc(stmt)
		a.accept(dest, src)
		return
	}
	switch stmt := stmt.(type) {
	case nil:
		panic(fmt.Errorf("nil statement passed"))
	case semantic.Expression:
		a.visitExpression(stmt)
	case *semantic.Assert:
		a.visitExpression(stmt.Condition)
	case *semantic.Read:
		a.read(stmt.Slice)
	case *semantic.Write:
		a.write(stmt.Slice)
	case *semantic.Copy:
		a.read(stmt.Src)
		a.write(stmt.Dst)
		a.accept(stmt.Dst, stmt.Src)
	case *semantic.Block:
		a.scoped(func() {
			for _, s := range stmt.Statements {
				a.visitStatement(s)
			}
		})
	case *semantic.Branch:
		a.visitExpression(stmt.Condition)
		a.visitStatement(stmt.True)
		if stmt.False != nil {
			a.visitStatement(stmt.False)
		}
	case *semantic.Switch:
		for _, c := range stmt.Cases {
			for _, cond := range c.Conditions {
				a.compare(stmt.Value, cond)
			}
			a.visitStatement(c.Block)
		}
	case *semantic.Iteration:
		a.scoped(func() {
			a.declare(stmt.Iterator)
			a.visitExpression(stmt.From)
			a.visitExpression(stmt.To)
			a.visitStatement(stmt.Block)
		})
	case *semantic.MapIteration:
		a.scoped(func() {
			a.declare(stmt.IndexIterator)
			a.declare(stmt.KeyIterator)
			a.declare(stmt.ValueIterator)
			a.visitExpression(stmt.Map)
			if hasPath(stmt.Map, a.api) {
				a.flow(a.locateKey(stmt.Map), stmt.KeyIterator)
				a.flow(a.locateElem(stmt.Map), stmt.ValueIterator)
			}
		})
	case *semantic.MapRemove:
		a.flow(a.locateKey(stmt.Map), stmt.Key)
	case *semantic.DeclareLocal:
		a.declare(stmt.Local)
		a.accept(stmt.Local, stmt.Local.Value)
	case *semantic.Return:
		if stmt.Value != nil {
			a.accept(stmt.Function.Return, stmt.Value)
		}
	case *semantic.Abort:
		// don't do anything at the moment.
	case *semantic.Fence:
		if stmt.Statement != nil {
			a.visitStatement(stmt.Statement)
		}
	default:
		panic(fmt.Errorf("Unexpected statement type %T:%v@%p", stmt, stmt, stmt))
	}
}
