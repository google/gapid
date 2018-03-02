// Copyright (C) 2018 Google Inc.
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

// Package ia64 implements a subset of the symbol mangling for the itanium ABI
// (standard for GCC).
//
// See: https://itanium-cxx-abi.github.io/cxx-abi/abi.html#mangling
package ia64

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/gapil/compiler/mangling"
)

// Mangle returns the entity mangled conforming to the IA64 ABI.
func Mangle(s mangling.Entity) string {
	m := mangler{
		bytes.Buffer{},
		map[mangling.Entity]int{},
	}
	m.mangle(s)
	return m.String()
}

type mangler struct {
	bytes.Buffer
	subs map[mangling.Entity]int
}

func (m *mangler) mangle(v mangling.Entity) {
	m.WriteString("_Z")
	m.encoding(v)
}

func (m *mangler) encoding(v mangling.Entity) {
	if _, ok := v.(mangling.Named); !ok {
		unhandled("encoding", v)
	}
	m.name(v)
	if f, ok := v.(*mangling.Function); ok {
		m.bareFunctionType(f)
	}
}

func (m *mangler) name(v mangling.Entity) {
	switch v := v.(type) {
	case mangling.Scoped:
		s := v.Scope()
		switch {
		case s == nil || isStdNamespace(s):
			// Entities declared at global scope, or in namespace std, are
			// mangled as unscoped names
			m.unscoped(v)
			return
		case isFunction(s):
			// Entities declared within a function, including members of local
			// classes, are mangled with <local-name>.
			m.local(v)
		case isNamespace(s), isClass(s):
			// Entities declared in a namespace or class scope are mangled with
			// <nested-name>.
			m.nested(v)
		default:
			unhandled("name", v)
		}
	}
}

func isFunction(v mangling.Entity) bool  { _, ok := v.(*mangling.Function); return ok }
func isNamespace(v mangling.Entity) bool { _, ok := v.(*mangling.Namespace); return ok }
func isClass(v mangling.Entity) bool     { _, ok := v.(*mangling.Class); return ok }

func (m *mangler) bareFunctionType(f *mangling.Function) {
	if isTemplated(f) {
		m.ty(f.Return)
	}

	if len(f.Parameters) > 0 {
		for _, p := range f.Parameters {
			m.ty(p)
		}
	} else {
		m.ty(mangling.Void)
	}
}

func (m *mangler) nested(v mangling.Entity) {
	m.WriteRune('N')
	defer m.WriteRune('E')

	m.cvQualifiers(v)
	m.refQualifiers(v)

	if isTemplated(v) {
		m.templatePrefix(v)
		m.templateArgs(v)
	} else {
		if scope := parentScope(v); scope != nil {
			m.prefix(scope)
		}
		m.unqualified(v)
	}

	// TODO: N [<CV-qualifiers>] [<ref-qualifier>] <template-prefix> <template-args> E
}

func (m *mangler) unscoped(v mangling.Entity) {
	if isStdNamespace(v) {
		m.WriteString("St")
	}
	m.unqualified(v)
}

func (m *mangler) unqualified(v mangling.Entity) {
	// TODO: <operator-name> [<abi-tags>]
	// TODO: <ctor-dtor-name>
	m.source(v)
	// TODO: <unnamed-type-name>
	// TODO: DC <source-name>+ E
}

func parentScope(v mangling.Entity) mangling.Scope {
	if s, ok := v.(mangling.Scoped); ok {
		return s.Scope()
	}
	return nil
}

func (m *mangler) templatePrefix(v mangling.Entity) {
	// TODO: <template-param> # template template parameter
	m.substitution(v, func() {
		if scope := parentScope(v); scope != nil {
			m.prefix(scope)
		}
		m.unqualified(v)
	})
}

func (m *mangler) templateArgs(v mangling.Entity) {
	m.WriteRune('I')
	for _, t := range v.(mangling.Templated).TemplateArguments() {
		m.ty(t)
	}
	m.WriteRune('E')
}

func (m *mangler) prefix(v mangling.Entity) {
	if isTemplated(v) {
		m.templatePrefix(v)
		m.templateArgs(v)
		return
	}

	m.substitution(v, func() {
		switch {
		case isClass(v), isNamespace(v):
			if scope := parentScope(v); scope != nil {
				m.prefix(scope)
			}
			m.unqualified(v)
			return
		}
		unhandled("prefix", v)

		// TODO: <template-param>                   # template type parameter
		// TODO: <decltype>                         # decltype qualifier
		// TODO: <prefix> <data-member-prefix>      # initializer of a data member
		// TODO: <substitution>
	})
}

