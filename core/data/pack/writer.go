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

package pack

import (
	"io"

	"github.com/golang/protobuf/proto"
)

type (
	// Writer is the type for a pack file writer.
	// They should only be constructed by NewWriter.
	Writer struct {
		// Types is the set of registered types encoded through this writer.
		Types   *Types
		buf     *proto.Buffer
		sizebuf *proto.Buffer
		to      io.Writer
	}
)

// NewWriter constructs and returns a new Writer that writes to the supplied
// output stream.
// This method will write the packfile magic and header to the underlying
// stream.
func NewWriter(to io.Writer) (*Writer, error) {
	w := &Writer{
		Types:   NewTypes(),
		buf:     proto.NewBuffer(make([]byte, 0, initalBufferSize)),
		sizebuf: proto.NewBuffer(make([]byte, 0, maxVarintSize)),
		to:      to,
	}
	if err := w.writeMagic(); err != nil {
		return nil, err
	}
	header := &Header{Version: &version}
	if err := w.writeHeader(header); err != nil {
		return nil, err
	}
	return w, nil
}

// Marshal writes a new object to the packfile, preceding it with a
// type entry if needed.
func (w *Writer) Marshal(msg proto.Message) error {
	entry, added := w.Types.AddMessage(msg)
	if added {
		if err := w.writeType(entry); err != nil {
			return err
		}
	}
	return w.writeSection(entry.Index, "", msg)
}

func (w *Writer) writeType(t Type) error {
	return w.writeSection(specialSection, t.Name, t.Descriptor)
}

func (w *Writer) writeMagic() error {
	_, err := w.to.Write(magicBytes)
	return err
}

func (w *Writer) writeHeader(h *Header) error {
	if err := w.buf.Marshal(h); err != nil {
		return err
	}
	return w.flushChunk()
}

func (w *Writer) writeSection(tag uint64, name string, msg proto.Message) error {
	if err := w.buf.EncodeVarint(tag); err != nil {
		return err
	}
	if name != "" {
		if err := w.buf.EncodeStringBytes(name); err != nil {
			return err
		}
	}
	if err := w.buf.Marshal(msg); err != nil {
		return err
	}
	return w.flushChunk()
}

func (w *Writer) flushChunk() error {
	size := len(w.buf.Bytes())
	if err := w.sizebuf.EncodeVarint(uint64(size)); err != nil {
		return err
	}
	_, err := w.to.Write(w.sizebuf.Bytes())
	w.sizebuf.Reset()
	if err != nil {
		return err
	}
	_, err = w.to.Write(w.buf.Bytes())
	w.buf.Reset()
	return err
}
