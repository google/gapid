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
	"io"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

type xmlTree struct {
	rootHolder
	strings     *stringPool
	resourceMap *xmlResourceMap
	chunks      []chunk
}

func (c xmlTree) xml(ctx *xmlContext) string {
	b := bytes.Buffer{}
	for _, chunk := range c.chunks {
		s := chunk.xml(ctx)
		b.WriteString(s)
		b.WriteRune('\n')
	}
	return b.String()
}

func (c *xmlTree) visit(visitor chunkVisitor) {
	ctx := xmlContext{
		strings:    c.strings,
		namespaces: map[string]string{},
		tab:        "  ",
	}

	visitor(&ctx, c, beforeContextChange)
	for _, chunk := range c.chunks {
		visitor(&ctx, chunk, beforeContextChange)
		ctxChange, ok := chunk.(contextChange)
		if ok {
			ctxChange.updateContext(&ctx)
			visitor(&ctx, chunk, afterContextChange)
		}
	}
}

func (c *xmlTree) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)

	var chunk chunk
	var err error
	var ok bool

	if chunk, err = decodeChunk(r, c); err != nil {
		return err
	}
	if c.strings, ok = chunk.(*stringPool); !ok {
		return fmt.Errorf("Expected string pool chunk, got %T", chunk)
	}

	if chunk, err = decodeChunk(r, c); err != nil {
		return err
	}
	if c.resourceMap, ok = chunk.(*xmlResourceMap); !ok {
		return fmt.Errorf("Expected resource map chunk, got %T", chunk)
	}

	for {
		chunk, err = decodeChunk(r, c)
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}
		c.chunks = append(c.chunks, chunk)
	}
}

func (c *xmlTree) encode() []byte {
	return encodeChunk(resXMLType, func(w binary.Writer) {
		// No custom header.
	}, func(w binary.Writer) {
		w.Data(c.strings.encode())
		w.Data(c.resourceMap.encode())
		for _, chunk := range c.chunks {
			w.Data(chunk.encode())
		}
	})
}

func (c *xmlTree) toXmlString() string {
	return c.xml(&xmlContext{
		strings:    c.strings,
		namespaces: map[string]string{},
		tab:        "  ",
	})
}

func (c *xmlTree) decodeString(r binary.Reader) stringPoolRef {
	idx := r.Uint32()
	if idx != missingString {
		return stringPoolRef{c.strings, idx}
	}
	return stringPoolRef{nil, idx}

}

// ensureAttributeMapsToResource finds a name mapping to the given resource id.
// If such a name does not exist, it is added to the string pool after the last
// string associated with a resource id, shifting all the strings after it. The
// resource map is updated to associate this string's position in the pool with
// the given resource id.
func (xml *xmlTree) ensureAttributeNameMapsToResource(resourceId uint32, attrName string) stringPoolRef {
	attrIdx, foundAttr := xml.resourceMap.indexOf(resourceId)
	if foundAttr {
		poolRef, found := xml.strings.findFromStringPoolIndex(attrIdx)
		if found {
			if poolRef.get() != attrName {
				panic("Attribute found but name mismatch. Perhaps this is allowed?")
			}
			return poolRef
		} else {
			panic("Resource map or string pool broken.")
		}
	}

	insertIndex := len(xml.resourceMap.ids)
	xml.resourceMap.ids = append(xml.resourceMap.ids, resourceId)
	return xml.strings.insertStringAtIndex(attrName, insertIndex)
}
