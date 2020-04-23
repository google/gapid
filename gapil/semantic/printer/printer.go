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

// Package printer provides a human-readable printer for the semantic tree
// nodes.
package printer

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/gapid/core/text/reflow"
	"github.com/google/gapid/gapil/semantic"
)

func list(l interface{}) string {
	v := reflect.ValueOf(l)
	parts := make([]string, v.Len())
	for i := range parts {
		parts[i] = fmt.Sprintf("%v", v.Index(i).Interface())
	}
	return strings.Join(parts, ", ")
}

// Printer exposes methods for appending string representations of semantic
// Statements, Expressions and Types. While the strings generated closely match
// the syntax of the API file, the Printer has been written to aid in debugging,
// and may not produce parseable output.
type Printer struct {
	reflow *reflow.Writer
	buf    *bytes.Buffer
}

// New returns a new Printer.
func New() *Printer {
	buf := &bytes.Buffer{}
	return &Printer{
		reflow: reflow.New(buf),
		buf:    buf,
	}
}

func (p *Printer) list(list interface{}, print interface{}) {
	l, f := reflect.ValueOf(list), reflect.ValueOf(print)
	for i, c := 0, l.Len(); i < c; i++ {
		f.Call([]reflect.Value{l.Index(i)})
		if i < c-1 {
			p.WriteString(", ")
		}
	}
}

func (p Printer) String() string {
	if p.reflow == nil {
		return "" // Was never initialized
	}
	p.reflow.Flush()
	return p.buf.String()
}

// WriteString appends s to the Printer's buffer.
func (p *Printer) WriteString(s string) {
	p.reflow.Write([]byte(s))
}

// WriteRune appends r cto the Printer's buffer.
func (p *Printer) WriteRune(r rune) {
	p.reflow.WriteRune(r)
}

// WriteFunction appends the string representation of f to the Printer's
// buffer.
func (p *Printer) WriteFunction(f *semantic.Function) *Printer {
	p.WriteString("// order: ")
	p.WriteString(f.Order.String())
	p.WriteString("¶")
	switch {
	case f.Extern:
		p.WriteString("extern ")
	case f.Subroutine:
		p.WriteString("sub ")
	default:
		p.WriteString("cmd ")
	}
	p.WriteType(f.Signature.Return)
	p.WriteRune(' ')
	p.WriteString(f.Name())
	p.WriteRune('(')
	params := f.CallParameters()
	p.list(params, func(param *semantic.Parameter) {
		p.WriteType(param.Type)
		p.WriteRune(' ')
		p.WriteString(param.Name())
	})
	p.WriteString(") ")
	p.WriteStatement(f.Block)
	return p
}

