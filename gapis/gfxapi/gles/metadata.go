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

package gles

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

// Annotate all atoms in the namespace
func AddMetadata(n *registry.Namespace) {
	n.Visit(func(c binary.Class) {
		ent := c.Schema()
		if ent == nil {
			return
		}
		if atom.FindMetadata(ent) != nil {
			// Don't add metadata, if there is already metadata. Duplicates
			// can happen because it is not easy to distinguish Frozen from
			// Generated.
			return
		}
		u := n.LookupUpgrader(ent.Signature())
		if u == nil {
			return
		}
		obj := u.New()
		if a, ok := obj.(atom.Atom); ok {
			atom.AddMetadata(a, ent)
		} else if s, ok := obj.(slice); ok {
			addSliceMetadata(s, ent)
		}
	})
}

func init() {
	binary_init()
	AddMetadata(Namespace)
	if err := atom.AddSnippetsFromBase64String(
		Namespace, embedded[snippets_base64_file]); err != nil {
		panic(fmt.Errorf("Error decoding atom snippets: %v", err))
	}
	var state *State
	entity := state.Class().Schema()
	if err := gfxapi.AddStateSnippetsFromBase64String(
		entity,
		embedded[globals_snippets_base64_file]); err != nil {
		panic(fmt.Errorf("Error decoding global state snippets: %v", err))
	}
	gfxapi.AddCanFollowState(reflect.TypeOf(state).Elem(), &entity.Metadata)
}

func addSliceMetadata(s slice, ent *binary.Entity) {
	meta := &memory.SliceMetadata{
		ElementTypeName: s.ElementTypeName(),
	}
	ent.Metadata = append(ent.Metadata, meta)
}
