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

package binaryxml

import (
	"bytes"
	"strings"

	"github.com/google/gapid/core/data/binary"
)

type xmlAttributeList []xmlAttribute

func (l xmlAttributeList) forName(s stringPoolRef) (*xmlAttribute, bool) {
	for idx, at := range l {
		if at.name == s {
			return &l[idx], true
		}
	}
	return nil, false
}

func (l xmlAttributeList) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	for _, a := range l {
		b.WriteRune('\n')
		b.WriteString(strings.Repeat(ctx.tab, ctx.indent+2))
		b.WriteString(a.xml(ctx))
	}
	return b.String()
}

type xmlAttribute struct {
	namespace  stringPoolRef
	name       stringPoolRef
	rawValue   stringPoolRef
	typedValue typedValue
}

func (a xmlAttribute) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	if a.namespace.isValid() {
		ns := a.namespace.get()
		if prefix, ok := ctx.namespaces[ns]; ok {
			b.WriteString(prefix)
		} else {
			b.WriteString(a.namespace.get())
		}
		b.WriteRune(':')
	}
	b.WriteString(a.name.get())
	b.WriteRune('=')
	b.WriteRune('"')
	if a.rawValue.isValid() {
		b.WriteString(a.rawValue.get())
	} else {
		b.WriteString(a.typedValue.String())
	}
	b.WriteRune('"')
	return b.String()
}

const xmlAttributeSize = 20

func (a *xmlAttribute) decode(r binary.Reader, root *xmlTree) error {
	a.namespace = root.decodeString(r)
	a.name = root.decodeString(r)
	a.rawValue = root.decodeString(r)
	typedValue, err := decodeValue(r, root)
	a.typedValue = typedValue
	return err
}

func (a *xmlAttribute) encode(w binary.Writer) {
	a.namespace.encode(w)
	a.name.encode(w)
	a.rawValue.encode(w)
	a.typedValue.encode(w)
}

type attributesByResourceId struct {
	attributes xmlAttributeList
	xml        *xmlTree
}

func (as attributesByResourceId) Len() int {
	return len(as.attributes)
}

func (as attributesByResourceId) Swap(i, j int) {
	as.attributes[i], as.attributes[j] = as.attributes[j], as.attributes[i]
}

func (as attributesByResourceId) Less(i, j int) bool {
	rm := as.xml.resourceMap

	a := as.attributes[i].name
	r1 := uint32(0xffffffff)
	if a.stringPoolIndex() < uint32(len(rm.ids)) {
		r1 = rm.ids[a.stringPoolIndex()]
	}

	b := as.attributes[j].name
	r2 := uint32(0xffffffff)
	if b.stringPoolIndex() < uint32(len(rm.ids)) {
		r2 = rm.ids[b.stringPoolIndex()]
	}

	return r1 < r2 || ((r1 == r2) && a.get() < b.get())
}
