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

package compiler

////////////////////////////////////////////////////////////////////////////////
// All types in this file need to match those in  gapil/runtime/cc/runtime.h  //
////////////////////////////////////////////////////////////////////////////////

//#include "gapil/runtime/cc/runtime.h"
import "C"

// ErrorCode is an error code returned (possibly in a <value, errcode> pair)
// by a command or subroutine.
type ErrorCode uint32

const (
	// ErrSuccess is the error code for a success.
	ErrSuccess = ErrorCode(C.GAPIL_ERR_SUCCESS)

	// ErrAborted is the error code for a command that called abort().
	ErrAborted = ErrorCode(C.GAPIL_ERR_ABORTED)
)

// Map constants
const (
	mapElementEmpty   = (uint64)(C.GAPIL_MAP_ELEMENT_EMPTY)
	mapElementFull    = (uint64)(C.GAPIL_MAP_ELEMENT_FULL)
	mapElementUsed    = (uint64)(C.GAPIL_MAP_ELEMENT_USED)
	mapGrowMultiplier = (uint64)(C.GAPIL_MAP_GROW_MULTIPLIER)
	minMapSize        = (uint64)(C.GAPIL_MIN_MAP_SIZE)
	mapMaxCapacity    = (float32)(C.GAPIL_MAP_MAX_CAPACITY)
)

func init() {
	if ((minMapSize & (minMapSize - 1)) != 0) ||
		((mapGrowMultiplier & (mapGrowMultiplier - 1)) != 0) {
		panic("Map size must be a power of 2")
	}
}

// Field names for the context_t runtime type.
const (
	ContextGlobals    = "globals"
	ContextArena      = "arena"
	ContextThread     = "thread"
	ContextCmdID      = "cmd_id"
	ContextCmdArgs    = "cmd_args"
	ContextNextPoolID = "next_pool_id"
)

// Field names for the slice_t runtime type.
const (
	SlicePool  = "pool"
	SliceRoot  = "root"
	SliceBase  = "base"
	SliceSize  = "size"
	SliceCount = "count"
)

// Field names for the gapil_rtti_t runtime type.
const (
	RTTIKind      = "kind"
	RTTIAPIIndex  = "api_index"
	RTTITypeIndex = "type_index"
	RTTITypeName  = "type_name"
	RTTIReference = "reference"
	RTTIRelease   = "release"
)

// Field names for the gapil_any_t runtime type.
const (
	AnyRefCount = "ref_count"
	AnyArena    = "arena"
	AnyRTTI     = "rtti"
	AnyValue    = "value"
)

// Field names for the gapil_msg_arg_t runtime type.
const (
	MsgArgName  = "name"
	MsgArgValue = "value"
)

// Field names for the gapil_msg_t runtime type.
const (
	MsgRefCount   = "ref_count"
	MsgArena      = "arena"
	MsgIdentifier = "identifier"
	MsgArgs       = "args"
)

// Field names for the pool_t runtime type.
const (
	PoolRefCount = "ref_count"
	PoolID       = "id"
	PoolSize     = "size"
	PoolBuffer   = "buffer"
)

// Field names for the map_t runtime type.
const (
	MapRefCount = "ref_count"
	MapArena    = "arena"
	MapCount    = "count"
	MapCapacity = "capacity"
	MapElements = "elements"
)

// Field names for the string_t runtime type.
const (
	StringRefCount = "ref_count"
	StringArena    = "arena"
	StringLength   = "length"
	StringData     = "data"
)

// Field names for the ref_t runtime type.
const (
	RefRefCount = "ref_count"
	RefArena    = "arena"
	RefValue    = "value"
)

// Field names for the buffer_t runtime type.
const (
	BufArena = "arena"
	BufData  = "data"
	BufCap   = "capacity"
	BufSize  = "size"
	BufAlign = "alignment"
)

// gapil_data_access enumerator values.
const (
	Read  = C.GAPIL_READ
	Write = C.GAPIL_WRITE
)

// gapil_kind enumerator values.
const (
	KindBool      = uint32(C.GAPIL_KIND_BOOL)
	KindU8        = uint32(C.GAPIL_KIND_U8)
	KindS8        = uint32(C.GAPIL_KIND_S8)
	KindU16       = uint32(C.GAPIL_KIND_U16)
	KindS16       = uint32(C.GAPIL_KIND_S16)
	KindF32       = uint32(C.GAPIL_KIND_F32)
	KindU32       = uint32(C.GAPIL_KIND_U32)
	KindS32       = uint32(C.GAPIL_KIND_S32)
	KindF64       = uint32(C.GAPIL_KIND_F64)
	KindU64       = uint32(C.GAPIL_KIND_U64)
	KindS64       = uint32(C.GAPIL_KIND_S64)
	KindInt       = uint32(C.GAPIL_KIND_INT)
	KindUint      = uint32(C.GAPIL_KIND_UINT)
	KindSize      = uint32(C.GAPIL_KIND_SIZE)
	KindChar      = uint32(C.GAPIL_KIND_CHAR)
	KindArray     = uint32(C.GAPIL_KIND_ARRAY)
	KindClass     = uint32(C.GAPIL_KIND_CLASS)
	KindEnum      = uint32(C.GAPIL_KIND_ENUM)
	KindMap       = uint32(C.GAPIL_KIND_MAP)
	KindPointer   = uint32(C.GAPIL_KIND_POINTER)
	KindReference = uint32(C.GAPIL_KIND_REFERENCE)
	KindSlice     = uint32(C.GAPIL_KIND_SLICE)
	KindString    = uint32(C.GAPIL_KIND_STRING)
)
