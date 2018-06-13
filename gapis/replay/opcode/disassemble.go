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

package opcode

import (
	"io"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

// Disassemble disassembles and returns the stream of encoded Opcodes from r,
// stopping once an EOF is reached.
func Disassemble(r io.Reader, byteOrder device.Endian) ([]Opcode, error) {
	d := endian.Reader(r, byteOrder)
	opcodes := []Opcode{}
	for {
		opcode, err := Decode(d)
		switch err {
		case nil:
			opcodes = append(opcodes, opcode)
		case io.EOF:
			return opcodes, nil
		default:
			return nil, err
		}
	}
}
