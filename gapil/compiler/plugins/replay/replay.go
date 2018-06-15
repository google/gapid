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

// Package replay is a plugin for the gapil compiler to generate replay opcodes
// for commands.
package replay

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapis/replay/opcode"
	"github.com/google/gapid/gapis/replay/protocol"
)

// replayer is the compiler plugin that adds replay opcode generation
// functionality.
type replayer struct {
	*compiler.C
	replayABI  *device.ABI
	callFPtrTy codegen.Type
	getOpcodes *codegen.Function
}

func (r *replayer) Build(c *compiler.C) {
	*r = replayer{
		C:         c,
		replayABI: c.Settings.TargetABI,
	}
	r.getOpcodes = c.M.Function(c.T.BufPtr, GetReplayOpcodes, c.T.CtxPtr)
	c.Build(r.getOpcodes, func(s *compiler.S) { s.Return(s.Ctx.Index(0, data, opcodes)) })
}

func (r *replayer) emitLabel(s *compiler.S) {
	cmdID := s.Ctx.Index(0, compiler.ContextCmdID).Load().Cast(r.T.Uint32)
	cmdID = s.And(cmdID, s.Scalar(uint32(0x3ffffff))) // Labels have 26 bit values.

	r.write(s, r.packCX(s, protocol.OpLabel, cmdID))
}

func (r *replayer) buildCall(cmd *semantic.Function) *codegen.Function {
	callFunc := r.M.Function(r.T.Void, "call_"+cmd.Name(), r.T.CtxPtr)

	r.C.Build(callFunc, func(s *compiler.S) {
		// Load the command parameters
		r.LoadParameters(s, cmd)

		// Push the arguments
		for _, p := range cmd.CallParameters() {
			r.pushValue(s, r.Parameter(s, p), p.Type)
		}

		// Emit the CALL
		var apiIdx uint8
		if r.API.Index != nil {
			apiIdx = uint8(*r.API.Index)
		}
		cmdIdx := r.API.CommandIndex(cmd)
		if cmdIdx < 0 {
			panic("Command not found in API?!")
		}
		apiAndFunc := s.Scalar(opcode.PackAPIIndexFunctionID(apiIdx, uint16(cmdIdx)))
		pushReturn := cmd.Return.Type != semantic.VoidType // TODO
		r.write(s, r.packCX(s, protocol.OpCall, r.setBit(s, apiAndFunc, 24, pushReturn)))
	})

	return callFunc
}

func (r *replayer) emitCall(s *compiler.S) {
	callFunc := s.Ctx.Index(0, data, call).Load()
	s.CallIndirect(callFunc, s.Ctx)
}

func (r *replayer) pushValue(s *compiler.S, val *codegen.Value, ty semantic.Type) {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			r.writePush(s, val, protocol.Type_Bool)
		case semantic.IntType:
			size := r.replayABI.MemoryLayout.Integer.Size
			switch size {
			case 8:
				r.writePush(s, val, protocol.Type_Int8)
			case 16:
				r.writePush(s, val, protocol.Type_Int16)
			case 32:
				r.writePush(s, val, protocol.Type_Int32)
			case 64:
				r.writePush(s, val, protocol.Type_Int64)
			default:
				r.Fail("Unsupported integer size: %v", size)
			}
		case semantic.UintType:
			size := r.replayABI.MemoryLayout.Integer.Size
			switch size {
			case 8:
				r.writePush(s, val, protocol.Type_Uint8)
			case 16:
				r.writePush(s, val, protocol.Type_Uint16)
			case 32:
				r.writePush(s, val, protocol.Type_Uint32)
			case 64:
				r.writePush(s, val, protocol.Type_Uint64)
			default:
				r.Fail("Unsupported integer size: %v", size)
			}
		case semantic.SizeType:
			size := r.replayABI.MemoryLayout.Size.Size
			switch size {
			case 8:
				r.writePush(s, val, protocol.Type_Uint8)
			case 16:
				r.writePush(s, val, protocol.Type_Uint16)
			case 32:
				r.writePush(s, val, protocol.Type_Uint32)
			case 64:
				r.writePush(s, val, protocol.Type_Uint64)
			default:
				r.Fail("Unsupported size_t size: %v", size)
			}
		case semantic.CharType:
			size := r.replayABI.MemoryLayout.Char.Size
			switch size {
			case 8:
				r.writePush(s, val, protocol.Type_Uint8)
			case 16:
				r.writePush(s, val, protocol.Type_Uint16)
			case 32:
				r.writePush(s, val, protocol.Type_Uint32)
			case 64:
				r.writePush(s, val, protocol.Type_Uint64)
			default:
				r.Fail("Unsupported char_t size: %v", size)
			}
		case semantic.Int8Type:
			r.writePush(s, val, protocol.Type_Int8)
		case semantic.Uint8Type:
			r.writePush(s, val, protocol.Type_Uint8)
		case semantic.Int16Type:
			r.writePush(s, val, protocol.Type_Int16)
		case semantic.Uint16Type:
			r.writePush(s, val, protocol.Type_Uint16)
		case semantic.Int32Type:
			r.writePush(s, val, protocol.Type_Int32)
		case semantic.Uint32Type:
			r.writePush(s, val, protocol.Type_Uint32)
		case semantic.Int64Type:
			r.writePush(s, val, protocol.Type_Int64)
		case semantic.Uint64Type:
			r.writePush(s, val, protocol.Type_Uint64)
		case semantic.Float32Type:
			r.writePush(s, val, protocol.Type_Float)
		case semantic.Float64Type:
			r.writePush(s, val, protocol.Type_Double)
		}
	case *semantic.StaticArray:
		r.Fail("TODO")
	case *semantic.Pointer:
		r.Fail("TODO")
	case *semantic.Class:
		r.Fail("TODO")
	}
}

