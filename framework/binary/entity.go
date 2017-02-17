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

package binary

import (
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Entity represents the encodable type information for an object.
// In its compact mode, the entity contains only the information strictly required to generate its signature, and not
// any of metadata or display names.
type Entity struct {
	Package  string    // The package that declared the struct.
	Display  string    // The display name of the class, not set in compact form.
	Identity string    // The true name of the class.
	Version  string    // The version string of the class, if set.
	Exported bool      // Whether the class is exported from its package, not set in compact form.
	Fields   FieldList // Descriptions of the fields of the class.
	Metadata []Object  // The metadata for the class, not set in compact form

	subspace *Subspace // Cache of the sub-types of this entity
	// signature is the cached result of a call to Signature.
	// It is never encoded to a stream.
	signature string
}

func spaceToUnderscore(r rune) rune {
	if unicode.IsSpace(r) {
		return '_'
	}
	return r
}

// TypeName returns a string for the typename in a form compatible with
// the name codegen would use for this schema type. If t is in the package
// pkg a relative name should be used.
func TypeName(t reflect.Type, pkg string) string {
	pkgPath := t.PkgPath()
	if pkgPath == "" || pkg == pkgPath {
		return t.Name()
	}
	fullname := pkgPath + "." + t.Name()
	return strings.Map(spaceToUnderscore, path.Base(fullname))
}

// MakeTypeFun is a function type for building schema types. The function
// makes a schema type corresponding to t. If t has sub-types then
// makeType will be called to build them. Names given to schema sub-types
// should be relative to pkg.
type MakeTypeFun func(
	t reflect.Type, tag reflect.StructTag, makeType MakeTypeFun, pkg string) Type

// InitEntity initializes a schema entity 'e' from the Go reflect type 't'.
// 'makeType' is used to build schema types for sub-types.
func InitEntity(t reflect.Type, makeType MakeTypeFun, e *Entity) {
	if t.Kind() != reflect.Struct {
		panic(fmt.Errorf("Type %v is not struct-kind (kind = %v)", t, t.Kind()))
	}

	pkg := t.PkgPath()
	e.Package = path.Base(pkg)
	name := t.Name()
	runes := ([]rune)(name)
	if len(runes) != 0 {
		e.Exported = unicode.IsUpper(runes[0])
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if v, _ := strconv.ParseBool(f.Tag.Get("disable")); v {
			continue
		}
		if i == 0 && f.Anonymous && f.Type.Kind() == reflect.Struct &&
			f.Type.NumField() == 0 {
			e.Version = f.Tag.Get("version")
			e.Identity = f.Tag.Get("identity")
			continue
		}
		sf := Field{Declared: f.Name, Type: makeType(f.Type, f.Tag, makeType, pkg)}
		e.Fields = append(e.Fields, sf)
	}
	if e.Identity == "" {
		// Always set the Identity.
		e.Identity = name
	} else if e.Identity != name {
		// Only set display if it differs from Identity.
		e.Display = name
	}
	e.Signature() // This caches the signature in the entity
}

// Name returns the name of the Entity.
func (e *Entity) Name() string {
	if e.Display != "" {
		return e.Display
	}
	return e.Identity
}

type Signature string

// Signature returns a canonical string representations of an entities signature.
// If two entities have the same Signature, the are assumed to represent the same type
func (e *Entity) Signature() Signature {
	if e.signature == "" {
		// Internally signature is implemented by the fmt.Formatter interface with the format specifier 'z'
		e.signature = fmt.Sprintf("%z", e)
	}
	return Signature(e.signature)
}

// IsPOD checks if the entity is a valid POD type
func (e *Entity) IsPOD() bool {
	for _, field := range e.Fields {
		if !field.Type.IsPOD() {
			return false
		}
	}
	return true
}

// IsSimple checks if the entity is a valid Simple type
func (e *Entity) IsSimple() bool {
	for _, field := range e.Fields {
		if !field.Type.IsSimple() {
			return false
		}
	}
	return true
}

// Format implements the fmt.Formatter interface
func (e *Entity) Format(f fmt.State, c rune) {
	// if c is 'z' then we are printing in signature format.
	// this is an internal implementation detail, the only code that should ever do this is the Signature method.
	fmt.Fprint(f, e.Package, ".", e.Identity)
	if e.Version != "" {
		fmt.Fprint(f, '@', e.Version)
	}
	if c != 'z' && e.Display != "" {
		fmt.Fprint(f, '(', e.Display, ")")
	}
	fmt.Fprint(f, "{")
	for i, field := range e.Fields {
		if i != 0 {
			fmt.Fprint(f, ",")
		}
		if c != 'z' && field.Declared != "" {
			fmt.Fprint(f, field.Declared, " ")
		}
		field.Type.Format(f, c)
	}
	fmt.Fprint(f, "}")
}

// appendSubTypes, appends the sub-types of 't' to the list 'l' and returns
// it. Inlines are expanded. If 't' is inline then append the expansion
// of its sub-types to the list 'l' and return it. If 't' is not inline
// just append 't' and return it.
func appendSubTypes(l TypeList, t SubspaceType) TypeList {
	if !t.HasSubspace() {
		return l
	}
	s := t.Subspace()
	if !s.Inline {
		return append(l, t)
	}
	for _, sub := range s.SubTypes {
		l = appendSubTypes(l, sub)
	}
	return l
}

// Subspace returns the subspace for the entity. The subspace of the
// entity is the the catenation of the field subtypes with inlines expanded.
func (e *Entity) Subspace() *Subspace {
	if e.subspace != nil {
		return e.subspace
	}
	sub := TypeList{}
	for _, f := range e.Fields {
		if f.Type.HasSubspace() {
			sub = appendSubTypes(sub, f.Type)
		}
	}
	e.subspace = &Subspace{SubTypes: sub}
	return e.subspace
}

// ExpandSubTypes returns the list of sub-types of this subspace with
// any inline sub-types expanded.
func (s *Subspace) ExpandSubTypes() TypeList {
	if s.expanded != nil {
		return s.expanded
	}
	l := TypeList{}
	for _, t := range s.SubTypes {
		l = appendSubTypes(l, t)
	}
	s.expanded = l
	return l
}

// FieldList is a slice of fields.
type FieldList []Field

// Field represents a name/type pair for a field in an Object.
type Field struct {
	Declared string // The name of the field, not set in compact form.
	Type     Type   // The type stored in the field.
}

// Subspace represents the sub-types which need decoder support for nested types.
type Subspace struct {
	Inline   bool     // true if the schema object inlines the encoding (arrays)
	Counted  bool     // true if the schema type is a counted (slice, map)
	SubTypes TypeList // the complete list of subtypes
	expanded TypeList // cache of the expanded list of subtypes
}

// SubspaceType is the interface provides by schema types in order for them
// to provide sub-type information to the decoder.
type SubspaceType interface {
	// Returns true if Subspace() will return non-nil.
	HasSubspace() bool
	// Subspace returns the subspace for this type. The subspace represents
	// the sub-types which need decoder support for nested types.
	// Nil means that this kind of schema object never has subtypes.
	Subspace() *Subspace
}

// Type represents the common interface to all type objects in the schema.
type Type interface {
	SubspaceType
	String() string         // The true name of the type.
	Representation() string // The encoded representation of the type.
	EncodeValue(e Encoder, value interface{})
	DecodeValue(d Decoder) interface{}
	Format(f fmt.State, c rune)
	IsPOD() bool
	IsSimple() bool
}

// TypeList used to represent the entities of any composed subtypes which
// need decoding.
type TypeList []SubspaceType

func trimPackage(n string) string {
	i := strings.LastIndex(n, ".")
	if i < 0 {
		return n
	}
	return n[i+1:]
}

func (f Field) Name() string {
	if f.Declared == "" && f.Type != nil {
		return trimPackage(f.Type.String())
	}
	return f.Declared
}

// Find searches the field list of the field with the specified name, returning
// the index of the field if found, otherwise -1.
func (l FieldList) Find(name string) int {
	for i, f := range l {
		if f.Name() == name {
			return i
		}
	}
	return -1
}
