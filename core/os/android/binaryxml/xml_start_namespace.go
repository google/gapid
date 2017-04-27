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

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

type xmlStartNamespace struct {
	rootHolder
	lineNumber      uint32
	comment         stringPoolRef
	namespacePrefix stringPoolRef
	namespaceURI    stringPoolRef
}

func (c *xmlStartNamespace) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(header), device.LittleEndian)
	c.lineNumber = r.Uint32()
	c.comment = c.root().decodeString(r)
	if err := r.Error(); err != nil {
		return err
	}
	r = endian.Reader(bytes.NewReader(data), device.LittleEndian)
	c.namespacePrefix = c.root().decodeString(r)
	c.namespaceURI = c.root().decodeString(r)
	return r.Error()
}

func (c *xmlStartNamespace) xml(ctx *xmlContext) string {
	c.updateContext(ctx)
	return ""
}

func (c *xmlStartNamespace) updateContext(ctx *xmlContext) {
	nsPrefix := c.namespacePrefix.get()
	nsURI := c.namespaceURI.get()
	ctx.namespaces[nsURI] = nsPrefix
	ctx.stack.push(c)
}

func (c *xmlStartNamespace) encode() []byte {
	return encodeChunk(resXMLStartNamespaceType, func(w binary.Writer) {
		w.Uint32(c.lineNumber)
		c.comment.encode(w)
	}, func(w binary.Writer) {
		c.namespacePrefix.encode(w)
		c.namespaceURI.encode(w)
	})
}