const (
	// Various bit-masks used by opcode writing.
	// Many opcodes can fit values into the opcode itself.
	// These masks are used to determine which values fit.

	mask19 = uint64(0x7ffff)
	mask20 = uint64(0xfffff)
	mask26 = uint64(0x3ffffff)
	mask45 = uint64(0x1fffffffffff)
	mask46 = uint64(0x3fffffffffff)
	mask52 = uint64(0xfffffffffffff)

	//     ▏60       ▏50       ▏40       ▏30       ▏20       ▏10
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●● mask19
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●● mask20
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●● mask26
	// ○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask45
	// ○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask46
	// ○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask52
	//                                            ▕      PUSHI 20     ▕
	//                                      ▕         EXTEND 26       ▕
)

func (r *replayer) writePush(s *compiler.S, v *codegen.Value, t protocol.Type) {
	V32 := func(v uint32) *codegen.Value { return s.Scalar(v) }
	V64 := func(v uint64) *codegen.Value { return s.Scalar(v) }

	push := func(s *compiler.S, v *codegen.Value) {
		r.write(s, r.packCYZ(s, protocol.OpPushI, V32(uint32(t)), v.Cast(r.T.Uint32)))
	}
	extend := func(s *compiler.S, v *codegen.Value) {
		r.write(s, r.packCX(s, protocol.OpExtend, v.Cast(r.T.Uint32)))
	}
	switch t {
	case protocol.Type_Float:
		v := v.Bitcast(r.T.Uint32)
		push(s, s.ShiftRight(v, V32(23)))
		mask := s.And(v, V32(0x7fffff))
		s.If(s.NotEqual(mask, V32(0)), func(s *compiler.S) {
			extend(s, mask)
		})
	case protocol.Type_Double:
		v := v.Bitcast(r.T.Uint64)
		push(s, s.ShiftRight(v, V64(52)))
		v = s.And(v, V64(mask52))
		s.If(s.NotEqual(v, V64(0)), func(s *compiler.S) {
			extend(s, s.ShiftRight(v, V64(26)))
			extend(s, s.And(v, V64(mask26)))
		})
	case protocol.Type_Int8, protocol.Type_Int16, protocol.Type_Int32, protocol.Type_Int64:
		v := v.Cast(r.T.Uint64)
		// Signed PUSHI types are sign-extended
		s.Switch([]compiler.SwitchCase{
			{
				// case v&^mask19 == 0:
				// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                                            ▕      PUSHI 20     ▕
				// push(v)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask19)), V64(0))}
				},
				Block: func(s *compiler.S) {
					push(s, v)
				},
			}, {
				// case v&^mask19 == ^mask19:
				// ●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                                            ▕      PUSHI 20     ▕
				// push(v & mask20)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask19)), V64(^mask19))}
				},
				Block: func(s *compiler.S) {
					push(s, s.And(v, V64(mask20)))
				},
			}, {
				// case v&^mask45 == 0:
				// ○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
				// push(v >> 26)
				// extend(v & mask26)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask45)), V64(0))}
				},
				Block: func(s *compiler.S) {
					push(s, s.ShiftRight(v, V64(26)))
					extend(s, s.And(v, V64(mask26)))
				},
			}, {
				// case v&^mask45 == ^mask45:
				// ●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
				// push((v >> 26) & mask20)
				// extend(v & mask26)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask45)), V64(^mask45))}
				},
				Block: func(s *compiler.S) {
					push(s, s.And(s.ShiftRight(v, V64(26)), V64(mask20)))
					extend(s, s.And(v, V64(mask26)))
				},
			},
		},
			// default:
			// ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
			// push(v >> 52)
			// extend((v >> 26) & mask26)
			// extend(v & mask26)
			func(s *compiler.S) {
				push(s, s.ShiftRight(v, V64(52)))
				extend(s, s.And(s.ShiftRight(v, V64(26)), V64(mask26)))
				extend(s, s.And(v, V64(mask26)))
			},
		)

	case protocol.Type_Bool,
		protocol.Type_Uint8, protocol.Type_Uint16, protocol.Type_Uint32, protocol.Type_Uint64,
		protocol.Type_AbsolutePointer, protocol.Type_ConstantPointer, protocol.Type_VolatilePointer:

		v := v.Cast(r.T.Uint64)
		// Signed PUSHI types are sign-extended
		s.Switch([]compiler.SwitchCase{
			{
				// case v&^mask20 == 0:
				// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                                            ▕      PUSHI 20     ▕
				// push(v)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask20)), V64(0))}
				},
				Block: func(s *compiler.S) {
					push(s, v)
				},
			}, {
				// case v&^mask46 == 0:
				// ○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
				//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
				// push(v >> 26)
				// extend(v & mask26)
				Conditions: func(s *compiler.S) []*codegen.Value {
					return []*codegen.Value{s.Equal(s.And(v, V64(^mask46)), V64(0))}
				},
				Block: func(s *compiler.S) {
					push(s, s.ShiftRight(v, V64(26)))
					extend(s, s.And(v, V64(mask26)))
				},
			},
		},
			// default:
			// ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
			// push(v >> 52)
			// extend((v >> 26) & mask26)
			// extend(v & mask26)
			func(s *compiler.S) {
				push(s, s.ShiftRight(v, V64(52)))
				extend(s, s.And(s.ShiftRight(v, V64(26)), V64(mask26)))
				extend(s, s.And(v, V64(mask26)))
			},
		)
	default:
		r.Fail("Cannot push value type %s", t)
	}
}

