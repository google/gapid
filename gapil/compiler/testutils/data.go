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

package testutils

import (
	"bytes"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

// Encode encodes all the vals to the returned byte slice with
// little-endianness. Struct alignment and padding is ignored.
func Encode(vals ...interface{}) []byte {
	buf := &bytes.Buffer{}
	w := endian.Writer(buf, device.LittleEndian)
	for _, d := range vals {
		binary.Write(w, d)
	}
	return buf.Bytes()
}

// Pad returns a slice of n zero bytes. Pad can be used to align data passed
// to Encode.
func Pad(n int) []byte { return make([]byte, n) }
