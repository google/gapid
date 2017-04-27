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
	"fmt"
	"strings"

	"sort"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

type xmlStartElement struct {
	rootHolder
	lineNumber uint32
	comment    stringPoolRef
	namespace  stringPoolRef
	name       stringPoolRef
	attributes xmlAttributeList
}

func (c *xmlStartElement) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(header), device.LittleEndian)
	c.lineNumber = r.Uint32()
	c.comment = c.root().decodeString(r)
	if err := r.Error(); err != nil {
		return err
	}

	r = endian.Reader(bytes.NewReader(data), device.LittleEndian)
	c.namespace = c.root().decodeString(r)
	c.name = c.root().decodeString(r)
	attributeStart := r.Uint16()
	attributeSize := r.Uint16()
	attributeCount := r.Uint16()
	if err := r.Error(); err != nil {
		return err
	}
	if attributeSize != xmlAttributeSize {
		return fmt.Errorf("Attribute size was not as expected. Got %d, expected %d",
			attributeSize, xmlAttributeSize)
	}
	if r.Uint16() != 0 {
		return fmt.Errorf("idIndex != 0 not supported.")
	}
	if r.Uint16() != 0 {
		return fmt.Errorf("classIndex != 0 not supported.")
	}
	if r.Uint16() != 0 {
		return fmt.Errorf("styleIndex != 0 not supported.")
	}

	r = endian.Reader(bytes.NewReader(data[attributeStart:]), device.LittleEndian)
	c.attributes = make([]xmlAttribute, attributeCount)
	for i := range c.attributes {
		c.attributes[i].decode(r, c.root())
	}
	return r.Error()
}

func (c *xmlStartElement) updateContext(ctx *xmlContext) {
	ctx.indent++
	ctx.stack.push(c)
}

func (c *xmlStartElement) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	b.WriteString(strings.Repeat(ctx.tab, ctx.indent))
	b.WriteRune('<')
	b.WriteString(c.name.get())
	if ns, ok := ctx.stack.head().(*xmlStartNamespace); ok {
		b.WriteRune('\n')
		b.WriteString(strings.Repeat(ctx.tab, ctx.indent+2))
		b.WriteString(`xmlns:`)
		b.WriteString(ns.namespacePrefix.get())
		b.WriteString(`="`)
		b.WriteString(ns.namespaceURI.get())
		b.WriteString(`" `)
	}
	b.WriteString(c.attributes.xml(ctx))
	b.WriteRune('>')
	c.updateContext(ctx)
	return b.String()
}

func (c *xmlStartElement) encode() []byte {
	return encodeChunk(resXMLStartElementType, func(w binary.Writer) {
		w.Uint32(c.lineNumber)
		c.comment.encode(w)
	}, func(w binary.Writer) {
		c.namespace.encode(w)
		c.name.encode(w)
		w.Uint16(20)                        // attributeStart
		w.Uint16(xmlAttributeSize)          // attributeSize
		w.Uint16(uint16(len(c.attributes))) // attributeCount
		w.Uint16(0)                         // id_index
		w.Uint16(0)                         // class_index
		w.Uint16(0)                         // style_index
		for _, at := range c.attributes {
			at.encode(w)
		}
	})
}

func (c *xmlStartElement) addAttribute(attr *xmlAttribute) {
	c.attributes = append(c.attributes, *attr)
	sort.Sort(attributesByResourceId{c.attributes, c.root()})
}
