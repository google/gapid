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

package atom

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapil/snippets"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

// Dynamic is a wrapper around a schema.Object that describes an atom object.
// Dynamic conforms to the Atom interface, and provides a number of methods
// for accessing the parameters, return value and observations.
type Dynamic struct {
	object *schema.Object
	info   *atomInfo
	extras Extras
	flags  Flags
}

var (
	_ Atom          = &Dynamic{} // Verify that Dynamic implements Atom.
	_ binary.Object = &Dynamic{} // Verify that Dynamic implements Atom.
)

// API returns the graphics API id this atom belongs to.
func (a *Dynamic) API() gfxapi.API {
	if a.info.meta == nil {
		return nil
	}
	return gfxapi.Find(a.info.meta.API)
}

// AtomFlags returns the flags of the atom.
func (a *Dynamic) AtomFlags() Flags {
	return a.flags
}

// Extras returns all the Extras associated with the dynamic atom.
func (a *Dynamic) Extras() *Extras {
	return &a.extras
}

// Mutate is not supported by the Dynamic type, but is exposed in order to comform
// to the Atom interface. Mutate will always return an error.
func (*Dynamic) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return fmt.Errorf("Mutate not implemented for client atoms")
}

// ParameterCount returns the number of parameters this atom accepts. This count
// does not include the return value.
func (a *Dynamic) ParameterCount() int {
	return len(a.info.parameters)
}

// Parameter returns the index'th parameter Field and value.
func (a *Dynamic) Parameter(index int) (binary.Field, interface{}) {
	index = a.info.parameters[index]
	return a.object.Type.Fields[index], a.object.Fields[index]
}

// SetParameter sets the atom's index'th parameter to the specified value.
func (a *Dynamic) SetParameter(index int, value interface{}) {
	index = a.info.parameters[index]
	a.object.Fields[index] = value
}

// Result returns the atom's return Field and value. If the atom does not have
// a return value then nil, nil is returned.
func (a *Dynamic) Result() (*binary.Field, interface{}) {
	if a.info.result < 0 {
		return nil, nil
	}
	return &a.object.Type.Fields[a.info.result], a.object.Fields[a.info.result]
}

// SetResult sets the atom's result to the specified value.
func (a *Dynamic) SetResult(value interface{}) {
	if a.info.result < 0 {
		panic("Atom has no result")
	}
	a.object.Fields[a.info.result] = value
}

// Class returns the serialize information and functionality for this type.
func (a *Dynamic) Class() binary.Class {
	return dynamicClass{a.object.Class()}
}

type dynamicClass struct{ binary.Class }

func (c dynamicClass) Encode(e binary.Encoder, o binary.Object) {
	c.Class.Encode(e, o.(*Dynamic).object)
}

func (c dynamicClass) DecodeTo(d binary.Decoder, o binary.Object) {
	c.Class.DecodeTo(d, o.(*Dynamic).object)
}

// StringWithConstants returns the string description of the atom and its
// arguments. If constants is not nil it is used to provide symbolic values
// for arguments of the corresponding types.
func (a *Dynamic) StringWithConstants(constants []schema.ConstantSet) string {
	o := a.object
	params := make([]string, 0, len(o.Type.Fields))
	result := ""
	allLabels := findLabels(o.Type)
	meta := FindMetadata(o.Class().Schema())
	var displayName string
	if meta == nil {
		displayName = lowerCaseFirstCharacter(o.Type.Display)
	} else {
		displayName = meta.DisplayName
	}
	for i, f := range o.Type.Fields {
		v := o.Fields[i]
		// If v is represented as a memory pointer, use that instead.
		if dyn, ok := v.(*schema.Object); ok {
			if mp := tryMemoryPointer(dyn); mp != nil {
				v = *mp
			}
		} else if prim, pok := f.Type.(*schema.Primitive); pok {
			if cs := findConstantSet(prim, constants); cs != nil {
				// It is an "enum" try to come up with a single label, if possible.
				if candidates := findConstants(v, *cs); len(candidates) != 0 {
					if len(candidates) == 1 {
						v = candidates[0].Name
					} else {
						// Use the label snippets to disambiguate
						var preferred []schema.Constant

						// lowerCaseFirstCharacter ~ see b/27585620
						path := snippets.Field(snippets.Relative(displayName),
							lowerCaseFirstCharacter(f.Declared))
						labels := findParamLabels(allLabels, path)
						if labels != nil {
							for _, c := range candidates {
								for _, l := range labels.Labels {
									if c.Name == l {
										preferred = append(preferred, c)
									}
								}
							}
							if len(preferred) == 0 {
								// If the preferred list is empty, fallback to the originals.
								preferred = candidates
							}
						} else {
							preferred = candidates
						}
						if len(preferred) < 8 {
							// Use shortest heuristic
							v = pickShortestName(preferred)
						} else {
							// Too ambiguous just print "type(value)"
							v = fmt.Sprintf("%v(%v)", prim, v)
						}
					}
				} else {
					v = fmt.Sprintf("%v(%v)", prim, v)
				}
			}
		}
		if i != a.info.extras && i != a.info.result {
			params = append(params, fmt.Sprintf("%v: %v", f.Name(), v))
		}
		if i == a.info.result {
			result = fmt.Sprintf("-> %v", v)
		}
	}
	name := a.object.Type.Name()
	if a.info.meta != nil {
		name = a.info.meta.DisplayName
	}
	return fmt.Sprintf("%v(%v)%v", name, strings.Join(params, ", "), result)
}

