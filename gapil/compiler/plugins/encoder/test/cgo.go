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

package encodertest

//#include "core/memory/arena/cc/arena.h"
//#include "gapil/compiler/plugins/encoder/test/test.h"
import "C"

import (
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/google/gapid/gapil/compiler/plugins/encoder/test/encoder_pb"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

type callbacks []interface{}

type encoder struct {
	callbacks callbacks
	types     map[unsafe.Pointer]int64
	backrefs  map[unsafe.Pointer]int64
}

var encoders = map[*C.context]*encoder{}

//export gapil_encode_type
func gapil_encode_type(ctx *C.context, name *C.uint8_t, descSize C.uint32_t, descPtr unsafe.Pointer) C.int64_t {
	desc := &descriptor.DescriptorProto{}
	err := proto.Unmarshal(C.GoBytes(descPtr, (C.int)(descSize)), desc)
	if err != nil {
		panic(err)
	}

	e := encoders[ctx]
	e.callbacks = append(e.callbacks, cbEncodeType{
		Name: C.GoString((*C.char)((unsafe.Pointer)(name))),
		Desc: desc,
	})

	id, ok := e.types[descPtr]
	if !ok {
		id = int64(len(e.types))
		e.types[descPtr] = -id
	}
	return (C.int64_t)(id)
}

//export gapil_encode_object
func gapil_encode_object(ctx *C.context, isGroup uint8, ty uint32, dataSize uint32, dataPtr unsafe.Pointer) unsafe.Pointer {
	e := encoders[ctx]
	e.callbacks = append(e.callbacks, cbEncodeObject{
		IsGroup: isGroup != 0,
		Type:    ty,
		Data:    C.GoBytes(dataPtr, (C.int)(dataSize)),
	})
	return nil
}

//export gapil_slice_encoded
func gapil_slice_encoded(ctx *C.context, s *C.slice) {
	e := encoders[ctx]
	e.callbacks = append(e.callbacks, cbSliceEncoded{Size: int(s.size)})
}

//export gapil_encode_backref
func gapil_encode_backref(ctx *C.context, object unsafe.Pointer) C.int64_t {
	e := encoders[ctx]
	id, ok := e.backrefs[object]
	if !ok {
		id = int64(len(e.backrefs))
		e.backrefs[object] = -id
	}
	e.callbacks = append(e.callbacks, cbBackref{id})
	return (C.int64_t)(id)
}

type cbEncodeType struct {
	Name string
	Desc *descriptor.DescriptorProto
}

type cbEncodeObject struct {
	IsGroup bool
	Type    uint32
	Data    []byte
}

type cbSliceEncoded struct {
	Size int
}

type cbBackref struct {
	ID int64
}

func withEncoder(f func(ctx *C.context)) callbacks {
	a := C.arena_create()
	ctx := C.create_context(a)
	defer func() {
		C.destroy_context(ctx)
		C.arena_destroy(a)
	}()

	e := encoder{
		types:    map[unsafe.Pointer]int64{nil: 0},
		backrefs: map[unsafe.Pointer]int64{nil: 0},
	}
	encoders[ctx] = &e

	f(ctx)

	delete(encoders, ctx)
	return e.callbacks
}

func encodeCmdInts(cmd *pb.CmdInts, isGroup bool) callbacks {
	s := C.cmd_ints{
		thread: (C.uint64_t)(cmd.Thread),
		a:      (C.uint8_t)(cmd.A),
		b:      (C.int8_t)(cmd.B),
		c:      (C.uint16_t)(cmd.C),
		d:      (C.int16_t)(cmd.D),
		e:      (C.uint32_t)(cmd.E),
		f:      (C.int32_t)(cmd.F),
		g:      (C.uint64_t)(cmd.G),
		h:      (C.int64_t)(cmd.H),
	}

	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_ints__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeCmdIntsCall(call *pb.CmdIntsCall, isGroup bool) callbacks {
	s := C.cmd_intsCall{result: (C.int64_t)(call.Result)}

	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_intsCall__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeCmdFloats(cmd *pb.CmdFloats, isGroup bool) callbacks {
	s := C.cmd_floats{
		thread: (C.uint64_t)(cmd.Thread),
		a:      (C.float)(cmd.A),
		b:      (C.double)(cmd.B),
	}
	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_floats__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeCmdEnums(cmd *pb.CmdEnums, isGroup bool) callbacks {
	s := C.cmd_enums{
		thread: (C.uint64_t)(cmd.Thread),
		e:      (C.uint32_t)(cmd.E),
		e_s64:  (C.int64_t)(cmd.ES64),
	}
	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_enums__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeCmdArrays(cmd *pb.CmdArrays, isGroup bool) callbacks {
	s := C.cmd_arrays{
		thread: (C.uint64_t)(cmd.Thread),
		a:      [1]C.uint8_t{(C.uint8_t)(cmd.A[0])},
		b:      [2]C.int32_t{(C.int32_t)(cmd.B[0]), (C.int32_t)(cmd.B[1])},
		c:      [3]C.float{(C.float)(cmd.C[0]), (C.float)(cmd.C[1]), (C.float)(cmd.C[2])},
	}
	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_arrays__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeCmdPointers(cmd *pb.CmdPointers, isGroup bool) callbacks {
	s := C.cmd_pointers{
		thread: (C.uint64_t)(cmd.Thread),
		a:      (*C.uint8_t)((unsafe.Pointer)(uintptr(cmd.A))),
		b:      (*C.int32_t)((unsafe.Pointer)(uintptr(cmd.B))),
		c:      (*C.float)((unsafe.Pointer)(uintptr(cmd.C))),
	}
	return withEncoder(func(ctx *C.context) {
		C.cmd__cmd_pointers__encode(&s, ctx, toUint8(isGroup))
	})
}

func convBasicTypes(class *pb.BasicTypes, out *C.basic_types) (dispose func()) {
	arena := C.arena_create()

	n := C.gapil_make_string(arena, (C.uint64_t)(len(class.N)), (unsafe.Pointer)(C.CString(class.N)))
	*out = C.basic_types{
		a: (C.uint8_t)(class.A),
		b: (C.int8_t)(class.B),
		c: (C.uint16_t)(class.C),
		d: (C.int16_t)(class.D),
		e: (C.float)(class.E),
		f: (C.uint32_t)(class.F),
		g: (C.int32_t)(class.G),
		h: (C.double)(class.H),
		i: (C.uint64_t)(class.I),
		j: (C.int64_t)(class.J),
		k: (C.uint8_t)(class.K),
		l: (C.uint32_t)(class.L),
		m: (*C.uint32_t)((unsafe.Pointer)(uintptr(class.M))),
		n: n,
	}
	return func() {
		C.arena_destroy(arena)
	}
}

func convInnerClass(class *pb.InnerClass, out *C.inner_class) (dispose func()) {
	return convBasicTypes(class.A, &out.a)
}

func convNestedClasses(class *pb.NestedClasses, out *C.nested_classes) (dispose func()) {
	return convInnerClass(class.A, &out.a)
}

func encodeBasicTypes(class *pb.BasicTypes, isGroup bool) callbacks {
	s := C.basic_types{}

	dispose := convBasicTypes(class, &s)
	defer dispose()

	return withEncoder(func(ctx *C.context) {
		C.basic_types__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeNestedClasses(class *pb.NestedClasses, isGroup bool) callbacks {
	s := C.nested_classes{}

	dispose := convNestedClasses(class, &s)
	defer dispose()

	return withEncoder(func(ctx *C.context) {
		C.nested_classes__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeMapTypes(maps *pb.MapTypes, isGroup bool) callbacks {
	arena := C.arena_create()
	defer C.arena_destroy(arena)

	s := C.map_types{}
	C.create_map_u32(arena, &s.a)
	C.create_map_string(arena, &s.b)

	for i := range maps.A.Keys {
		C.insert_map_u32(s.a, (C.uint32_t)(maps.A.Keys[i]), (C.uint32_t)(maps.A.Values[i]))
	}
	for i := range maps.B.Keys {
		C.insert_map_string(s.b, (C.CString)(maps.B.Keys[i]), (C.CString)(maps.B.Values[i]))
	}

	s.c = s.a
	s.d = s.b

	return withEncoder(func(ctx *C.context) {
		C.map_types__encode(&s, ctx, toUint8(isGroup))
	})
}

func encodeRefTypes(refs *pb.RefTypes, isGroup bool) callbacks {
	arena := C.arena_create()
	defer C.arena_destroy(arena)

	s := C.ref_types{}

	a := C.create_basic_types_ref(arena, &s.a)
	b := C.create_inner_class_ref(arena, &s.b)

	disposeA := convBasicTypes(refs.A.Value, a)
	defer disposeA()

	disposeB := convInnerClass(refs.B.Value, b)
	defer disposeB()

	s.c = s.a
	s.d = s.b

	return withEncoder(func(ctx *C.context) {
		C.ref_types__encode(&s, ctx, toUint8(isGroup))
	})
}

func convSlice(in *memory_pb.Slice, out *C.slice) {
	out.root = C.uint64_t(in.Root)
	out.base = C.uint64_t(in.Base)
	out.size = C.uint64_t(in.Size)
	out.count = C.uint64_t(in.Count)
	out.pool = nil
}

func encodeSliceTypes(slices *pb.SliceTypes, isGroup bool) callbacks {
	s := C.slice_types{}

	convSlice(slices.A, &s.a)
	convSlice(slices.B, &s.b)
	convSlice(slices.C, &s.c)

	return withEncoder(func(ctx *C.context) {
		C.slice_types__encode(&s, ctx, toUint8(isGroup))
	})
}

func toUint8(b bool) C.uint8_t {
	if b {
		return 1
	}
	return 0
}
