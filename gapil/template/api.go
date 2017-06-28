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
	"fmt"
	"reflect"

	"github.com/google/gapid/gapil/semantic"
)

// Unpack throws away type information to work around template system limitations
// When you have a value of an interface type that carries methods, it fails to
// introspect the concrete type for its members, so the template can't see them.
// The result of Upack no longer has a type, so the concrete type members become
// visible.
func (*Functions) Unpack(v interface{}) interface{} { return v }

// GetAnnotation finds and returns the annotation on ty with the specified name.
// If the annotation cannot be found, or ty does not support annotations then
// GetAnnotation returns nil.
func (*Functions) GetAnnotation(ty interface{}, name string) *semantic.Annotation {
	a, ok := ty.(semantic.Annotated)
	if !ok {
		return nil
	}
	return a.GetAnnotation(name)
}

// WithAnnotation returns the list l filtered to those items with the specified
// annotation.
func (*Functions) WithAnnotation(name string, l interface{}) []interface{} {
	v := reflect.ValueOf(l)
	out := []interface{}{}
	for i, c := 0, v.Len(); i < c; i++ {
		n := v.Index(i).Interface()
		if n, ok := n.(semantic.Annotated); ok && n.GetAnnotation(name) != nil {
			out = append(out, n)
		}
	}
	return out
}

// WithoutAnnotation returns the list l filtered to those items without the
// specified annotation.
func (*Functions) WithoutAnnotation(name string, l interface{}) []interface{} {
	v := reflect.ValueOf(l)
	out := []interface{}{}
	for i, c := 0, v.Len(); i < c; i++ {
		n := v.Index(i).Interface()
		if n, ok := n.(semantic.Annotated); !ok || n.GetAnnotation(name) == nil {
			out = append(out, n)
		}
	}
	return out
}

// Underlying returns the underlying type for ty by recursively traversing the
// pseudonym chain until reaching and returning the first non-pseudoym type.
// If ty is not a pseudonym then it is simply returned.
func (f *Functions) Underlying(ty semantic.Type) semantic.Type {
	return semantic.Underlying(ty)
}

// Decompose returns the fundamental building block of a type. The intent
// is to provide type information to enhance the memory display. Arrays and
// Slices are decomposed into their element type, other types decompose into
// their underlying type. The builtin string decomposes to char. Pointers
// decompose into pointers to their decomposed To type. AnyType is returned
// for composite types which do not decompose into a builtin, pointer or
// an enum. As a special case semantic.StringType is returned if ty is
// a pointer to "char*" or "const char*".
func (f *Functions) Decompose(ty semantic.Type) semantic.Type {
	switch t := ty.(type) {
	case *semantic.Pseudonym:
		return f.Decompose(t.To)
	case *semantic.StaticArray:
		return f.Decompose(t.ValueType)
	case *semantic.Slice:
		return f.Decompose(t.To)
	case *semantic.Builtin:
		if t == semantic.StringType {
			return semantic.CharType
		}
		return t
	case *semantic.Pointer:
		to := f.Decompose(t.To)
		if t.To == to {
			return t
		}
		u := f.Underlying(t.To)
		if u == semantic.CharType {
			// This is a work around to the fact that I can't synthesize new
			// pointer types. Essentially "char*" and "const char*" become string.
			return semantic.StringType
		}
		return semantic.AnyType // Unknown
	case *semantic.Enum:
		// Do not decompose into an integer.
		return t
	default:
		return semantic.AnyType // Unknown
	}
}

// AllCommands returns a list of all cmd entries for a given API, regardless
// of whether they are free functions, class methods or pseudonym methods.
func (f *Functions) AllCommands(api interface{}) ([]interface{}, error) {
	switch api := api.(type) {
	case *semantic.API:
		var commands []interface{}
		for _, function := range api.Functions {
			commands = append(commands, function)
		}
		for _, class := range api.Classes {
			for _, method := range class.Methods {
				commands = append(commands, method)
			}
		}
		for _, pseudonym := range api.Pseudonyms {
			for _, method := range pseudonym.Methods {
				commands = append(commands, method)
			}
		}
		return commands, nil
	default:
		return nil, fmt.Errorf("first argument must be of type *semantic.API, was %T", api)
	}
}

// PackageOf walks the ownership hierarchy to find the API that the supplied
// object belongs to. If it is not the api being processed, then the import
// name of the api is returned.
func (f *Functions) PackageOf(v semantic.Node) string {
	switch v := v.(type) {
	case *semantic.API:
		if f.api == v {
			return ""
		}
		pkg := "unknown_"
		if f.api.Imported != nil {
			f.api.Imported.Visit(func(name string, node semantic.Node) {
				if api, ok := node.(*semantic.API); ok {
					if api == v {
						pkg = name
					}
				}
			})
		}
		return pkg
	case semantic.Owned:
		return f.PackageOf(v.Owner())
	default:
		return ""
	}
}

// TokenOf returns the cst token string that represents the supplied semantic node
func (f *Functions) TokenOf(v semantic.Node) string {
	ast := f.mappings.SemanticToAST[v]
	if len(ast) == 0 {
		return "*no ast*"
	}
	return f.mappings.CST(ast[0]).Token().String()
}
