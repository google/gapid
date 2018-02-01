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

type ErrorCode uint32

const (
	ErrSuccess = ErrorCode(C.GAPIL_ERR_SUCCESS)
	ErrAborted = ErrorCode(C.GAPIL_ERR_ABORTED)

	mapElementEmpty   = (uint64)(C.GAPIL_MAP_ELEMENT_EMPTY)
	mapElementFull    = (uint64)(C.GAPIL_MAP_ELEMENT_FULL)
	mapElementUsed    = (uint64)(C.GAPIL_MAP_ELEMENT_USED)
	mapGrowMultiplier = (uint64)(C.GAPIL_MAP_GROW_MULTIPLIER)
	minMapSize        = (uint64)(C.GAPIL_MIN_MAP_SIZE)
	mapMaxCapacity    = (float32)(C.GAPIL_MAP_MAX_CAPACITY)

	contextLocation    = "location"
	contextGlobals     = "globals"
	contextAppPool     = "app_pool"
	contextEmptyString = "empty_string"
	slicePool          = "pool"
	sliceRoot          = "root"
	sliceBase          = "base"
	sliceSize          = "size"
	poolRefCount       = "ref_count"
	poolBuffer         = "buffer"
	mapRefCount        = "ref_count"
	mapCount           = "count"
	mapCapacity        = "capacity"
	mapElements        = "elements"
	stringRefCount     = "ref_count"
	stringLength       = "length"
	stringData         = "data"
	refRefCount        = "ref_count"
	refValue           = "value"
)
