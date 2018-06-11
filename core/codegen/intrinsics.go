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
func (b *Builder) Memcpy(dst, src, size *Value) {
	isVolatile := b.Scalar(false)
	b.Call(b.m.memcpy, dst, src, size, isVolatile)
}

// Memset performs an intrinsic memory set to the given value.
// dst must be a void pointer (alias of u8*).
// val is value that is to be repeated size times into dst. val must be of type
// uint8.
// size is the copy size in bytes. size must be of type uint32.
func (b *Builder) Memset(dst, val, size *Value) {
	isVolatile := b.Scalar(false)
	b.Call(b.m.memset, dst, val, size, isVolatile)
}

// Memzero performs an intrinsic memory clear to 0.
// dst must be a void pointer (alias of u8*).
// size is the copy size in bytes. size must be of type uint32.
func (b *Builder) Memzero(dst, size *Value) {
	b.Memset(dst, b.Scalar(uint8(0)), size)
}