// WriteExpression appends the string representation of n to the Printer's
// buffer.
func (p *Printer) WriteExpression(n semantic.Expression) *Printer {
	switch n := n.(type) {
	case *semantic.ArrayIndex:
		p.WriteExpression(n.Array)
		p.WriteRune('[')
		p.WriteExpression(n.Index)
		p.WriteRune(']')
	case *semantic.ArrayInitializer:
		p.WriteType(n.Array)
		p.WriteRune('{')
		p.list(n.Values, p.WriteExpression)
		p.WriteRune('}')
	case semantic.BoolValue:
		if n {
			p.WriteString("true")
		} else {
			p.WriteString("false")
		}
	case *semantic.BinaryOp:
		p.WriteExpression(n.LHS)
		p.WriteString(n.Operator)
		p.WriteExpression(n.RHS)
	case *semantic.BitTest:
		p.WriteExpression(n.Bits)
		p.WriteString(" in ")
		p.WriteExpression(n.Bitfield)
	case *semantic.Call:
		if n.Target.Object != nil {
			p.WriteExpression(n.Target.Object)
			p.WriteRune('.')
		}
		p.function(n.Target.Function)
		p.WriteRune('(')
		p.list(n.Arguments, p.WriteExpression)
		p.WriteRune(')')
	case *semantic.Cast:
		p.WriteString("as!")
		p.WriteType(n.Type)
		p.WriteRune('(')
		p.WriteExpression(n.Object)
		p.WriteRune(')')
	case *semantic.ClassInitializer:
		p.WriteType(n.Class)
		p.WriteRune('{')
		p.list(n.Fields, func(n *semantic.FieldInitializer) {
			p.WriteExpression(n.Field)
			p.WriteString(": ")
			p.WriteExpression(n.Value)
		})
		p.WriteRune('}')
	case *semantic.Clone:
		p.WriteString("clone(")
		p.WriteExpression(n.Slice)
		p.WriteRune(')')
	case *semantic.Create:
		p.WriteString("new!")
		p.WriteType(n.Type.To)
		p.WriteRune('(')
		p.list(n.Initializer.Fields, func(n *semantic.FieldInitializer) {
			p.WriteExpression(n.Field)
			p.WriteString(": ")
			p.WriteExpression(n.Value)
		})
		p.WriteRune(')')
	case *semantic.EnumEntry:
		p.WriteString(n.Name())
	case *semantic.Field:
		p.WriteString(n.Name())
	case semantic.Float32Value:
		p.WriteString(fmt.Sprintf("%v", float32(n)))
	case semantic.Float64Value:
		p.WriteString(fmt.Sprintf("%v", float64(n)))
	case *semantic.Global:
		p.WriteString(n.Name())
	case *semantic.Ignore:
		p.WriteRune('_')
	case semantic.Int16Value:
		p.WriteString(fmt.Sprintf("%v", int16(n)))
	case semantic.Int32Value:
		p.WriteString(fmt.Sprintf("%v", int32(n)))
	case semantic.Int64Value:
		p.WriteString(fmt.Sprintf("%v", int64(n)))
	case semantic.Int8Value:
		p.WriteString(fmt.Sprintf("%v", int8(n)))
	case *semantic.Length:
		p.WriteString("len(")
		p.WriteExpression(n.Object)
		p.WriteRune(')')
	case *semantic.Local:
		p.WriteString(n.Name())
	case *semantic.Make:
		p.WriteString("make!")
		p.WriteType(n.Type)
		p.WriteRune('(')
		p.WriteExpression(n.Size)
		p.WriteRune(')')
	case *semantic.MapContains:
		p.WriteExpression(n.Key)
		p.WriteString(" in ")
		p.WriteExpression(n.Map)
	case *semantic.MapIndex:
		p.WriteExpression(n.Map)
		p.WriteRune('[')
		p.WriteExpression(n.Index)
		p.WriteRune(']')
	case *semantic.Member:
		p.WriteExpression(n.Object)
		p.WriteRune('.')
		p.WriteExpression(n.Field)
	case *semantic.MessageValue:
		p.WriteType(n.ExpressionType())
		p.WriteRune('(')
		p.list(n.Arguments, func(f *semantic.FieldInitializer) {
			p.WriteString(f.Field.Name())
			p.WriteString(": ")
			p.WriteExpression(f.Value)
		})
		p.WriteRune(')')
	case semantic.Null:
		p.WriteString("null")
	case *semantic.Parameter:
		p.WriteString(n.Named.Name())
	case *semantic.PointerRange:
		p.WriteExpression(n.Pointer)
		p.WriteRune('[')
		p.WriteExpression(n.Range)
		p.WriteRune(']')
	case *semantic.Select:
		p.WriteString("select ")
		p.WriteExpression(n.Value)
		p.WriteString(" {»¶")
		p.list(n.Choices, func(n *semantic.Choice) {
			p.WriteString("case ")
			p.list(n.Conditions, p.WriteExpression)
			p.WriteString(":»¶")
			p.WriteExpression(n.Expression)
			p.WriteString("«¶")
		})
		p.WriteString("")
		p.WriteString("«}¶")
	case *semantic.SliceIndex:
		p.WriteExpression(n.Slice)
		p.WriteRune('[')
		p.WriteExpression(n.Index)
		p.WriteRune(']')
	case *semantic.SliceRange:
		p.WriteExpression(n.Slice)
		p.WriteRune('[')
		p.WriteExpression(n.Range)
		p.WriteRune(']')
	case semantic.StringValue:
		p.WriteString(fmt.Sprintf("%v", string(n)))
	case semantic.Uint16Value:
		p.WriteString(fmt.Sprintf("%v", uint16(n)))
	case semantic.Uint32Value:
		p.WriteString(fmt.Sprintf("%v", uint32(n)))
	case semantic.Uint64Value:
		p.WriteString(fmt.Sprintf("%v", uint64(n)))
	case semantic.Uint8Value:
		p.WriteString(fmt.Sprintf("%v", uint8(n)))
	case *semantic.UnaryOp:
		p.WriteString(n.Operator)
		p.WriteExpression(n.Expression)
	case *semantic.Unknown:
		p.WriteRune('?')
	default:
		panic(fmt.Sprintf("Unknown expression type: %T", n))
	}
	return p
}

// WriteType appends the string representation of n to the Printer's buffer.
func (p *Printer) WriteType(n semantic.Type) *Printer {
	switch n := n.(type) {
	case *semantic.Builtin:
		p.WriteString(n.Name())
	case *semantic.Class:
		p.WriteString(n.Name())
	case *semantic.Enum:
		p.WriteString(n.Name())
	case *semantic.Pointer:
		if n.Const {
			p.WriteString("const ")
		}
		p.WriteType(n.To)
		p.WriteRune('*')
	case *semantic.Pseudonym:
		p.WriteString(n.Name())
	case *semantic.Slice:
		p.WriteType(n.To)
		p.WriteString("[]")
	case *semantic.Signature:
		p.WriteType(n.Return)
		p.WriteRune(' ')
		p.WriteString(n.Name())
		p.WriteRune('(')
		p.list(n.Arguments, func(ty semantic.Type) {
			p.WriteType(ty)
		})
		p.WriteString(")")
	case *semantic.StaticArray:
		p.WriteType(n.ValueType)
		p.WriteRune('[')
		p.WriteString(strconv.Itoa(int(n.Size)))
		p.WriteRune(']')
	case *semantic.Map:
		p.WriteString("map!(")
		p.WriteType(n.KeyType)
		p.WriteString(", ")
		p.WriteType(n.ValueType)
		p.WriteRune(')')
	case *semantic.Reference:
		p.WriteString("ref!")
		p.WriteType(n.To)
	default:
		panic(fmt.Sprintf("Unknown type: %T", n))
	}
	return p
}

