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

package archive

/*
#include <stdlib.h>
#include "core/cc/archive.h"
*/
import "C"
import "unsafe"

// Archive contains assets.
type Archive = *C.archive

// New creates an archive.
func New(name string) Archive {
	cstr := C.CString(name)
	defer C.free(unsafe.Pointer(cstr))
	return C.archive_create(cstr)
}

// Dispose flushes and closes the underlying archive.
func (a Archive) Dispose() {
	C.archive_destroy(a)
}

// Write writes key-value pair to the archive.
func (a Archive) Write(key string, value []byte) bool {
	cstr := C.CString(key)
	defer C.free(unsafe.Pointer(cstr))
	csize := C.size_t(len(value))
	return C.archive_write(a, cstr, unsafe.Pointer(&value[0]), csize) != 0
}
