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

package generate

import (
	"bytes"
	"fmt"
	"io"

	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/framework/binary/vle"

	"sort"

	"github.com/google/gapid/framework/binary"
)

type byID []*Struct

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].Signature() < a[j].Signature() }

type stream struct {
	name    string
	b       bytes.Buffer
	e       binary.Encoder
	d       binary.Decoder
	mode    binary.Mode
	count   int
	size    int
	largest int
	total   int
}

func newStream(mode binary.Mode) *stream {
	s := &stream{mode: mode}
	s.name = fmt.Sprint(mode)
	s.reset()
	return s
}

func (s *stream) reset() {
	s.b.Reset()
	s.e = cyclic.Encoder(vle.Writer(&s.b))
	s.e.SetMode(s.mode)
	s.d = cyclic.Decoder(vle.Reader(&s.b))
}

func (s *stream) test(e *binary.Entity) {
	s.e.Entity(e)
	s.size += s.b.Len()
	if s.e.Error() != nil {
		panic(fmt.Errorf("Failed encoding %s entity for %q, %v", s.name, e.Signature(), s.e.Error()))
	}
	got := s.d.Entity()
	if got == nil || s.d.Error() != nil {
		panic(fmt.Errorf("Failed reading %s entity for %q, %v", s.name, e.Signature(), s.d.Error()))
	}
	if e.Signature() != got.Signature() {
		panic(fmt.Errorf("Signature of %s entity did not match, expected %q got %q", s.name, e.Signature(), got.Signature()))
	}
	if s.mode != binary.Compact {
		es := fmt.Sprint(e)
		gots := fmt.Sprint(got)
		if es != gots {
			panic(fmt.Errorf("Full encoding did not match, expected %#v got %#v", es, gots))
		}
	}
}

func (s *stream) measure(e *binary.Entity) int {
	s.reset()
	s.e.Entity(e)
	start := s.b.Len()
	// now encode the entity directly to bypass the table
	schema.EncodeEntity(s.e, e)
	size := s.b.Len() - start + 2
	s.total += size
	s.count++
	if s.largest < size {
		s.largest = size
	}
	return size
}

func (s *stream) stats(w io.Writer) {
	fmt.Fprintln(w, "Total:", s.total)
	fmt.Fprintln(w, "Average:", s.total/s.count)
	fmt.Fprintln(w, "Largest:", s.largest)
}

func WriteAllSignatures(w io.Writer, modules Modules) {
	structs := []*Struct{}
	for _, m := range modules {
		if m.IsTest {
			continue
		}
		structs = append(structs, m.Structs...)
	}
	sort.Sort(byID(structs))
	full := newStream(binary.Full)
	compact := newStream(binary.Compact)
	// pre write the entire schema so the lookup table is full, and verify the encode/decode behaviour while doing it
	for _, s := range structs {
		full.test(&s.Entity)
		compact.test(&s.Entity)
	}
	for _, s := range structs {
		fs := full.measure(&s.Entity)
		cs := compact.measure(&s.Entity)
		fmt.Fprintln(w)
		mode := ""
		switch {
		case s.Entity.IsPOD():
			mode = "pod"
		case s.Entity.IsSimple():
			mode = "simple"
		default:
			mode = "entity"
		}
		fmt.Fprintln(w, s.Name(), ":", mode, ": full", fs, "compact", cs)
		fmt.Fprintln(w, s.Entity.Signature())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Schema stats:")
	fmt.Fprintln(w, "Count:", len(structs))
	fmt.Fprintln(w, "Full:", full.size)
	full.stats(w)
	fmt.Fprintln(w, "Compact:", compact.size)
	compact.stats(w)
}