func (m *mangler) ty(t mangling.Type) {
	switch t := t.(type) {
	case mangling.Builtin: // <builtin-type>
		m.builtin(t)
	case *mangling.Class: // <class-enum-type>
		m.substitution(t, func() { m.name(t) })
	case mangling.Pointer: // # pointer
		m.substitution(t, func() {
			m.WriteRune('P')
			m.ty(t.To)
		})
	case mangling.TemplateParameter:
		m.substitution(t, func() {
			m.WriteRune('T')
			if t == 0 {
				m.WriteRune('_')
			} else {
				m.WriteString(fmt.Sprintf("%d_", t-1))
			}
		})
	default:
		// TODO: <function-type>
		// TODO: <qualified-type>
		// TODO: <class-enum-type>
		// TODO: <array-type>
		// TODO: <pointer-to-member-type>
		// TODO: <template-param>
		// TODO: <template-template-param> <template-args>
		// TODO: <decltype>
		// TODO: R <type>        # l-value reference
		// TODO: O <type>        # r-value reference (C++11)
		// TODO: C <type>        # complex pair (C99)
		// TODO: G <type>        # imaginary (C99)
		unhandled("type", t)
	}
}

func (m *mangler) qualifiedTy(t mangling.Type) {
	m.qualifiers(t)
	m.ty(t)
}

func (m *mangler) qualifiers(t mangling.Type) {
	// extendedQualifiers
	m.cvQualifiers(t)
}

func (m *mangler) cvQualifiers(v mangling.Entity) {
	switch v := v.(type) {
	case *mangling.Function:
		if v.Const {
			m.WriteRune('K')
		}
	}
}

func (m *mangler) local(v mangling.Entity) { panic("Not implemented") }

func (m *mangler) builtin(t mangling.Type) {
	switch t {
	case mangling.Void:
		m.WriteRune('v')
	case mangling.WChar:
		m.WriteRune('w')
	case mangling.Bool:
		m.WriteRune('b')
	case mangling.Char:
		m.WriteRune('c')
	case mangling.SChar:
		m.WriteRune('a')
	case mangling.UChar:
		m.WriteRune('h')
	case mangling.Short:
		m.WriteRune('s')
	case mangling.UShort:
		m.WriteRune('t')
	case mangling.Int:
		m.WriteRune('i')
	case mangling.UInt:
		m.WriteRune('j')
	case mangling.Long:
		m.WriteRune('l')
	case mangling.ULong:
		m.WriteRune('m')
	case mangling.S64:
		m.WriteRune('x')
	case mangling.U64:
		m.WriteRune('y')
	case mangling.Float:
		m.WriteRune('f')
	case mangling.Double:
		m.WriteRune('d')
	case mangling.Ellipsis:
		m.WriteRune('z')
	default:
		unhandled("builtin", t)
	}
}

func (m *mangler) refQualifiers(v mangling.Entity) {}

func (m *mangler) source(v mangling.Entity) {
	n, ok := v.(mangling.Named)
	if !ok {
		unhandled("source", v)
	}
	name := n.GetName()
	m.WriteString(fmt.Sprint(len(name)))
	m.WriteString(name)
}

func isTemplated(v mangling.Entity) bool {
	switch v := v.(type) {
	case mangling.Templated:
		return len(v.TemplateArguments()) > 0
	}
	return false
}

func isStdNamespace(v mangling.Entity) bool {
	if n, ok := v.(*mangling.Namespace); ok {
		return n.Name == "std" && n.Parent == nil
	}
	return false
}

func isStdAllocator(v mangling.Entity) bool {
	if c, ok := v.(*mangling.Class); ok && c.Name == "allocator" {
		return isStdNamespace(c.Parent)
	}
	return false
}

func isStdBasicString(v mangling.Entity) bool {
	if c, ok := v.(*mangling.Class); ok && c.Name == "basic_string" {
		return isStdNamespace(c.Parent)
	}
	return false
}

func (m *mangler) substitution(v mangling.Entity, f func()) {
	switch {
	case isStdNamespace(v):
		m.WriteString("St")
	case isStdAllocator(v):
		m.WriteString("Sa")
	case isStdBasicString(v):
		m.WriteString("Sb")

	default:
		// TODO: Ss # ::std::basic_string < char,
		//            ::std::char_traits<char>,
		//            ::std::allocator<char> >
		// TODO: Si # ::std::basic_istream<char,  std::char_traits<char> >
		// TODO: So # ::std::basic_ostream<char,  std::char_traits<char> >
		// TODO: Sd # ::std::basic_iostream<char, std::char_traits<char> >

		if s, ok := m.subs[v]; ok {
			if s == 0 {
				m.WriteString("S_")
			} else {
				m.WriteString(fmt.Sprintf("S%v_", s-1))
			}
		} else {
			f()
			m.subs[v] = len(m.subs)
		}
	}
}

func unhandled(kind string, val mangling.Entity) {
	panic(fmt.Errorf("Unhandled %v: %T(%+v)", kind, val, val))
}
