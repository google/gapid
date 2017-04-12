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
	"context"
	"crypto/sha1"
	"hash"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
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
func Hash(ctx context.Context, val interface{}) (id.ID, error) {
	msg, err := toProto(ctx, val)
	if err != nil {
		return id.ID{}, nil
	}
	return hashProto(val, msg)
}

func hashProto(val interface{}, msg proto.Message) (id.ID, error) {
	h := sha1Pool.Get().(hash.Hash)
	h.Reset()
	h.Write([]byte(reflect.TypeOf(val).String()))

	buf := protobufPool.Get().(*proto.Buffer)
	buf.Reset()
	err := buf.EncodeMessage(msg)
	h.Write(buf.Bytes())

	out := id.ID{}
	copy(out[:], h.Sum(nil))

	protobufPool.Put(buf)
	sha1Pool.Put(h)

	if err != nil {
		return id.ID{}, err
	}
	return out, nil
}
