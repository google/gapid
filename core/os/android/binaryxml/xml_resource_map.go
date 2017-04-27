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

type xmlResourceMap struct {
	rootHolder
	ids []uint32
}

func (c *xmlResourceMap) encode() []byte {
	return encodeChunk(resXMLResourceMapType, func(w binary.Writer) {
		// No custom header.
	}, func(w binary.Writer) {
		for _, id := range c.ids {
			w.Uint32(id)
		}
	})
}

func (c *xmlResourceMap) decode(header, data []byte) error {
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	id := r.Uint32()
	for r.Error() == nil {
		c.ids = append(c.ids, id)
		id = r.Uint32()
	}
	return nil
}

func (xmlResourceMap) xml(*xmlContext) string { return "" }

func (c *xmlResourceMap) indexOf(attr uint32) (uint32, bool) {
	for i, id := range c.ids {
		if id == attr {
			return uint32(i), true
		}
	}
	return missingString, false
}
