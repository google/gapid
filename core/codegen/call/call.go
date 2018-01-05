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

// Package call provides methods for invoking C function pointers.
package call

import "unsafe"

// void V(void* f) { ((void (*)())(f))(); }
// int  I(void* f) { return ((int (*)())(f))(); }
// int  III(void* f, int a, int b) { return ((int (*)(int, int))(f))(a, b); }
import "C"

// V invokes the function f that has the signature void().
func V(f unsafe.Pointer) { C.V(f) }

// I invokes the function f that has the signature int().
func I(f unsafe.Pointer) int { return (int)(C.I(f)) }

// III invokes the function f that has the signature int(int, int).
func III(f unsafe.Pointer, a, b int) int { return (int)(C.III(f, (C.int)(a), (C.int)(b))) }
