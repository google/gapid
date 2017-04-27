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

type xmlCData struct {
	rootHolder
	lineNumber uint32
	comment    stringPoolRef
	data       stringPoolRef
	typedValue typedValue
}

func (c *xmlCData) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(header), device.LittleEndian)
	c.lineNumber = r.Uint32()
	c.comment = c.root().decodeString(r)
	if err := r.Error(); err != nil {
		return err
	}

	r = endian.Reader(bytes.NewReader(data), device.LittleEndian)
	c.data = c.root().decodeString(r)
	tv, err := decodeValue(r, c.root())
	if err != nil {
		return err
	}
	if err := r.Error(); err != nil {
		return err
	}

	c.typedValue = tv
	return nil
}

func (c *xmlCData) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	b.WriteString(strings.Repeat(ctx.tab, ctx.indent))
	b.WriteString("<![CDATA[")
	b.WriteString(c.data.get())
	b.WriteString("]]>")
	return b.String()
}

func (c *xmlCData) encode() []byte {
	return encodeChunk(resXMLCDataType, func(w binary.Writer) {
		w.Uint32(c.lineNumber)
		c.comment.encode(w)
	}, func(w binary.Writer) {
		c.data.encode(w)
		c.typedValue.encode(w)
	})
}