func (p *Printer) function(f *semantic.Function) {
	p.WriteString(f.Name())
}

// WriteStatement appends the string representation of n to the Printer's
// buffer.
func (p *Printer) WriteStatement(n semantic.Node) *Printer {
	if !p.statement(n) {
		panic(fmt.Sprintf("Unknown statement type: %T", n))
	}
	return p
}

func (p *Printer) statement(n interface{}) bool {
	switch n := n.(type) {
	case *semantic.Assert:
		p.WriteString("assert(")
		p.WriteExpression(n.Condition)
		p.WriteRune(')')
	case *semantic.Block:
		p.WriteString("{»¶")
		for _, s := range n.Statements {
			p.WriteStatement(s)
			p.WriteString("¶")
		}
		p.WriteString("«}")
	case *semantic.Branch:
		p.WriteString("if ")
		p.WriteExpression(n.Condition)
		p.WriteRune(' ')
		p.WriteStatement(n.True)
		if n.False != nil {
			p.WriteString(" else ¶")
			p.WriteStatement(n.False)
		}
	case *semantic.Call:
		p.WriteExpression(n)
	case *semantic.Copy:
		p.WriteString("copy(")
		p.WriteExpression(n.Dst)
		p.WriteString(", ")
		p.WriteExpression(n.Src)
		p.WriteRune(')')
	case *semantic.Print:
		p.WriteString("print(")
		p.list(n.Arguments, p.WriteExpression)
		p.WriteRune(')')
	case *semantic.Switch:
		p.WriteString("switch ")
		p.WriteExpression(n.Value)
		p.WriteString(" {»¶")
		for _, c := range n.Cases {
			p.WriteStatement(c)
		}
		p.WriteString("«}")
	case *semantic.Case:
		p.WriteString("case ")
		p.list(n.Conditions, p.WriteExpression)
		p.WriteString(": ")
		p.WriteStatement(n.Block)
		p.WriteString("¶")
	case *semantic.Choice:
		p.WriteString("case ")
		p.list(n.Conditions, p.WriteExpression)
		p.WriteString(": ")
		p.WriteExpression(n.Expression)
		p.WriteString("¶")
	case *semantic.Iteration:
		p.WriteString("TODO: ITERATION")
	case *semantic.MapIteration:
		p.WriteString("TODO: MAP ITERATION")
	case *semantic.Assign:
		p.WriteExpression(n.LHS)
		p.WriteRune(' ')
		p.WriteString(n.Operator)
		p.WriteRune(' ')
		p.WriteExpression(n.RHS)
	case *semantic.ArrayAssign:
		p.WriteExpression(n.To)
		p.WriteRune(' ')
		p.WriteString(n.Operator)
		p.WriteRune(' ')
		p.WriteExpression(n.Value)
	case *semantic.MapAssign:
		p.WriteExpression(n.To)
		p.WriteRune(' ')
		p.WriteString(n.Operator)
		p.WriteRune(' ')
		p.WriteExpression(n.Value)
	case *semantic.MapRemove:
		p.WriteString("delete(")
		p.WriteExpression(n.Map)
		p.WriteString(", ")
		p.WriteExpression(n.Key)
		p.WriteRune(')')
	case *semantic.MapClear:
		p.WriteString("clear(")
		p.WriteExpression(n.Map)
		p.WriteRune(')')
	case *semantic.SliceAssign:
		p.WriteExpression(n.To)
		p.WriteString(n.Operator)
		p.WriteExpression(n.Value)
	case *semantic.DeclareLocal:
		p.WriteExpression(n.Local)
		p.WriteString(" := ")
		p.WriteExpression(n.Local.Value)
	case *semantic.Read:
		p.WriteString("read(")
		p.WriteExpression(n.Slice)
		p.WriteRune(')')
	case *semantic.Return:
		if n.Value != nil {
			p.WriteString("return ")
			p.WriteExpression(n.Value)
		} else {
			p.WriteString("return")
		}
	case *semantic.Write:
		p.WriteString("write(")
		p.WriteExpression(n.Slice)
		p.WriteRune(')')
	case *semantic.Abort:
		p.WriteString("abort")
	case *semantic.Fence:
		p.WriteString("fence")
	default:
		return false
	}
	return true
}
