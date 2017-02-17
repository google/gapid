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

package pod

//TODO: Rename [Read|Write]Simple to POD?

// Readable is the interface to things that readable as Plain Old Data.
type Readable interface {
	// ReadSimple is invoked by a Reader to read the POD.
	ReadSimple(Reader)
}

// Writable is the interface to things that are writable as Plain Old Data.
type Writable interface {
	// WriteSimple is invoked by a Writer to write the POD.
	WriteSimple(Writer)
}
