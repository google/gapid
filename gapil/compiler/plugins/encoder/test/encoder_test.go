// Copyright (C) 2018 Google Inc.
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

// Package encodertest tests the encoder compiler plugin.
package encodertest

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory/memory_pb"

	pb "github.com/google/gapid/gapil/compiler/plugins/encoder/test/encoder_pb"
)

func checkCallbacks(ctx context.Context, name string, got, expected callbacks) {
	assert.For(ctx, "%v.callbacks", name).That(got).DeepEquals(expected)
	count := len(got)
	if count > len(expected) {
		count = len(expected)
	}
	for i := 0; i < count; i++ {
		got, expected := got[i], expected[i]
		if got, ok := got.(cbEncodeObject); ok {
			if expected, ok := expected.(cbEncodeObject); ok {
				assert.For(ctx, "%v[%v].object.Data", name, i).ThatSlice(got.Data).Equals(expected.Data)
			}
		}
	}
}

func TestCmdInts(t *testing.T) {
	ctx := log.Testing(t)

	params := pb.CmdInts{
		Thread: 0x12345678,
		A:      0xff,
		B:      -0x80,
		C:      0xffff,
		D:      -0x8000,
		E:      0xffffffff,
		F:      -0x80000000,
		G:      -1,
		H:      -0x8000000000000000,
	}
	call := pb.CmdIntsCall{
		Result: 0x80,
	}
	checkCallbacks(ctx, "params", encodeCmdInts(&params, true), callbacks{
		expectedType(&pb.CmdInts{}),
		cbEncodeObject{Type: 1, IsGroup: true, Data: encodeProto(&params)},
	})
	checkCallbacks(ctx, "call", encodeCmdIntsCall(&call, false), callbacks{
		expectedType(&pb.CmdIntsCall{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&call)},
	})
}

func TestCmdFloats(t *testing.T) {
	ctx := log.Testing(t)

	params := pb.CmdFloats{
		Thread: 0x10,
		A:      1234.5678,
		B:      123456789.987654321,
	}
	checkCallbacks(ctx, "params", encodeCmdFloats(&params, false), callbacks{
		expectedType(&pb.CmdFloats{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&params)},
	})
}

func TestCmdEnums(t *testing.T) {
	ctx := log.Testing(t)

	params := pb.CmdEnums{
		Thread: 0x10,
		E:      100,
		ES64:   -9223372036854775808,
	}
	checkCallbacks(ctx, "params", encodeCmdEnums(&params, false), callbacks{
		expectedType(&pb.CmdEnums{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&params)},
	})
}

func TestCmdArrays(t *testing.T) {
	ctx := log.Testing(t)

	params := pb.CmdArrays{
		Thread: 0x10,
		A:      []int64{1},
		B:      []int64{1, 2},
		C:      []float32{1, 2, 3},
	}
	checkCallbacks(ctx, "params", encodeCmdArrays(&params, false), callbacks{
		expectedType(&pb.CmdArrays{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&params)},
	})
}

func TestCmdPointers(t *testing.T) {
	ctx := log.Testing(t)

	params := pb.CmdPointers{
		Thread: 0x10,
		A:      0x12345678,
		B:      0xabcdef42,
		C:      0x0123456789abcdef,
	}
	checkCallbacks(ctx, "params", encodeCmdPointers(&params, false), callbacks{
		expectedType(&pb.CmdPointers{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&params)},
	})
}

func TestBasicTypes(t *testing.T) {
	ctx := log.Testing(t)

	class := pb.BasicTypes{
		A: 10,
		B: 20,
		C: 30,
		D: 40,
		E: 50,
		F: 60,
		G: 70,
		H: 80,
		I: 90,
		J: 100,
		K: 1,
		L: 0x10,
		M: 0x1234,
		N: "meow",
	}
	checkCallbacks(ctx, "basic_types", encodeBasicTypes(&class, false), callbacks{
		expectedType(&pb.BasicTypes{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&class)},
	})
}

