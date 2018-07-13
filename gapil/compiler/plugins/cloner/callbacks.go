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

package cloner

import "github.com/google/gapid/core/codegen"

//#define QUOTE(x) #x
//#define DECL_GAPIL_CLONER_CB(RETURN, NAME, ...) \
//	const char* NAME##_sig = QUOTE(RETURN NAME(__VA_ARGS__));
//#include "gapil/runtime/cc/cloner/cloner.h"
import "C"

// callbacks are the runtime functions used to do the cloing.
type callbacks struct {
	createCloneTracker  *codegen.Function
	destroyCloneTracker *codegen.Function
	cloneTrackerLookup  *codegen.Function
	cloneTrackerTrack   *codegen.Function
}

func (c *cloner) parseCallbacks() {
	c.callbacks.createCloneTracker = c.M.ParseFunctionSignature(C.GoString(C.gapil_create_clone_tracker_sig))
	c.callbacks.destroyCloneTracker = c.M.ParseFunctionSignature(C.GoString(C.gapil_destroy_clone_tracker_sig))
	c.callbacks.cloneTrackerLookup = c.M.ParseFunctionSignature(C.GoString(C.gapil_clone_tracker_lookup_sig))
	c.callbacks.cloneTrackerTrack = c.M.ParseFunctionSignature(C.GoString(C.gapil_clone_tracker_track_sig))
}
