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
	"context"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
)

// Writer is the type for a pack file writer.
// They should only be constructed by NewWriter.
type Writer struct {
	types   *types
	id      uint64
	buf     *proto.Buffer
	sizebuf *proto.Buffer
	to      io.Writer
}

// NewWriter constructs and returns a new Writer that writes to the supplied
// output stream.
// This method will write the packfile magic and header to the underlying
// stream.
func NewWriter(to io.Writer) (*Writer, error) {
	w := &Writer{
		types:   newTypes(false),
		buf:     proto.NewBuffer(make([]byte, 0, initalBufferSize)),
		sizebuf: proto.NewBuffer(make([]byte, 0, maxVarintSize)),
		to:      to,
	}
	if err := w.writeMagic(); err != nil {
		return nil, err
	}
	header := &Header{Version: version}
	if err := w.writeHeader(header); err != nil {
		return nil, err
	}
	return w, nil
}

// BeginGroup is called to start a new root group.
func (w *Writer) BeginGroup(ctx context.Context, msg proto.Message) (id uint64, err error) {
	return w.writeMessage(msg, true, nil)
}

// BeginChildGroup is called to start a new group as a child of the group with
// the parent identifier.
func (w *Writer) BeginChildGroup(ctx context.Context, msg proto.Message, parentID uint64) (id uint64, err error) {
	return w.writeMessage(msg, true, &parentID)
}

// EndGroup finalizes the group with the given identifier. It is illegal to
// attempt to add children to the group after this is called.
// EndGroup should be closed immediately after the last child has been added
// to the group.
func (w *Writer) EndGroup(ctx context.Context, id uint64) error {
	if err := w.buf.EncodeZigzag64(tagGroupFinalizer); err != nil {
		return err
	}
	if err := w.writeID(id); err != nil {
		return err
	}
	return w.flushChunk()
}

// Object is called to declare an object outside of any group.
func (w *Writer) Object(ctx context.Context, msg proto.Message) error {
	_, err := w.writeMessage(msg, false, nil)
	return err
}

// ChildObject is called to declare an object in the group with the given
// identifier.
func (w *Writer) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	_, err := w.writeMessage(msg, false, &parentID)
	return err
}

func (w *Writer) writeMessage(msg proto.Message, isGroup bool, parentID *uint64) (id uint64, err error) {
	id = w.id

	ty, added := w.types.addMessage(msg)
	if added {
		if err := w.writeType(ty); err != nil {
			return 0, err
		}
	}

	tag := uint64(tagFromTyIdxAndGroup(ty.index, isGroup))
	if err := w.buf.EncodeZigzag64(tag); err != nil {
		return 0, err
	}

	if parentID == nil {
		if err := w.buf.EncodeVarint(0); err != nil {
			return 0, err
		}
	} else {
		if err := w.writeID(*parentID); err != nil {
			return 0, err
		}
	}

	if err := w.buf.Marshal(msg); err != nil {
		return 0, err
	}

	if isGroup {
		w.id++
	}

	return id, w.flushChunk()
}

func (w *Writer) writeID(id uint64) error {
	if id >= w.id {
		return fmt.Errorf("Invalid parentID: %v", id)
	}
	return w.buf.EncodeVarint(w.id - id)
}

func (w *Writer) writeType(t ty) error {
	if err := w.buf.EncodeZigzag64(tagDeclareType); err != nil {
		return err
	}
	if err := w.buf.EncodeStringBytes(t.name); err != nil {
		return err
	}
	if err := w.buf.Marshal(t.desc); err != nil {
		return err
	}
	return w.flushChunk()
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
