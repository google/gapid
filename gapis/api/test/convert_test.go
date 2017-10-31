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

package test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
)

func TestReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	o := &TestObject{Value: 42}
	m := NewU32ːTestObjectᵐ().
		Add(4, TestObject{Value: 40}).
		Add(5, TestObject{Value: 50})
	oldExtra := &TestExtra{
		Data: U8ˢ{
			root:  0x1000,
			base:  0x1000,
			count: 42,
			pool:  memory.PoolID(1),
		},
		Object: TestObject{Value: 10},
		ObjectArray: TestObjectː2ᵃ{
			TestObject{Value: 20},
			TestObject{Value: 30},
		},
		RefObject:      o,
		RefObjectAlias: o,
		NilRefObject:   nil,
		Entries:        m,
		EntriesAlias:   m,
		NilMap:         U32ːTestObjectᵐ{}, // Nil map is bad practice, but handle it correctly.
		RefEntries: NewU32ːTestObjectʳᵐ().
			Add(0, o).
			Add(6, &TestObject{Value: 60}).
			Add(7, &TestObject{Value: 70}).
			Add(9, nil),
		Strings: NewStringːu32ᵐ().
			Add("one", 1).
			Add("two", 2).
			Add("three", 3),
		BoolMap: NewU32ːboolᵐ().
			Add(0, false).
			Add(1, true),
	}

	msg, err := protoconv.ToProto(ctx, oldExtra)
	assert.For("ToProto").ThatError(err).Succeeded()
	log.I(ctx, "ProtoMessage: %v", proto.MarshalTextString(msg))
	e, err := protoconv.ToObject(ctx, msg)
	assert.For("ToObject").ThatError(err).Succeeded()
	newExtra := e.(*TestExtra)
	assert.For("Deserialized").TestDeepEqual(newExtra, oldExtra)
	assert.For("Object ref").That(newExtra.RefObject).Equals(newExtra.RefObjectAlias)
	assert.For("Map ref").That(newExtra.Entries).Equals(newExtra.EntriesAlias)
}
