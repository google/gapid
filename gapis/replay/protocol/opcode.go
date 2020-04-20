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

package protocol

import "fmt"

// Opcode is one of the opcodes supported by the replay virtual machine.
type Opcode int

const (
	OpCall           = Opcode(0)
	OpPushI          = Opcode(1)
	OpLoadC          = Opcode(2)
	OpLoadV          = Opcode(3)
	OpLoad           = Opcode(4)
	OpPop            = Opcode(5)
	OpStoreV         = Opcode(6)
	OpStore          = Opcode(7)
	OpResource       = Opcode(8)
	OpPost           = Opcode(9)
	OpCopy           = Opcode(10)
	OpClone          = Opcode(11)
	OpStrcpy         = Opcode(12)
	OpExtend         = Opcode(13)
	OpAdd            = Opcode(14)
	OpLabel          = Opcode(15)
	OpSwitchThread   = Opcode(16)
	OpJumpLabel      = Opcode(17)
	OpJumpNZ         = Opcode(18)
	OpJumpZ          = Opcode(19)
	OpNotification   = Opcode(20)
	OpWait           = Opcode(21)
	OpInlineResource = Opcode(22)
)

// String returns the human-readable name of the opcode.
func (t Opcode) String() string {
	switch t {
	case OpCall:
		return "Call"
	case OpPushI:
		return "PushI"
	case OpLoadC:
		return "LoadC"
	case OpLoadV:
		return "LoadV"
	case OpLoad:
		return "Load"
	case OpPop:
		return "Pop"
	case OpStoreV:
		return "StoreV"
	case OpStore:
		return "Store"
	case OpResource:
		return "Resource"
	case OpPost:
		return "Post"
	case OpCopy:
		return "Copy"
	case OpClone:
		return "Clone"
	case OpStrcpy:
		return "Strcpy"
	case OpExtend:
		return "Extend"
	case OpAdd:
		return "Add"
	case OpLabel:
		return "Label"
	case OpSwitchThread:
		return "SwitchThread"
	case OpJumpLabel:
		return "JumpLabel"
	case OpJumpNZ:
		return "JumpNZ"
	case OpJumpZ:
		return "JumpZ"
	case OpNotification:
		return "Notification"
	case OpWait:
		return "Wait"
	case OpInlineResource:
		return "InlineResource"
	default:
		panic(fmt.Errorf("Unknown Opcode %d", uint32(t)))
	}
}
