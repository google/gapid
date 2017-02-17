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

// Package vle implements pod.Reader and pod.Writer using a variable
// length encoding format.
//
// Boolean values are encoded as single bytes, where 0 represents false and non-
// zero represents true.
//
// 8 bit values are encoded as single bytes.
// 16, 32 and 64 bit unsigned integers are encoded into a one or more bytes.
// The number of sequential ones starting from the most-significant bit of the
// first encoded byte describe the number of additional bytes that make up the
// unsigned integer. If the run of ones does not fill the byte, then the run is
// terminated with a zero bit. The unsigned integer value is then big-endian
// encoded into the remainder of the bits from the first byte and any
// additional bytes.
//
// For example, the 16-bit number 0xABC would be encoded as follows:
//
//   MSB                         LSB   MSB                          LSB
//  ╔═══╤═══╤═══╤═══╤═══╤═══╤═══╤═══╗ ╔═══╤═══╤═══╤═══╤═══╤═══╤═══╤═══╗
//  ║C₀ │C₁ │D₁₃│D₁₂│D₁₁│D₁₀│D₉ │D₈ ║ ║D₇ │D₆ │D₅ │D₄ │D₃ │D₂ │D₁ │D₀ ║
//  ║   │   │   │   │   │   │   │   ║ ║   │   │   │   │   │   │   │   ║
//  ║ 1 │ 0 │ 0 │ 0 │ 1 │ 0 │ 1 │ 0 ║ ║ 1 │ 0 │ 1 │ 1 │ 1 │ 1 │ 0 │ 0 ║
//  ╚═══╧═══╧═══╧═══╧═══╧═══╧═══╧═══╝ ╚═══╧═══╧═══╧═══╧═══╧═══╧═══╧═══╝
//                Byte₀                             Byte₁
//
// C has a run of 1, meaning there is one extra byte of data.
// D holds the unsigned integer value 0xABC with bits 0000 1010 1011 1100.
//
// Signed integers are converted to unsigned integers by interleaving negative
// then positive numbers before being encoded as unsigned integers (as described
// above). For example the signed numbers [±0, -1, +1, -2, +2] are first
// transformed to the unsigned integers [0, 1, 2, 3, 4] before being encoded.
// This is done so that small negative and small positive numbers are encoded
// with fewer bytes.
//
// 32 bit floating-point numbers are first converted to a 32 bit unsigned
// integer using math.Float32bits and then byte-reversed before being encoded
// as a unsigned integer.
//
// 64 bit floating-point numbers are first converted to a 64 bit unsigned
// integer using math.Float64bits and then byte-reversed before being encoded
// as a unsigned integer.
//
// The bytes of floating-point numbers are reversed so that floating pointer
// numbers with simple fractional parts are encoded with fewer bytes, as
// these are considered the common case.
//
// String encodes a 32 bit unsigned integer representing the length of the
// string in bytes followed by the string data encoded in UTF-8:
//   count                  uint32 // in bytes
//   data0, data1, data2... byte   // string in UTF-8
//
package vle
