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

package gfxapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"unicode"

	"github.com/google/gapid/core/gapil/snippets"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/vle"
	"github.com/google/gapid/gapis/service/path"
)

// GlobalSnippets is kindred grouped table of snippets with paths that apply
// to the global state. It is used to provide snippets for the API state
// object and ultimately the state view in the UI.
type GlobalSnippets []snippets.KindredSnippets

// loadSnippets decodes the input reader and populates the receiver with
// the decoded snippet objects. Return non-nil if there was an error reading
// or decoding. The reader is a stream of Variant of type
// snippets.KindredSnippets.
func (g *GlobalSnippets) loadSnippets(in io.Reader) error {
	d := cyclic.Decoder(vle.Reader(in))
	for {
		obj := d.Variant()
		if d.Error() != nil {
			if d.Error() == io.EOF {
				return nil
			}
			return d.Error()
		}
		snips, ok := obj.(snippets.KindredSnippets)
		if !ok {
			return fmt.Errorf("Expected snippets.KindredSnippets got %T", obj)
		}
		*g = append(*g, snips)
	}
}

// AddStateSnippetsFromReader decodes the input reader and adds snippet objects
// to the specified entity which should be the schema entity for the API
// state object associated with the snippets. Returns an error if there
// was an error reading or decoding.
func AddStateSnippetsFromReader(stateEntity *binary.Entity, reader io.Reader) error {
	var g GlobalSnippets
	if err := g.loadSnippets(reader); err != nil {
		return err
	}
	for _, snip := range g {
		stateEntity.Metadata = append(stateEntity.Metadata, snip)
	}
	return nil
}

// AddStateSnippetsFromBase64String decodes the specified string as base64
// encoded binary decoder input. Decoded snippet objects are added to the
// specified entity which should be the schema entity for the API state object
// associated with the snippets. Returns a non-nil error if there was an
// error decoding base64 or binary decoding. Base64 encoding is used so that
// the binary encoded string can be embedded directly in the library using
// the "embed" tool.
func AddStateSnippetsFromBase64String(stateEntity *binary.Entity, b64 string) error {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return AddStateSnippetsFromReader(stateEntity, bytes.NewBuffer(buf))
}

type seenStack []reflect.Type

func (seen seenStack) contains(t reflect.Type) bool {
	for _, tt := range seen {
		if t == tt {
			return true
		}
	}
	return false
}

// Generate snippets for CanFollow this information is not captured in the
// API file, but only in the Go code by the interface path.Linker.
// Consequently we generate these snippets at startup using reflection.

// addCanFollow adds CanFollow snippets for the type 't' and its children.
func addCanFollow(pth snippets.Pathway, t reflect.Type, snips *[]binary.Object, seen seenStack) {
	if seen.contains(t) {
		// Avoid cycles
		return
	}
	seen = append(seen, t)

	if t.Kind() == reflect.Ptr {
		// Note pointer types in Go are a consequence of reference types in the
		// API language, so we just dereference them.
		addCanFollow(pth, t.Elem(), snips, seen)
		return
	}

	if t.Kind() != reflect.Struct && pth == nil {
		panic(fmt.Errorf("non-struct type %v at top-level", t.Name))
	}

	linker := reflect.TypeOf((*path.Linker)(nil)).Elem()
	if t.Implements(linker) {
		*snips = append(*snips, &snippets.CanFollow{Path: pth})
	}

	// Visit the children.
	switch t.Kind() {
	case reflect.Struct:
		addCanFollowStruct(pth, t, snips, seen, false)
	case reflect.Array, reflect.Slice:
		addCanFollow(snippets.Elem(pth), t.Elem(), snips, seen)
	case reflect.Map:
		addCanFollow(snippets.Key(pth), t.Key(), snips, seen)
		addCanFollow(snippets.Elem(pth), t.Elem(), snips, seen)
	}
}

// addCanFollowStruct adds CanFollow snippets to a struct and all its children.
func addCanFollowStruct(path snippets.Pathway, t reflect.Type, snips *[]binary.Object, seen seenStack, fixCase bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if len(f.PkgPath) != 0 {
			// Non-empty package path means this field is unexported, ignore.
			continue
		}
		if i == 0 && f.Anonymous && f.Type.Kind() == reflect.Struct &&
			f.Type.NumField() == 0 {
			// Look for tags on the binary.Generate field
			typeName := f.Tag.Get("display")
			globals := f.Tag.Get("globals")
			if globals != "" {
				if fixCase {
					// Fix case only applies to parameters, not globals
					return
				}
				if globals != snippets.GlobalsTypename {
					panic(fmt.Errorf("Type %v is tagged globals:\"%s\". Does not match with \"%s\"", t, globals, snippets.GlobalsTypename))
				}
				typeName = globals
			}
			if path == nil {
				if typeName == "" {
					return
				}
				path = snippets.Relative(typeName)
			}
			continue
		}
		addCanFollow(field(path, f.Name, fixCase), f.Type, snips, seen)
	}
}

func addCanFollowAtom(t reflect.Type, snips *[]binary.Object) {
	addCanFollowStruct(nil, t, snips, seenStack{t}, true)
}

// AddCanFollowState add CanFollow snippets for the global API state object.
func AddCanFollowState(t reflect.Type, snips *[]binary.Object) {
	addCanFollowStruct(nil, t, snips, seenStack{t}, false)
}

func lowerCaseFirstCharacter(str string) string {
	a := []rune(str)
	a[0] = unicode.ToLower(a[0])
	return string(a)
}

func field(path snippets.Pathway, name string, fixcase bool) snippets.Pathway {
	if fixcase {
		// See b/27585620
		name = lowerCaseFirstCharacter(name)
	}
	return snippets.Field(path, name)
}

func init() {
	linker := reflect.TypeOf((*path.Linker)(nil)).Elem()

	linkableMetaFactory := func(t reflect.Type, e *binary.Entity) {
		defer func() {
			if r := recover(); r != nil {
				panic(fmt.Errorf("panic during %v: %v", t.Name, r))
			}
		}()
		// Add CanFollow snippets for all the atoms
		if t.Implements(linker) {
			// Ideally we would use atom.Atom here, but that creates a cycle.
			return
		}
		// Add CanFollow snippets to the parameters of the atom
		addCanFollowAtom(t, &e.Metadata)
	}
	registry.Factories.AddMetaFactory(linkableMetaFactory)
}
