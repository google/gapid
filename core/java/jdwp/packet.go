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

package jdwp

import (
	"fmt"

	"github.com/google/gapid/core/data/binary"
)

type packetID uint32

type packetFlags uint8

const packetIsReply = packetFlags(0x80)

type cmdPacket struct {
	id     packetID
	flags  packetFlags
	cmdSet cmdSet
	cmdID  cmdID
	data   []byte
}

// JDWP uses the following structs for all communication:
//
// struct cmdPacket {
//   length uint32       4 bytes
//   id     packetID     4 bytes
//   flags  packetFlags  1 bytes
//   cmdSet cmdSet       1 bytes
//   cmd    uint8        1 bytes
//   data   []byte       N bytes
// }
//
// struct reply {
//   length uint32       4 bytes
//   id     packetID     4 bytes
//   flags  packetFlags  1 bytes
//   err    errorCode    2 bytes
//   data   []byte       N bytes
// }

func (p cmdPacket) write(w binary.Writer) error {
	w.Uint32(11 + uint32(len(p.data)))
	w.Uint32(uint32(p.id))
	w.Uint8(uint8(p.flags))
	w.Uint8(uint8(p.cmdSet))
	w.Uint8(uint8(p.cmdID))
	w.Data(p.data)
	return w.Error()
}

type replyPacket struct {
	id   packetID
	err  Error
	data []byte
}

func (c *Connection) readPacket() (interface{}, error) {
	len := c.r.Uint32()
	if err := c.r.Error(); err != nil {
		return nil, err
	}
	if len < 11 {
		return replyPacket{}, fmt.Errorf("Packet length too short (%d)", len)
	}
	id := packetID(c.r.Uint32())
	flags := packetFlags(c.r.Uint8())
	if flags&packetIsReply != 0 {
		// Reply packet
		out := replyPacket{
			id:  id,
			err: Error(c.r.Uint16()),
		}
		out.data = make([]byte, len-11)
		c.r.Data(out.data)
		return out, c.r.Error()
	}
	// Command packet
	out := cmdPacket{
		cmdSet: cmdSet(c.r.Uint8()),
		cmdID:  cmdID(c.r.Uint8()),
	}
	out.data = make([]byte, len-11)
	c.r.Data(out.data)
	return out, c.r.Error()
}
