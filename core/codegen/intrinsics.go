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

package codegen

// Memcpy performs an intrinsic memory copy of size bytes from src to dst.
// dst and src must be a void pointer (alias of u8*).
// size is the copy size in bytes. size must be of type uint32.
// align is the minimum pointer alignment of dst and src. align must of type
// uint32, or nil to represent no guaranteed alignment.
func (b *Builder) Memcpy(dst, src, size *Value) {
	isVolatile := b.Scalar(false)
	b.Call(b.m.memcpy, dst, src, size, isVolatile)
}
