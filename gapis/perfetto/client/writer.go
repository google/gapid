// Copyright (C) 2019 Google Inc.
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

	"github.com/golang/protobuf/proto"

	ipc "protos/perfetto/ipc"
)

const (
	tag = (1 << 3) /* tag id */ | 2 /* type */
)

// PacketWriter serializes Perfetto packets to a Writer.
type PacketWriter struct {
	out    io.Writer
	buffer []byte
}

// NewPacketWriter returns a packet writer that serializes to the given writer.
func NewPacketWriter(out io.Writer) *PacketWriter {
	return &PacketWriter{out: out}
}

// Write serializes the given packet slices.
func (w *PacketWriter) Write(slices []*ipc.ReadBuffersResponse_Slice) error {
	b := w.buffer
	for _, s := range slices {
		b = append(b, s.Data...)
		if s.GetLastSliceForPacket() {
			if err := w.write(b); err != nil {
				w.buffer = nil
				return err
			}
			b = nil
		}
	}
	w.buffer = b
	return nil
}

func (w *PacketWriter) write(packet []byte) error {
	buf := make([]byte, 0, 1+10)
	b := proto.NewBuffer(buf)
	b.EncodeVarint(tag)
	b.EncodeVarint(uint64(len(packet)))
	buf = append(b.Bytes(), packet...)
	_, err := w.out.Write(buf)
	return err
}
