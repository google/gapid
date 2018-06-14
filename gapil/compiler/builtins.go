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

// Field names for the context_t runtime type.
const (
	ContextLocation   = "location"
	ContextGlobals    = "globals"
	ContextArena      = "arena"
	ContextCmdID      = "cmd_id"
	ContextNextPoolID = "next_pool_id"
	ContextArguments  = "arguments"
)

// Field names for the slice_t runtime type.
const (
	SlicePool  = "pool"
	SliceRoot  = "root"
	SliceBase  = "base"
	SliceSize  = "size"
	SliceCount = "count"
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
	BufData = "data"
	BufCap  = "capacity"
	BufSize = "size"
)
