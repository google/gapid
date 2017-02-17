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

// Package pack provides methods to deal with self describing files of proto data.
//
// The file format consists of a magic marker, followed by a Header.
// After that is a repeated sequence of uvarint length, tag and matching encoded message pair.
// Some section tags will also be followed by a string.
// The tag 0 is special, and marks a type entry, the body will be a descriptor.DescriptorProto.
package pack
