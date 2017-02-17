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
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/google/gapid/core/gapil/snippets"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/vle"
)

// SnippetsByAtomName is a mapping from atom name to a slice of snippet
// groups for that atom.
type SnippetsByAtomName map[string][]snippets.KindredSnippets

// loadSnippets decodes the input reader and populates the receiver with
// the decoded snippet objects. Return non-nil if there was an error reading
// or decoding. The reader is a stream of Variant of type
// *snippets.AtomSnippets.
func (a SnippetsByAtomName) loadSnippets(in io.Reader) error {
	d := cyclic.Decoder(vle.Reader(in))
	for {
		obj := d.Variant()
		if d.Error() != nil {
			if d.Error() == io.EOF {
				return nil
			}
			return d.Error()
		}
		atomSnips, ok := obj.(*snippets.AtomSnippets)
		if !ok {
			return fmt.Errorf("Expected snippets.AtomSnippets got %T", obj)
		}
		a[atomSnips.AtomName] = atomSnips.Snippets
	}
}

// addErr build an error from a previous (possibly nil) error and a
// new one.
func addErr(prev error, err error) error {
	if prev == nil {
		return err
	} else {
		return fmt.Errorf("%v also\n%v", prev, err)
	}
}

// AddSnippets add snippets from the receiver to the specified namespace.
// Returns an error if the receiver contained atoms which were not in
// the namespace.
func (a SnippetsByAtomName) AddSnippets(n *registry.Namespace) error {
	seen := make(map[string]*binary.Entity)
	var err error
	n.VisitDirect(func(c binary.Class) {
		ent := c.Schema()
		if ent == nil {
			return
		}
		meta := FindMetadata(ent)
		if meta == nil {
			return
		}
		if snips, ok := a[meta.DisplayName]; ok {
			for _, v := range snips {
				ent.Metadata = append(ent.Metadata, v)
			}
			if other, ok := seen[meta.DisplayName]; ok {
				err = addErr(err, fmt.Errorf("Display name duplicate %s for %v and %v",
					meta.DisplayName, other.Signature, ent.Signature))
				return
			}
			seen[meta.DisplayName] = ent
		}
	})
	if err != nil {
		return err
	}
	if len(seen) != len(a) {
		return fmt.Errorf("Did not find %d atoms which have snippets",
			len(a)-len(seen))
	}
	return nil
}

// AddSnippetsFromReader decodes the input reader and adds snippet objects
// to the entities in the namespaces. Returns an error if there was an error
// reading or decoding or there were atoms which were not in the namespace.
func AddSnippetsFromReader(n *registry.Namespace, reader io.Reader) error {
	a := make(SnippetsByAtomName)
	if err := a.loadSnippets(reader); err != nil {
		return err
	}
	return a.AddSnippets(n)
}

// AddSnippetsFromFile opens the specified filename and decodes it.
// Snippet objects from the file are added to the entities in the namespaces.
// Returns a non-nil error if there was an error reading or decoding or
// there were atoms which were not in the namespace.
func AddSnippetsFromFile(n *registry.Namespace, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return AddSnippetsFromReader(n, bufio.NewReader(f))
}

// AddSnippetsFromBase64String decodes the specified string as base64 encoded
// binary decoder input. Decoded snippet objects are added to the entities in
// the namespaces. Returns a non-nil error if there was an error decoding base64
// or binary decoding or if there were atoms which were not in the namespace.
// Base64 encoding is used so that the binary encoded string can be embedded
// directly in the library using the "embed" tool.
func AddSnippetsFromBase64String(n *registry.Namespace, b64 string) error {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return AddSnippetsFromReader(n, bytes.NewBuffer(buf))
}
