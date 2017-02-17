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

package database

import (
	"crypto/sha1"
	"hash"
	"io"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/vle"
)

var (
	sha1Pool     = sync.Pool{New: func() interface{} { return sha1.New() }}
	protobufPool = sync.Pool{New: func() interface{} { return &proto.Buffer{} }}
)

// Hash returns a unique id.ID based on the contents of the object.
// Two objects of identical content will return the same ID, and the
// probability of two objects with different content generating the same ID
// will be ignorable.
// Objects with a graph structure are allowed.
// Only members that would be encoded using a binary.Encoder are considered.
func Hash(v interface{}) (id.ID, error) {
	id := id.ID{}

	h := sha1Pool.Get().(hash.Hash)
	h.Reset()

	var err error
	switch v := v.(type) {
	case proto.Message:
		h.Write([]byte(reflect.TypeOf(v).String()))
		buf := protobufPool.Get().(*proto.Buffer)
		buf.Reset()
		err = buf.EncodeMessage(v)
		h.Write(buf.Bytes())
		protobufPool.Put(buf)
	case WriterForHash:
		err = v.WriteForHash(h)
	default:
		var o binary.Object
		if o, err = binary.Box(v); err != nil {
			sha1Pool.Put(h)
			return id, err
		}
		e := cyclic.Encoder(vle.Writer(h))
		e.Object(o)
		err = e.Error()
	}
	copy(id[:], h.Sum(nil))
	sha1Pool.Put(h)
	return id, err
}

// WriterForHash can be implemented by classes which can trivially
// serialize themselves for hashing purposes. The intent is for
// this serialization to be faster than going through boxing and
// cyclic encoding, for objects that are guaranteed to be simple.
type WriterForHash interface {
	// WriteForHash serializes the object to the given Writer.
	// Two objects which would fail a reflect.DeepEqual() must
	// output different serialized representations.
	WriteForHash(w io.Writer) error
}
