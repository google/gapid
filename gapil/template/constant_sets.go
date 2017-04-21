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

package template

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

type constsetBuilder struct {
	sets    []constset.Set // Deterministic ordered.
	setIdx  map[string]int // Used for removing duplicates.
	symOff  map[string]int // Symbol offsets.
	symbols bytes.Buffer   // Full packed symbols.
}

func (b *constsetBuilder) build() constset.Pack {
	return constset.Pack{
		Symbols: constset.Symbols(b.symbols.String()),
		Sets:    b.sets,
	}
}

func (b *constsetBuilder) addSym(sym string) (int, int) {
	offset, ok := b.symOff[sym]
	if !ok {
		offset = b.symbols.Len()
		b.symOff[sym] = offset
		b.symbols.WriteString(sym)
	}
	return offset, len(sym)
}

func (b *constsetBuilder) addLabels(labels analysis.Labels, isBitfield bool) int {
	keys := make(u64s, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	set := constset.Set{
		IsBitfield: isBitfield,
		Entries:    make([]constset.Entry, len(keys)),
	}
	for i, k := range keys {
		set.Entries[i].V = k
		set.Entries[i].O, set.Entries[i].L = b.addSym(labels[k])
	}

	id := fmt.Sprintf("%+v", set)
	idx, ok := b.setIdx[id]
	if !ok {
		idx = len(b.sets)
		b.setIdx[id] = idx
		b.sets = append(b.sets, set)
	}
	return idx
}

type u64s []uint64

func (p u64s) Len() int           { return len(p) }
func (p u64s) Less(i, j int) bool { return p[i] < p[j] }
func (p u64s) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ConstantSets consatins the constset.Pack, and each semantic mapping to a
// constset.Set index.
type ConstantSets struct {
	Pack constset.Pack
	Sets map[semantic.Node]int
}

type nodeLabeler struct {
	nodes map[semantic.Node]analysis.Labels
	seen  map[analysis.Value]bool
}

func (nl *nodeLabeler) addLabels(n semantic.Node, v analysis.Value) {
	ev, ok := v.(*analysis.EnumValue)
	if !ok || len(ev.Labels) == 0 {
		return // Parameter wasn't an enum, or had no labels.
	}
	l, ok := nl.nodes[n]
	if !ok {
		l = analysis.Labels{}
		nl.nodes[n] = l
	}
	l.Merge(ev.Labels)
}

func (nl *nodeLabeler) traverse(n semantic.Node, v analysis.Value) {
	if nl.seen[v] {
		return
	}
	nl.seen[v] = true
	nl.addLabels(n, v)
	switch v := v.(type) {
	case *analysis.ClassValue:
		for n, fv := range v.Fields {
			nl.traverse(v.Class.FieldByName(n), fv)
		}
	case *analysis.MapValue:
		for k, v := range v.KeyToValue {
			nl.traverse(nil, k)
			nl.traverse(nil, v)
		}
	}
}

func buildConstantSets(api *semantic.API, mappings *resolver.Mappings) *ConstantSets {
	b := &constsetBuilder{
		setIdx: map[string]int{},
		symOff: map[string]int{},
	}

	a := analysis.Analyze(api, mappings)

	constsets := map[semantic.Node]int{}

	for _, f := range api.Functions {
		for _, p := range f.FullParameters {
			v, ok := a.Parameters[p]
			if !ok {
				continue // No analysis for this parameter.
			}
			ev, ok := v.(*analysis.EnumValue)
			if !ok || len(ev.Labels) == 0 {
				continue // Parameter wasn't an enum, or had no labels.
			}
			constsets[p] = b.addLabels(ev.Labels, ev.Ty.IsBitfield)
		}
	}

	nl := nodeLabeler{
		nodes: map[semantic.Node]analysis.Labels{},
		seen:  map[analysis.Value]bool{},
	}
	// Gather all the labeled nodes.
	for n, v := range a.Globals {
		nl.traverse(n, v)
	}
	for n, v := range a.Instances {
		nl.traverse(n, v)
	}
	// Create constant sets for all the nodes.
	for n, l := range nl.nodes {
		isBitfield := false // TODO
		constsets[n] = b.addLabels(l, isBitfield)
	}

	return &ConstantSets{
		Pack: b.build(),
		Sets: constsets,
	}
}

func (f *Functions) constantSets() *ConstantSets {
	if f.cs != nil {
		return f.cs
	}
	f.cs = buildConstantSets(f.api, f.mappings)
	return f.cs
}

// ConstantSets returns the full constants set pack for the API.
func (f *Functions) ConstantSets() constset.Pack {
	return f.constantSets().Pack
}

// ConstantSetIndex returns the constant set for the given parameter.
func (f *Functions) ConstantSetIndex(n semantic.Node) int {
	if i, ok := f.constantSets().Sets[n]; ok {
		return i
	}
	return -1
}
