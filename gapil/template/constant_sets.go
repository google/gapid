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
	"sort"

	"fmt"

	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapil/semantic"
)

// constsetEntryBuilder is a transient structure used to build a constset.Entry.
type constsetEntryBuilder struct {
	name string
	val  uint64
}
type constsetEntryBuilderList []constsetEntryBuilder

// constsetSetBuilder is a transient structure used to build a constset.Set.
type constsetSetBuilder struct {
	isBitfield bool // true if this set is for a bitfield.
	entries    constsetEntryBuilderList
	idx        *int // pointer to index of the Set in the built Pack.
}

// constsetPackBuilder is a transient structure used to build a constset.Pack.
type constsetPackBuilder struct {
	syms   map[string]struct{}            // contains all the used symbol names.
	sets   map[string]*constsetSetBuilder // set builder key to builder.
	setIdx map[string]int                 // set builder key to index in sets.
}

func (l constsetEntryBuilderList) Len() int           { return len(l) }
func (l constsetEntryBuilderList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l constsetEntryBuilderList) Less(i, j int) bool { return l[i].val < l[j].val }

func (b *constsetPackBuilder) addLabels(labels analysis.Labels, isBitfield bool) *int {
	set := &constsetSetBuilder{
		isBitfield: isBitfield,
		entries:    make([]constsetEntryBuilder, 0, len(labels)),
	}
	for v, n := range labels {
		b.syms[n] = struct{}{}
		set.entries = append(set.entries, constsetEntryBuilder{n, v})
	}
	sort.Sort(set.entries)
	key := fmt.Sprintf("%v %+v", isBitfield, set.entries)
	if set, ok := b.sets[key]; ok {
		return set.idx
	}
	set.idx = new(int)
	b.sets[key] = set
	return set.idx
}

func (b *constsetPackBuilder) sortedSyms() []string {
	syms := make([]string, 0, len(b.syms))
	for sym := range b.syms {
		syms = append(syms, sym)
	}
	sort.Strings(syms)
	return syms
}

func (b *constsetPackBuilder) build() constset.Pack {
	out := constset.Pack{}

	offsets := map[string]int{}
	for _, s := range b.sortedSyms() {
		offsets[s] = len(out.Symbols)
		out.Symbols += constset.Symbols(s)
	}

	keys := make([]string, 0, len(b.sets))
	for key := range b.sets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out.Sets = make([]constset.Set, len(keys))
	for i, key := range keys {
		set := b.sets[key]
		*set.idx = i
		entries := make([]constset.Entry, len(set.entries))
		for i, e := range set.entries {
			entries[i] = constset.Entry{
				V: e.val,
				O: offsets[e.name],
				L: len(e.name),
			}
		}
		out.Sets[i] = constset.Set{
			IsBitfield: set.isBitfield,
			Entries:    entries,
		}
	}

	return out
}

type u64s []uint64

func (p u64s) Len() int           { return len(p) }
func (p u64s) Less(i, j int) bool { return p[i] < p[j] }
func (p u64s) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ConstantSets consatins the constset.Pack, and each semantic mapping to a
// constset.Set index.
type ConstantSets struct {
	Pack constset.Pack
	Sets map[semantic.Node]*int
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
			nl.traverse(n, k)
			nl.traverse(n, v)
		}
	}
}

func buildConstantSets(api *semantic.API, mappings *semantic.Mappings) *ConstantSets {
	b := &constsetPackBuilder{
		syms:   map[string]struct{}{},
		sets:   map[string]*constsetSetBuilder{},
		setIdx: map[string]int{},
	}

	a := analysis.Analyze(api, mappings)

	constsets := map[semantic.Node]*int{}

	// Create a constant set for all the API's enums that didn't opt out
	for _, e := range api.Enums {
		// "analyze_usage" opts out of the default constant set generation,
		// uses analysis to determine which constants should go together
		if e.Annotations.GetAnnotation("analyze_usage") == nil {
			labels := analysis.Labels{}
			for _, entry := range e.Entries {
				value, ok := semantic.AsUint64(entry.Value)
				if !ok {
					panic(fmt.Sprintf("Unsupported enum number type: %v", e.NumberType))
				}
				// In cases a constant value has multiple labels (multiple
				// labels defining the same constant value in the same enum),
				// the last label will be adopted as the name of the constant
				// value.
				labels[value] = string(entry.Named)
			}
			constsets[e] = b.addLabels(labels, e.IsBitfield)
		}
	}

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
			// Check if we already created a constset for this parameter type
			if idx, ok := constsets[ev.Ty]; ok {
				constsets[p] = idx
			} else {
				// This type has been annotated with "analyze_usage"
				constsets[p] = b.addLabels(ev.Labels, ev.Ty.IsBitfield)
			}
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
		isBitfield := false
		var idx *int
		if ty, err := semantic.TypeOf(n); err == nil {
			if enum, ok := ty.(*semantic.Enum); ok {
				if i, ok := constsets[enum]; ok {
					idx = i
				}
				isBitfield = enum.IsBitfield
			}
		}
		if idx == nil {
			idx = b.addLabels(l, isBitfield)
		}
		constsets[n] = idx
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
		return *i
	}
	return -1
}