func (r *replayer) write(s *compiler.S, v *codegen.Value) {
	bufPtr := s.Ctx.Index(0, data, opcodes)
	r.AppendBufferData(s, bufPtr, s.LocalInit("write", v))
}

func (r *replayer) setBit(s *compiler.S, v *codegen.Value, bit uint, high bool) *codegen.Value {
	if high {
		return s.Or(v, s.Scalar(uint32(1<<bit)))
	}
	return s.And(v, s.Scalar(^uint32(1<<bit)))
}

// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
func (r *replayer) packC(s *compiler.S, c protocol.Opcode) *codegen.Value {
	if c >= 0x3f {
		r.Fail("c exceeds 6 bits (0x%x)", c)
	}
	return s.Scalar(uint32(c) << 26)
}

// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
func (r *replayer) packCX(s *compiler.S, c protocol.Opcode, x *codegen.Value) *codegen.Value {
	s.If(s.GreaterThan(x, s.Scalar(uint32(0x3ffffff))), func(s *compiler.S) {
		r.Log(s, log.Fatal, "x exceeds 26 bits (0x%x)", x)
	})
	return s.Or(r.packC(s, c), x)
}

// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃y │y │y │y │y │y ┃z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
func (r *replayer) packCYZ(s *compiler.S, c protocol.Opcode, y, z *codegen.Value) *codegen.Value {
	s.If(s.GreaterThan(y, s.Scalar(uint32(0x3f))), func(s *compiler.S) {
		r.Log(s, log.Fatal, "y exceeds 6 bits (0x%x)", y)
	})
	s.If(s.GreaterThan(z, s.Scalar(uint32(0xfffff))), func(s *compiler.S) {
		r.Log(s, log.Fatal, "z exceeds 20 bits (0x%x)", z)
	})
	return s.Or(s.Or(r.packC(s, c), s.ShiftLeft(y, s.Scalar(uint32(20)))), z)
}
