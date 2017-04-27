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
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

type xmlEndElement struct {
	rootHolder
	lineNumber uint32
	comment    stringPoolRef
	namespace  stringPoolRef
	name       stringPoolRef
}

func (c xmlEndElement) String() string {
	b := bytes.Buffer{}
	return b.String()
}

func (c *xmlEndElement) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(header), device.LittleEndian)
	c.lineNumber = r.Uint32()
	c.comment = c.root().decodeString(r)
	if err := r.Error(); err != nil {
		return err
	}
	r = endian.Reader(bytes.NewReader(data), device.LittleEndian)
	c.namespace = c.root().decodeString(r)
	c.name = c.root().decodeString(r)
	return r.Error()
}

func (c *xmlEndElement) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	c.updateContext(ctx)
	b.WriteString(strings.Repeat(ctx.tab, ctx.indent))
	b.WriteString("</")
	b.WriteString(c.name.get())
	b.WriteRune('>')
	return b.String()
}

func (c *xmlEndElement) updateContext(ctx *xmlContext) {
	ctx.stack.pop()
	ctx.indent--
}

func (c *xmlEndElement) encode() []byte {
	return encodeChunk(resXMLEndElementType, func(w binary.Writer) {
		w.Uint32(c.lineNumber)
		c.comment.encode(w)
	}, func(w binary.Writer) {
		c.namespace.encode(w)
		c.name.encode(w)
	})
}