func TestNestedClasses(t *testing.T) {
	ctx := log.Testing(t)

	basic := pb.BasicTypes{
		A: 10,
		E: 50,
		F: 60,
		H: 80,
		K: 1,
		N: "meow",
	}
	inner := pb.InnerClass{A: &basic}
	nested := pb.NestedClasses{A: &inner}
	checkCallbacks(ctx, "nested_classes", encodeNestedClasses(&nested, false), callbacks{
		expectedType(&pb.NestedClasses{}),
		expectedType(&pb.InnerClass{}),
		expectedType(&pb.BasicTypes{}),
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&nested)},
	})
}

func TestMapTypes(t *testing.T) {
	ctx := log.Testing(t)

	class := pb.MapTypes{
		A: &pb.Sint64ToSint64Map{
			ReferenceID: 1,
			Keys:        []int64{10, 20, 30},
			Values:      []int64{200, 100, 300},
		},
		B: &pb.StringToStringMap{
			ReferenceID: 2,
			Keys:        []string{"snake", "cat", "dog"},
			Values:      []string{"hiss", "meow", "woof"},
		},
		// C and D are copies of A and B
		C: &pb.Sint64ToSint64Map{ReferenceID: 1},
		D: &pb.StringToStringMap{ReferenceID: 2},
	}
	checkCallbacks(ctx, "map_types", encodeMapTypes(&class, false), callbacks{
		expectedType(&pb.MapTypes{}),
		expectedType(&pb.Sint64ToSint64Map{}),
		expectedType(&pb.StringToStringMap{}),
		cbBackref{ID: 1},
		cbBackref{ID: 2},
		cbBackref{ID: -1},
		cbBackref{ID: -2},
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&class)},
	})
}

func TestRefTypes(t *testing.T) {
	ctx := log.Testing(t)

	class := pb.RefTypes{
		A: &pb.BasicTypesRef{
			ReferenceID: 1,
			Value: &pb.BasicTypes{
				A: 10,
				E: 50,
				F: 60,
				H: 80,
				K: 1,
				N: "meow",
			},
		},
		B: &pb.InnerClassRef{
			ReferenceID: 2,
			Value: &pb.InnerClass{
				A: &pb.BasicTypes{
					A: 10,
					E: 50,
					F: 60,
					H: 80,
					K: 1,
					N: "meow",
				},
			},
		},
		// C and D are copies of A and B
		C: &pb.BasicTypesRef{ReferenceID: 1},
		D: &pb.InnerClassRef{ReferenceID: 2},
	}
	checkCallbacks(ctx, "ref_types", encodeRefTypes(&class, false), callbacks{
		expectedType(&pb.RefTypes{}),
		expectedType(&pb.BasicTypesRef{}),
		expectedType(&pb.BasicTypes{}),
		expectedType(&pb.InnerClassRef{}),
		expectedType(&pb.InnerClass{}),
		expectedType(&pb.BasicTypes{}),

		cbBackref{ID: 1},
		cbBackref{ID: 2},
		cbBackref{ID: -1},
		cbBackref{ID: -2},
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&class)},
	})
}

func TestSliceTypes(t *testing.T) {
	ctx := log.Testing(t)

	class := pb.SliceTypes{
		A: &memory_pb.Slice{
			Root:  0x1000,
			Base:  0x2000,
			Size:  0x10,
			Count: 0x10,
			Pool:  0,
		},
		B: &memory_pb.Slice{
			Root:  0x2000,
			Base:  0x3000,
			Size:  0x80,
			Count: 0x20,
			Pool:  0,
		},
		C: &memory_pb.Slice{
			Root:  0x3000,
			Base:  0x4000,
			Size:  0xc0,
			Count: 0x30,
			Pool:  0,
		},
	}
	checkCallbacks(ctx, "slice_types", encodeSliceTypes(&class, false), callbacks{
		expectedType(&pb.SliceTypes{}),
		expectedType(&memory_pb.Slice{}),
		cbSliceEncoded{int(class.A.Size)},
		cbSliceEncoded{int(class.B.Size)},
		cbSliceEncoded{int(class.C.Size)},
		cbEncodeObject{Type: 1, IsGroup: false, Data: encodeProto(&class)},
	})
}

func expectedType(m proto.Message) cbEncodeType {
	desc, err := protoutil.DescriptorOf(m.(protoutil.Described))
	if err != nil {
		panic(err)
	}
	return cbEncodeType{
		Name: proto.MessageName(m),
		Desc: desc,
	}
}

func encodeProto(m proto.Message) []byte {
	data, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	return data
}
