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

// Package endian implements binary.Reader and binary.Writer for reading /
// writing simple primivite types from a binary source.
//
// Boolean values are encoded as single bytes, where 0 represents false and non-
// zero represents true.
//
// Numeric types are all encoded as the simple native representation, but no
// attempt is made to align them.
//
// Strings are encoded in C style null terminated form.
package endian
