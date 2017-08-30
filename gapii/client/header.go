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

package client

import (
	"io"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

var magic = [4]byte{'s', 'p', 'y', '0'}

const version = 1

// The GAPII header is defined as:
//
// const size_t MAX_PATH = 512;
//
// struct ConnectionHeader {
//     uint8_t  mMagic[4];                     // 's', 'p', 'y', '0'
//     uint32_t mVersion;                      // 1
//     uint32_t mObserveFrameFrequency;        // non-zero == enabled.
//     uint32_t mObserveDrawFrequency;         // non-zero == enabled.
//     uint32_t mStartFrame;                   // non-zero == Frame to start at.
//     uint32_t mNumFrames;                    // non-zero == Number of frames to capture.
//     uint32_t mAPIs;                         // Bitset of APIS to enable.
//     uint32_t mFlags;                        // Combination of FLAG_XX bits.
//     char     mLibInterceptorPath[MAX_PATH]; // Path to libinterceptor.so
// };
//
// All fields are encoded little-endian with no compression, regardless of
// architecture. All changes must be kept in sync with:
//   platform/tools/gpu/gapii/cc/connection_header.h

func sendHeader(out io.Writer, options Options, gvrHandle uint64, libInterceptorPath string) error {
	const maxPath = 512
	w := endian.Writer(out, device.LittleEndian)
	for _, m := range magic {
		w.Uint8(m)
	}
	w.Uint32(version)
	w.Uint32(options.ObserveFrameFrequency)
	w.Uint32(options.ObserveDrawFrequency)
	w.Uint32(options.StartFrame)
	w.Uint32(options.FramesToCapture)
	w.Uint32(options.APIs)
	w.Uint32(uint32(options.Flags))
	w.Uint64(gvrHandle)
	var path [maxPath]byte
	copy(path[:], libInterceptorPath)
	w.Data(path[:])
	return w.Error()
}
