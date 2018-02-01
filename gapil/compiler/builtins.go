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
// All types in this file need to match those in gapil/compiler/cc/builtins.h //
////////////////////////////////////////////////////////////////////////////////

//#include "gapil/compiler/cc/builtins.h"
import "C"

type ErrorCode uint32

const (
	ErrSuccess = ErrorCode(C.ERR_SUCCESS)
	ErrAborted = ErrorCode(C.ERR_ABORTED)

	mapElementEmpty   = (uint64)(C.mapElementEmpty)
	mapElementFull    = (uint64)(C.mapElementFull)
	mapElementUsed    = (uint64)(C.mapElementUsed)
	mapGrowMultiplier = (uint64)(C.mapGrowMultiplier)
	minMapSize        = (uint64)(C.minMapSize)
	mapMaxCapacity    = (float32)(C.mapMaxCapacity)

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
