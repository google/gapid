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

package replay

//#include "gapil/runtime/cc/replay/asm.h"
import "C"

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

// asm holds the types used to build the replay instructions.
type asm struct {
	value        codegen.Type
	call         codegen.Type
	push         codegen.Type
	pop          codegen.Type
	copy         codegen.Type
	clone        codegen.Type
	load         codegen.Type
	store        codegen.Type
	strcpy       codegen.Type
	resource     codegen.Type
	post         codegen.Type
	add          codegen.Type
	label        codegen.Type
	switchthread codegen.Type
}

func (r *replayer) parseAsmTypes() {
	r.T.asm.value = r.T.TypeOf(C.gapil_replay_asm_value{})

	// instructions
	r.T.asm.call = r.T.TypeOf(C.gapil_replay_asm_call{})
	r.T.asm.push = r.T.TypeOf(C.gapil_replay_asm_push{})
	r.T.asm.pop = r.T.TypeOf(C.gapil_replay_asm_pop{})
	r.T.asm.copy = r.T.TypeOf(C.gapil_replay_asm_copy{})
	r.T.asm.clone = r.T.TypeOf(C.gapil_replay_asm_clone{})
	r.T.asm.load = r.T.TypeOf(C.gapil_replay_asm_load{})
	r.T.asm.store = r.T.TypeOf(C.gapil_replay_asm_store{})
	r.T.asm.strcpy = r.T.TypeOf(C.gapil_replay_asm_strcpy{})
	r.T.asm.resource = r.T.TypeOf(C.gapil_replay_asm_resource{})
	r.T.asm.post = r.T.TypeOf(C.gapil_replay_asm_post{})
	r.T.asm.add = r.T.TypeOf(C.gapil_replay_asm_add{})
	r.T.asm.label = r.T.TypeOf(C.gapil_replay_asm_label{})
	r.T.asm.switchthread = r.T.TypeOf(C.gapil_replay_asm_switchthread{})
}

const (
	asmValueType = "data_type"
	asmValueData = "data"
)

type asmType uint32

const (
	asmTypeBool               = asmType(C.GAPIL_REPLAY_ASM_TYPE_BOOL)
	asmTypeInt8               = asmType(C.GAPIL_REPLAY_ASM_TYPE_INT8)
	asmTypeInt16              = asmType(C.GAPIL_REPLAY_ASM_TYPE_INT16)
	asmTypeInt32              = asmType(C.GAPIL_REPLAY_ASM_TYPE_INT32)
	asmTypeInt64              = asmType(C.GAPIL_REPLAY_ASM_TYPE_INT64)
	asmTypeUint8              = asmType(C.GAPIL_REPLAY_ASM_TYPE_UINT8)
	asmTypeUint16             = asmType(C.GAPIL_REPLAY_ASM_TYPE_UINT16)
	asmTypeUint32             = asmType(C.GAPIL_REPLAY_ASM_TYPE_UINT32)
	asmTypeUint64             = asmType(C.GAPIL_REPLAY_ASM_TYPE_UINT64)
	asmTypeFloat              = asmType(C.GAPIL_REPLAY_ASM_TYPE_FLOAT)
	asmTypeDouble             = asmType(C.GAPIL_REPLAY_ASM_TYPE_DOUBLE)
	asmTypeAbsolutePointer    = asmType(C.GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER)
	asmTypeConstantPointer    = asmType(C.GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER)
	asmTypeVolatilePointer    = asmType(C.GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER)
	asmTypeObservedPointerNS0 = asmType(C.GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0)
)

type asmNamespace uint32

type asmInst uint8

const (
	asmInstCall         = asmInst(C.GAPIL_REPLAY_ASM_INST_CALL)
	asmInstPush         = asmInst(C.GAPIL_REPLAY_ASM_INST_PUSH)
	asmInstPop          = asmInst(C.GAPIL_REPLAY_ASM_INST_POP)
	asmInstCopy         = asmInst(C.GAPIL_REPLAY_ASM_INST_COPY)
	asmInstClone        = asmInst(C.GAPIL_REPLAY_ASM_INST_CLONE)
	asmInstLoad         = asmInst(C.GAPIL_REPLAY_ASM_INST_LOAD)
	asmInstStore        = asmInst(C.GAPIL_REPLAY_ASM_INST_STORE)
	asmInstStrcpy       = asmInst(C.GAPIL_REPLAY_ASM_INST_STRCPY)
	asmInstResource     = asmInst(C.GAPIL_REPLAY_ASM_INST_RESOURCE)
	asmInstPost         = asmInst(C.GAPIL_REPLAY_ASM_INST_POST)
	asmInstAdd          = asmInst(C.GAPIL_REPLAY_ASM_INST_ADD)
	asmInstLabel        = asmInst(C.GAPIL_REPLAY_ASM_INST_LABEL)
	asmInstSwitchthread = asmInst(C.GAPIL_REPLAY_ASM_INST_SWITCHTHREAD)
)

func (r *replayer) asmWrite(s *compiler.S, v *codegen.Value) {
	bufPtr := s.Ctx.Index(0, data, stream)
	r.AppendBufferData(s, bufPtr, s.LocalInit("write", v))
}

func (r *replayer) asmWriteInst(s *compiler.S, ty asmInst, inst *codegen.Value) {
	r.asmWrite(s, s.Scalar(ty))
	r.asmWrite(s, inst)
}

