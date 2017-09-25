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
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/third_party/src/github.com/pkg/errors"
)

// ErrUnknownType is the error returned by Reader.Unmarshal() when it
// encounters an unknown proto type.
type ErrUnknownType struct{ TypeName string }

func (e ErrUnknownType) Error() string { return fmt.Sprintf("Unknown proto type '%s'", e.TypeName) }

// Read reads the pack file from the supplied stream.
// This function will read the header from the stream, adjusting it's position.
// It may read extra bytes from the stream into an internal buffer.
func Read(ctx context.Context, from io.Reader, events Events) error {
	r := &reader{
		types:  newTypes(),
		from:   from,
		buf:    make([]byte, 0, initalBufferSize),
		events: events,
	}
	r.pb = proto.NewBuffer(r.buf)
	if major, _, err := r.readHeader(); err != nil {
		return err
	} else if !(MinVersion <= major && major <= MaxVersion) {
		return ErrUnsupportedVersion{Version: major}
	}
	for ; !task.Stopped(ctx); r.id++ {
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

func (r *reader) unmarshal(ctx context.Context) (err error) {
	// Protobuf library returns zig-zag encoded integers as uint64.
	var rawSize, rawParent, rawType uint64
	if rawSize, err = r.readChunk(); err != nil {
		return err
	}

	if int32(rawSize) < 0 {
		name, err := r.pb.DecodeStringBytes()
		if err != nil {
			return err
		}
		desc := &descriptor.DescriptorProto{}
		if err = r.pb.Unmarshal(desc); err != nil {
			return err
		}
		r.types.add(name, desc)
		return nil
	}

	// Read first two fields of object instance. If missing, they are implicitly set to 0.
	if rawParent, err = r.pb.DecodeZigzag64(); err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	if rawType, err = r.pb.DecodeZigzag64(); err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	hasParent, parent := int(rawParent) < 0, uint64(sint.Abs(int(rawParent)))
	hasChildren, tyIdx := int(rawType) < 0, uint64(sint.Abs(int(rawType)))

	if tyIdx == 0 { // Null-terminator
		if hasParent {
			if err := r.events.EndGroup(ctx, r.id-parent); err != nil {
				return err
			}
		}
	} else { // New object instance
		if tyIdx >= r.types.count() {
			return fmt.Errorf("Unknown type index: %v. Type count: %v.", tyIdx, r.types.count())
		}
		ty := *r.types.entries[tyIdx]
		msg := ty.create()
		if err := r.pb.Unmarshal(msg); err != nil {
			return err
		}
		if !hasParent {
			if hasChildren {
				err = r.events.BeginGroup(ctx, msg, r.id)
			} else {
				err = r.events.Object(ctx, msg)
			}
		} else {
			parentID := r.id - parent
			if hasChildren {
				err = r.events.BeginChildGroup(ctx, msg, r.id, parentID)
			} else {
				err = r.events.ChildObject(ctx, msg, parentID)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *reader) readHeader() (major int, minor int, err error) {
	if err := r.readN(16); err != nil {
		return 0, 0, err
	}
	str := string(r.pb.Bytes())
	if str[0:9] == "protopack" {
		return 1, 0, nil
	}
	if str[0:11] == "ProtoPack\r\n" &&
		str[11] >= '0' &&
		str[12] == '.' &&
		str[13] >= '0' &&
		str[14] == '\n' &&
		str[15] == 0 {
		return int(str[11] - '0'), int(str[13] - '0'), nil
	}
	return 0, 0, ErrIncorrectMagic
}

func (r *reader) readChunk() (size uint64, err error) {
	// Make sure we have enough bytes for the maxiumum a varint could be, but don't
	// fail if the eof is within that range
	if err := r.readN(maxVarintSize); err != nil {
		if err != io.ErrUnexpectedEOF && err != io.EOF {
			return 0, err
		}
	}
	data := r.pb.Bytes()
	size, n := proto.DecodeVarint(data)
	r.bufOffset -= len(data) - n
	if n == 0 || size == 0 {
		return 0, io.EOF
	}
	size = (size >> 1) ^ uint64((int64(size&1)<<63)>>63) // Decode zig-zag encoding
	return size, r.readN(sint.Abs(int(size)))
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
