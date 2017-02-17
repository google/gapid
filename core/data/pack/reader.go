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
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type (
	// Reader is the type for a pack file reader.
	// They should only be constructed by NewReader.
	Reader struct {
		// Types is the set of registered types this reader will decode.
		Types *Types
		buf   []byte
		next  int
		pb    *proto.Buffer
		from  io.Reader
		total int
	}
)

// NewReader builds a pack file reader that gets it's data from the supplied
// stream.
// This function will read the header from the stream, adjusting it's position.
// It may read extra bytes from the stream into an internal buffer.
func NewReader(from io.Reader) (*Reader, error) {
	r := newReader(from)
	if err := r.readMagic(); err != nil {
		return nil, err
	}
	if _, err := r.readHeader(); err != nil {
		return nil, err
	}
	return r, nil
}

func newReader(from io.Reader) *Reader {
	r := &Reader{
		Types: NewTypes(),
		from:  from,
	}
	r.buf = make([]byte, 0, initalBufferSize)
	r.pb = proto.NewBuffer(r.buf)
	return r
}

// Unmarshal reads the next data section from the file, consuming any special
// sections on the way.
func (r *Reader) Unmarshal() (proto.Message, error) {
	for {
		tag, err := r.readSection()
		if err != nil {
			return nil, err
		}
		if tag != specialSection {
			typ := r.Types.Get(tag)
			msg := reflect.New(typ.Type).Interface().(proto.Message)
			if err := r.pb.Unmarshal(msg); err != nil {
				return nil, err
			}
			return msg, nil
		}
		if err := r.readType(); err != nil {
			return nil, err
		}
	}
}

func (r *Reader) readType() error {
	name, err := r.readSectionName()
	if err != nil {
		return err
	}
	d := &descriptor.DescriptorProto{}
	if err = r.pb.Unmarshal(d); err != nil {
		return err
	}
	t, _ := r.Types.AddName(name)
	if t.Descriptor != nil {
		// TODO: validate the descriptor matches
	} else {
		t.Descriptor = d
	}
	return nil
}

func (r *Reader) readMagic() error {
	if err := r.readN(len(magicBytes)); err != nil {
		return err
	}
	if string(r.pb.Bytes()) != Magic {
		return ErrIncorrectMagic
	}
	return nil
}

func (r *Reader) readHeader() (*Header, error) {
	if err := r.readChunk(); err != nil {
		return nil, err
	}
	header := &Header{}
	if err := r.pb.Unmarshal(header); err != nil {
		return nil, err
	}
	if header.GetVersion().GetMajor() != version.GetMajor() {
		return header, ErrUnknownVersion
	}
	return header, nil
}

func (r *Reader) readSection() (uint64, error) {
	if err := r.readChunk(); err != nil {
		return 0, err
	}
	return r.pb.DecodeVarint()
}

func (r *Reader) readSectionName() (string, error) {
	return r.pb.DecodeStringBytes()
}

func (r *Reader) readChunk() error {
	// Make sure we have enough bytes for the maxiumum a varint could be, but don't
	// fail if the eof is within that range
	if err := r.readN(maxVarintSize); err != nil {
		if err != io.ErrUnexpectedEOF && err != io.EOF {
			return err
		}
	}
	data := r.pb.Bytes()
	size, n := proto.DecodeVarint(data)
	r.next -= len(data) - n
	if n == 0 {
		return io.EOF
	}
	return r.readN(int(size))
}

// readN makes sure there is size bytes available in the buffer if possible
func (r *Reader) readN(size int) error {
	remains := r.buf[r.next:]
	extra := size - len(remains)
	if extra <= 0 {
		// We have all the data we need, just reslice
		r.pb.SetBuf(remains[:size])
		r.next += size
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
	r.next = size
	r.total += n
	return err
}