func (r *replayer) asmValue(s *compiler.S, val *codegen.Value, ty asmType) *codegen.Value {
	return s.Zero(r.T.asm.value).
		Insert(asmValueType, s.Scalar(ty)).
		Insert(asmValueData, val)
}

func (r *replayer) asmBool(s *compiler.S, v *codegen.Value) *codegen.Value {
	return r.asmValue(s, v.Cast(r.T.Uint64), asmTypeBool)
}

func (r *replayer) asmVolatilePtr(s *compiler.S, addr *codegen.Value) *codegen.Value {
	return r.asmValue(s, addr.Cast(r.T.Uint64), asmTypeVolatilePointer)
}

func (r *replayer) asmAbsolutePtr(s *compiler.S, addr *codegen.Value) *codegen.Value {
	return r.asmValue(s, addr.Cast(r.T.Uint64), asmTypeAbsolutePointer)
}

func (r *replayer) asmConstantPtr(s *compiler.S, addr *codegen.Value) *codegen.Value {
	return r.asmValue(s, addr.Cast(r.T.Uint64), asmTypeConstantPointer)
}

func (r *replayer) asmObservedPtr(s *compiler.S, addr *codegen.Value, ns asmNamespace) *codegen.Value {
	return r.asmValue(s, addr.Cast(r.T.Uint64), asmTypeObservedPointerNS0+asmType(ns))
}

func (r *replayer) asmResource(s *compiler.S, resID, dst *codegen.Value) {
	r.asmWriteInst(s, asmInstResource,
		s.Zero(r.T.asm.resource).
			Insert("index", resID).
			Insert("dest", dst))
}

func (r *replayer) asmCall(s *compiler.S, apiIdx, funcID int, pushReturn bool) {
	r.asmWriteInst(s, asmInstCall,
		s.Zero(r.T.asm.call).
			Insert("push_return", s.Scalar(pushReturn).Cast(r.T.Uint8)).
			Insert("api_index", s.Scalar(uint8(apiIdx))).
			Insert("function_id", s.Scalar(uint16(funcID))))
}

func (r *replayer) asmLabel(s *compiler.S, label *codegen.Value) {
	r.asmWriteInst(s, asmInstLabel,
		s.Zero(r.T.asm.label).
			Insert("value", label))
}

func (r *replayer) asmClone(s *compiler.S, n *codegen.Value) {
	r.asmWriteInst(s, asmInstClone,
		s.Zero(r.T.asm.clone).
			Insert("n", n))
}

func (r *replayer) asmStore(s *compiler.S, dst *codegen.Value) {
	r.asmWriteInst(s, asmInstStore,
		s.Zero(r.T.asm.store).
			Insert("dst", dst))
}

func (r *replayer) asmLoad(s *compiler.S, src *codegen.Value, ty asmType) {
	r.asmWriteInst(s, asmInstLoad,
		s.Zero(r.T.asm.load).
			Insert("source", src).
			Insert("data_type", s.Scalar(ty)))
}

func (r *replayer) asmPush(s *compiler.S, val *codegen.Value) {
	r.asmWriteInst(s, asmInstPush,
		s.Zero(r.T.asm.push).
			Insert("value", val))
}

func (r *replayer) asmType(ty semantic.Type) asmType {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return asmTypeBool
		case semantic.IntType:
			size := r.replayLayout.Integer.Size
			switch size {
			case 1:
				return asmTypeInt8
			case 2:
				return asmTypeInt16
			case 4:
				return asmTypeInt32
			case 8:
				return asmTypeInt64
			default:
				r.Fail("Unsupported integer size: %v", size)
			}
		case semantic.UintType:
			size := r.replayLayout.Integer.Size
			switch size {
			case 1:
				return asmTypeUint8
			case 2:
				return asmTypeUint16
			case 4:
				return asmTypeUint32
			case 8:
				return asmTypeUint64
			default:
				r.Fail("Unsupported integer size: %v", size)
			}
		case semantic.SizeType:
			size := r.replayLayout.Size.Size
			switch size {
			case 1:
				return asmTypeUint8
			case 2:
				return asmTypeUint16
			case 4:
				return asmTypeUint32
			case 8:
				return asmTypeUint64
			default:
				r.Fail("Unsupported size_t size: %v", size)
			}
		case semantic.CharType:
			size := r.replayLayout.Char.Size
			switch size {
			case 1:
				return asmTypeUint8
			case 2:
				return asmTypeUint16
			case 4:
				return asmTypeUint32
			case 8:
				return asmTypeUint64
			default:
				r.Fail("Unsupported char_t size: %v", size)
			}
		case semantic.Int8Type:
			return asmTypeInt8
		case semantic.Uint8Type:
			return asmTypeUint8
		case semantic.Int16Type:
			return asmTypeInt16
		case semantic.Uint16Type:
			return asmTypeUint16
		case semantic.Int32Type:
			return asmTypeInt32
		case semantic.Uint32Type:
			return asmTypeUint32
		case semantic.Int64Type:
			return asmTypeInt64
		case semantic.Uint64Type:
			return asmTypeUint64
		case semantic.Float32Type:
			return asmTypeFloat
		case semantic.Float64Type:
			return asmTypeDouble
		}
	case *semantic.Pointer:
		return asmTypeVolatilePointer
	}
	r.Fail("Unhandled type %v", ty.Name())
	return 0
}
