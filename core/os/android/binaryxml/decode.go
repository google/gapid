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
	"context"
	"fmt"
	"io"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/pkg/errors"
)

// AOSP references:
// https://android.googlesource.com/platform/frameworks/base/+/master/tools/aapt2/XmlFlattener.cpp
// https://android.googlesource.com/platform/frameworks/base/+/master/include/androidfw/ResourceTypes.h

const (
	resNullType              = 0x0000
	resStringPoolType        = 0x0001
	resTableType             = 0x0002
	resXMLType               = 0x0003
	resXMLFirstChunkType     = 0x0100
	resXMLStartNamespaceType = 0x0100
	resXMLEndNamespaceType   = 0x0101
	resXMLStartElementType   = 0x0102
	resXMLEndElementType     = 0x0103
	resXMLCDataType          = 0x0104
	resXMLLastChunkType      = 0x017f
	resXMLResourceMapType    = 0x0180
	resTablePackageType      = 0x0200
	resTableTypeType         = 0x0201
	resTableTypeSpecType     = 0x0202
	resTableLibraryType      = 0x0203
)

const (
	beforeContextChange = 0x00
	afterContextChange  = 0x01
)

type chunkVisitor func(*xmlContext, chunk, int)

type contextChange interface {
	updateContext(*xmlContext)
}

// Decode decodes a binary Android XML file to a string.
func Decode(ctx context.Context, data []byte) (string, error) {
	xmlTree, err := decodeXmlTree(bytes.NewReader(data))
	if err != nil {
		return "", log.Err(ctx, err, "Decoding binary XML")
	}
	return xmlTree.toXmlString(), nil
}

type rootHolder struct {
	rootNode *xmlTree
}

func (rh *rootHolder) root() *xmlTree {
	return rh.rootNode
}

func (rh *rootHolder) setRoot(x *xmlTree) {
	rh.rootNode = x
}

type chunk interface {
	root() *xmlTree
	setRoot(x *xmlTree)

	decode(header, data []byte) error
	xml(*xmlContext) string
	encode() []byte
}

func decodeXmlTree(r io.Reader) (*xmlTree, error) {
	br := endian.Reader(r, device.LittleEndian)

	chunk, err := decodeChunk(br, &xmlTree{})
	if err != nil {
		return nil, err
	}

	tree, ok := chunk.(*xmlTree)
	if !ok {
		return nil, fmt.Errorf("Expected XML tree, found chunk type %T", chunk)
	}

	return tree, nil
}

func decodeChunk(r binary.Reader, x *xmlTree) (chunk, error) {
	ty := r.Uint16()
	if err := r.Error(); err != nil {
		return nil, err
	}
	headerSize := r.Uint16()
	if headerSize < 8 {
		return nil, fmt.Errorf("Unexpected header size %d", headerSize)
	}
	dataSize := r.Uint32()
	header := make([]byte, headerSize-8)
	data := make([]byte, dataSize-uint32(headerSize))
	r.Data(header)
	r.Data(data)
	var c chunk
	switch ty {
	case resXMLResourceMapType:
		c = &xmlResourceMap{}
	case resStringPoolType:
		c = &stringPool{}
	case resXMLCDataType:
		c = &xmlCData{}
	case resXMLEndElementType:
		c = &xmlEndElement{}
	case resXMLEndNamespaceType:
		c = &xmlEndNamespace{}
	case resXMLStartElementType:
		c = &xmlStartElement{}
	case resXMLStartNamespaceType:
		c = &xmlStartNamespace{}
	case resXMLType:
		c = x
	default:
		return nil, fmt.Errorf("Unknown chunk type 0x%x", ty)
	}
	c.setRoot(x)
	err := c.decode(header, data)
	if errors.Cause(err) == io.EOF {
		return nil, fmt.Errorf("Chunk type %T read past end of data", c)
	}
	return c, err
}

func decodeLength(r binary.Reader) uint32 {
	length := uint32(r.Uint16())
	if length&0x8000 != 0 {
		panic("UNTESTED CODE")
		length = (length << 16) | uint32(r.Uint16())
	}
	return length
}

func encodeLength(w binary.Writer, length uint32) {
	if length >= 0x8000 {
		panic("TODO: UNSUPPORTED")
	}
	w.Uint16(uint16(length))
}

// encodeChunk takes functions that output chunk-specific header and data to a writer, and then uses them to
// compute header and chunk sizes, as well as writing the whole chunk to a byte array, which is then returned.
func encodeChunk(chunkType uint16, headerf func(w binary.Writer), dataf func(w binary.Writer)) []byte {
	var headerBuffer bytes.Buffer
	headerf(endian.Writer(&headerBuffer, device.LittleEndian))
	headerBytes := headerBuffer.Bytes()

	var dataBuffer bytes.Buffer
	dataf(endian.Writer(&dataBuffer, device.LittleEndian))
	dataBytes := dataBuffer.Bytes()

	var chunkBuffer bytes.Buffer
	w := endian.Writer(&chunkBuffer, device.LittleEndian)
	w.Uint16(chunkType)
	w.Uint16(uint16(len(headerBytes) + 8))
	w.Uint32(uint32(len(headerBytes) + len(dataBytes) + 8))
	w.Data(headerBytes)
	w.Data(dataBytes)

	return chunkBuffer.Bytes()
}
