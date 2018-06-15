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

package encoder

import "github.com/google/gapid/core/codegen"

//#define QUOTE(x) #x
//#define DECL_GAPIL_ENCODER_CB(RETURN, NAME, ...) \
//	const char* NAME##_sig = QUOTE(RETURN NAME(__VA_ARGS__));
//#include "gapil/runtime/cc/encoder.h"
import "C"

// callbacks are the runtime functions used to do the encoding.
type callbacks struct {
	encodeType    *codegen.Function
	encodeObject  *codegen.Function
	encodeBackref *codegen.Function
	sliceEncoded  *codegen.Function
}

func (e *encoder) parseCallbacks() {
	e.callbacks.encodeType = e.M.ParseFunctionSignature(C.GoString(C.gapil_encode_type_sig))
	e.callbacks.encodeObject = e.M.ParseFunctionSignature(C.GoString(C.gapil_encode_object_sig))
	e.callbacks.encodeBackref = e.M.ParseFunctionSignature(C.GoString(C.gapil_encode_backref_sig))
	e.callbacks.sliceEncoded = e.M.ParseFunctionSignature(C.GoString(C.gapil_slice_encoded_sig))
}