// String returns the string description of the atom and its arguments.
func (a *Dynamic) String() string {
	return a.StringWithConstants(nil)
}

// atomInfo is a cache of precalculated atom information from the schema.
type atomInfo struct {
	meta       *Metadata
	extras     int   // index on fields, or -1
	parameters []int // indices on fields
	result     int   // index on fields, or -1
}

// typeName generate a canonical type name from a possibly relative name
func typeName(pkg string, t binary.Type) string {
	str := t.String()
	repr := fmt.Sprintf("%r", t)
	if repr == str {
		return str
	}
	if strings.Index(str, ".") == -1 {
		return pkg + "." + str
	}
	return str
}

// pickShortestName, from a set of constants choice the shortest name string
func pickShortestName(constants []schema.Constant) string {
	shortLen := math.MaxInt32
	shortest := ""
	for _, constant := range constants {
		l := len(constant.Name)
		if l < shortLen {
			shortLen = l
			shortest = constant.Name
		}
	}
	return shortest
}

// findConstantSet, find the constant set for a given type or return nil
func findConstantSet(t binary.Type, constants []schema.ConstantSet) *schema.ConstantSet {
	for _, cs := range constants {
		if t.String() == cs.Type.String() {
			return &cs
		}
	}
	return nil
}

// findConstants, find all the possible constants in cs for a given value v.
func findConstants(v interface{}, cs schema.ConstantSet) []schema.Constant {
	var constants []schema.Constant
	for _, c := range cs.Entries {
		if v == c.Value {
			constants = append(constants, c)
		}
	}
	return constants
}

// tryMemoryPointer tries to convert an object to a memory pointer
// if the schema representation is compatible.
// There are several aliases for Memory.Pointer which are unique types,
// but we want to render them as pointers.
func tryMemoryPointer(o *schema.Object) *memory.Pointer {
	mp := memory.Pointer{}
	mpSchema := mp.Class().Schema()
	mpFields := mpSchema.Fields

	if len(mpFields) != len(o.Type.Fields) {
		return nil
	}

	for i, f := range o.Type.Fields {
		ft := typeName(o.Type.Package, f.Type)
		mpt := typeName(mpSchema.Package, mpFields[i].Type)
		if f.Declared != mpFields[i].Declared || ft != mpt {
			return nil
		}
	}

	mp.Address = o.Fields[0].(uint64)
	mp.Pool = memory.PoolID(o.Fields[1].(uint32))
	return &mp
}

// findLabels collects label snippets from the entities metadata
func findLabels(entity *schema.ObjectClass) []*snippets.Labels {
	var labels []*snippets.Labels
	for _, obj := range entity.Metadata {
		if l, ok := obj.(*snippets.Labels); ok {
			labels = append(labels, l)
		}
	}
	return labels
}

// findParamLabels finds labels for the specified parameter path.
func findParamLabels(allLabels []*snippets.Labels, path snippets.Pathway) *snippets.Labels {
	if allLabels == nil {
		return nil
	}
	for _, l := range allLabels {
		if snippets.Equal(l.Path, path) {
			return l
		}
	}
	return nil
}

// lowerCaseFirstCharacter return str with the first character lowercased.
func lowerCaseFirstCharacter(str string) string {
	if len(str) == 0 {
		return str
	}
	a := []rune(str)
	a[0] = unicode.ToLower(a[0])
	return string(a)
}

func newAtomInfo(entity *binary.Entity) *atomInfo {
	meta := FindMetadata(entity)
	class := &atomInfo{meta: meta, extras: -1, result: -1}
	// Find the extras, if present
	for i, f := range entity.Fields {
		if s, ok := f.Type.(*schema.Slice); ok {
			if _, ok := s.ValueType.(*schema.Interface); ok {
				if f.Declared == "extras" {
					class.extras = i
					continue
				}
			}
		}
		if f.Name() == "Result" {
			class.result = i
			continue
		}
		class.parameters = append(class.parameters, i)
	}
	return class
}

var (
	wrappers = map[binary.Signature]*atomInfo{}
)

func wrapTo(a *Dynamic, obj *schema.Object) error {
	entity := obj.Class().Schema()
	info := wrappers[entity.Signature()]
	if info == nil {
		info = newAtomInfo(entity)
		wrappers[entity.Signature()] = info
	}
	a.info, a.object = info, obj
	if info.extras >= 0 {
		if info.extras >= len(a.object.Fields) {
			return fmt.Errorf("Missing extras field in %s", entity.Name())
		}
		value := a.object.Fields[info.extras]
		extras, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("Extras field is of type %T in %s", value, entity.Name())
		}
		a.extras = make(Extras, len(extras))
		for i := range extras {
			a.extras[i] = extras[i].(Extra)
		}
	}
	if info.meta != nil {
		if info.meta.DrawCall {
			a.flags |= DrawCall
		}
		if info.meta.EndOfFrame {
			a.flags |= EndOfFrame
		}
	}
	return nil
}

func Wrap(obj *schema.Object) (Atom, error) {
	a := &Dynamic{}
	if err := wrapTo(a, obj); err != nil {
		return nil, err
	}
	return a, nil
}
