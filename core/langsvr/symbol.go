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

package langsvr

import "github.com/google/gapid/core/langsvr/protocol"

// Symbol represents information about programming constructs like variables,
// classes, interfaces etc.
type Symbol struct {
	// The name of this symbol.
	Name string

	// The kind of this symbol.
	Kind SymbolKind

	// The location of this symbol.
	Location Location

	// The name of the symbol containing this symbol.
	Container *Symbol
}

// SymbolList is a list of symbols
type SymbolList []Symbol

// Add appends a new symbol to the list.
func (l *SymbolList) Add(name string, kind SymbolKind, loc Location, container *Symbol) Symbol {
	s := Symbol{name, kind, loc, container}
	*l = append(*l, s)
	return s
}

// Filter returns a new symbol list with only the symbols that pass the
// predicate function.
func (l SymbolList) Filter(pred func(Symbol) bool) SymbolList {
	out := make(SymbolList, 0, len(l))
	for _, s := range l {
		if pred(s) {
			out = append(out, s)
		}
	}
	return out
}

func (s Symbol) toProtocol() protocol.SymbolInformation {
	out := protocol.SymbolInformation{
		Name:     s.Name,
		Kind:     protocol.SymbolKind(s.Kind),
		Location: s.Location.toProtocol(),
	}
	if s.Container != nil {
		out.ContainerName = &s.Container.Name
	}
	return out
}

func (l SymbolList) toProtocol() []protocol.SymbolInformation {
	out := make([]protocol.SymbolInformation, len(l))
	for i, s := range l {
		out[i] = s.toProtocol()
	}
	return out
}

// SymbolKind is an enumerator of symbol kinds.
type SymbolKind int

const (
	KindFile        = SymbolKind(protocol.KindFile)
	KindModule      = SymbolKind(protocol.KindModule)
	KindNamespace   = SymbolKind(protocol.KindNamespace)
	KindPackage     = SymbolKind(protocol.KindPackage)
	KindClass       = SymbolKind(protocol.KindClass)
	KindMethod      = SymbolKind(protocol.KindMethod)
	KindProperty    = SymbolKind(protocol.KindProperty)
	KindField       = SymbolKind(protocol.KindField)
	KindConstructor = SymbolKind(protocol.KindConstructor)
	KindEnum        = SymbolKind(protocol.KindEnum)
	KindInterface   = SymbolKind(protocol.KindInterface)
	KindFunction    = SymbolKind(protocol.KindFunction)
	KindVariable    = SymbolKind(protocol.KindVariable)
	KindConstant    = SymbolKind(protocol.KindConstant)
	KindString      = SymbolKind(protocol.KindString)
	KindNumber      = SymbolKind(protocol.KindNumber)
	KindBoolean     = SymbolKind(protocol.KindBoolean)
	KindArray       = SymbolKind(protocol.KindArray)
)
