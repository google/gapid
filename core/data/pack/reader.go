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
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/third_party/src/github.com/pkg/errors"
)

// ErrUnknownType is the error returned by Reader.Unmarshal() when it
// encounters an unknown proto type.
type ErrUnknownType struct{ TypeName string }

func (e ErrUnknownType) Error() string { return fmt.Sprintf("Unknown proto type '%s'", e.TypeName) }

// Read reads the pack file from the supplied stream.
// This function will read the header from the stream, adjusting it's position.
// It may read extra bytes from the stream into an internal buffer.
func Read(ctx context.Context, from io.Reader, events Events, forceDynamic bool) error {
	r := &reader{
		types:  newTypes(forceDynamic),
		from:   from,
		buf:    make([]byte, 0, initalBufferSize),
		events: events,
	}
	r.pb = proto.NewBuffer(r.buf)
	if err := r.readMagic(); err != nil {
		return err
	}
	if _, err := r.readHeader(); err != nil {
		return err
	}
	for !task.Stopped(ctx) {
		if err := r.unmarshal(ctx); err != nil {
			cause := errors.Cause(err)
			if cause == io.EOF || cause == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
	}
	return task.StopReason(ctx)
}

// reader is the type for a pack file reader.
// They should only be constructed by NewReader.
type reader struct {
	types     *types
	events    Events
	id        uint64
	buf       []byte
	bufOffset int
	pb        *proto.Buffer
	from      io.Reader
}

func (r *reader) unmarshal(ctx context.Context) error {
	if err := r.readChunk(); err != nil {
		return err
	}
	tag, err := r.pb.DecodeZigzag64()
	if err != nil {
		return err
	}

	switch tag {
	case tagGroupFinalizer:
		idx, err := r.pb.DecodeVarint()
		if err != nil {
			return err
		}
		id := r.id - idx
		if err := r.events.EndGroup(ctx, id); err != nil {
			return err
		}

	case tagDeclareType:
		name, err := r.pb.DecodeStringBytes()
		if err != nil {
			return err
		}
		desc := &descriptor.DescriptorProto{}
		if err = r.pb.Unmarshal(desc); err != nil {
			return err
		}
		if _, ok := r.types.addNameAndDesc(name, desc); !ok {
			return ErrUnknownType{name}
		}
		return nil

	default:
		tyIdx, isGroup := tyIdxAndGroupFromTag(tag)
		if tyIdx >= r.types.count() {
			return fmt.Errorf("Unknown type index: %v. Type count: %v. Tag: %v", tyIdx, r.types.count(), tag)
		}
		ty := *r.types.entries[tyIdx]
		parentIdx, err := r.pb.DecodeVarint()
		if err != nil {
			return err
		}
		msg := ty.create()
		if err := r.pb.Unmarshal(msg); err != nil {
			return err
		}
		if parentIdx == 0 {
			if isGroup {
				err = r.events.BeginGroup(ctx, msg, r.id)
			} else {
				err = r.events.Object(ctx, msg)
			}
		} else {
			parentID := r.id - parentIdx
			if isGroup {
				err = r.events.BeginChildGroup(ctx, msg, r.id, parentID)
			} else {
				err = r.events.ChildObject(ctx, msg, parentID)
			}
		}
		if err != nil {
			return err
		}
		if isGroup {
			r.id++
		}
	}
	return nil
}

func (r *reader) readMagic() error {
	if err := r.readN(len(magicBytes)); err != nil {
		return err
	}
	if string(r.pb.Bytes()) != Magic {
		return ErrIncorrectMagic
	}
	return nil
}

func (r *reader) readHeader() (*Header, error) {
	if err := r.readChunk(); err != nil {
		return nil, err
	}
	header := &Header{}
	if err := r.pb.Unmarshal(header); err != nil {
		return nil, err
	}
	if got := header.GetVersion(); got.LessThan(MinVersion) || got.GreaterThan(MaxVersion) {
		return header, ErrUnsupportedVersion{*got}
	}
	return header, nil
}

func (r *reader) readChunk() error {
	// Make sure we have enough bytes for the maxiumum a varint could be, but don't
	// fail if the eof is within that range
	if err := r.readN(maxVarintSize); err != nil {
		if err != io.ErrUnexpectedEOF && err != io.EOF {
			return err
		}
	}
	data := r.pb.Bytes()
	size, n := proto.DecodeVarint(data)
	r.bufOffset -= len(data) - n
	if n == 0 {
		return io.EOF
	}
	return r.readN(int(size))
}

// readN makes sure there is size bytes available in the buffer if possible
func (r *reader) readN(size int) error {
	remains := r.buf[r.bufOffset:]
	extra := size - len(remains)
	if extra <= 0 {
		// We have all the data we need, just reslice
		r.pb.SetBuf(remains[:size])
		r.bufOffset += size
		return nil
	}
	// We need more data first
	if size > cap(r.buf) {
		// Our buffer is not big enough, so we need a new one
		// We over size the array for more efficient growth
		r.buf = make([]byte, size+size/4)
	} else {
		// set our buffer back to cap before the read
		r.buf = r.buf[:cap(r.buf)]
	}
	// Copy any existing data to the start of the buffer
	copy(r.buf, remains)
	// Read at least the extra bytes we need, but possibly more
	n, err := io.ReadAtLeast(r.from, r.buf[len(remains):], extra)
	// Slice back down to the amount we actually got
	r.buf = r.buf[:len(remains)+n]
	if size > len(r.buf) {
		size = len(r.buf)
	}
	r.pb.SetBuf(r.buf[:size])
	r.bufOffset = size
	return err
}
