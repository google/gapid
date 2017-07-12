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
	OpCall         = 0
	OpPushI        = 1
	OpLoadC        = 2
	OpLoadV        = 3
	OpLoad         = 4
	OpPop          = 5
	OpStoreV       = 6
	OpStore        = 7
	OpResource     = 8
	OpPost         = 9
	OpCopy         = 10
	OpClone        = 11
	OpStrcpy       = 12
	OpExtend       = 13
	OpAdd          = 14
	OpLabel        = 15
	OpSwitchThread = 16
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
	default:
		panic(fmt.Errorf("Unknown ValueType %d", uint32(t)))
	}
}
