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

// Package encoder is a plugin for the gapil compiler to generate command and
// state encode functions.
package encoder

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/text/cases"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/serialization"
)

const (
	initialBufferCapacity = 1024
)

type encoder struct {
	*compiler.C

	// Generated functions.
	funcs funcs

	// All the entities that can be encoded.
	entities entities

	// Runtime functions used to perform the encoding.
	callbacks callbacks
}

// encoder generated functions.
type funcs struct {
	writeVarint *codegen.Function                     // void write_varint(buffer*, u64 val)
	writeZigzag *codegen.Function                     // void write_zigzag(buffer*, u64 val)
	encodeToBuf map[*semantic.Class]*codegen.Function // void(context*, buffer*)
}

// Build implements the compiler.Plugin interface.
func (e *encoder) Build(c *compiler.C) {
	*e = encoder{
		C: c,
		funcs: funcs{
			encodeToBuf: map[*semantic.Class]*codegen.Function{},
		},
		entities: entities{
			types:      map[semantic.Type]*entity{},
			funcParams: map[*semantic.Function]*entity{},
			funcCalls:  map[*semantic.Function]*entity{},
			state:      map[*semantic.API]*entity{},
		},
	}

	e.parseCallbacks()

	e.entities.buildTypes(e)

	e.buildWriteFuncs()
	e.buildEncoderFuncs()
}

func (e *encoder) buildWriteFuncs() {
	e.funcs.writeVarint = e.M.Function(e.T.Void, "write_varint", e.T.CtxPtr, e.T.BufPtr, e.T.Uint64).
		LinkOnceODR()
	e.C.Build(e.funcs.writeVarint, func(s *compiler.S) {
		buf, val := s.Parameter(1), s.Parameter(2)
		// while (i >= 0x80) {
		// 	bytes[length] = static_cast<uint8>(i | 0x80);
		// 	i >>= 7;
		// 	++length;
		// }
		// bytes[length] = static_cast<uint8>(i);
		i := s.LocalInit("i", val.Cast(e.T.Uint64))
		length := s.LocalInit("length", s.Scalar(uint32(0)))
		bytes := s.Local("bytes", e.T.TypeOf([10]byte{}))
		s.While(func() *codegen.Value {
			return s.GreaterOrEqualTo(i.Load(), s.Scalar(uint64(0x80)))
		}, func() {
			v, idx := i.Load(), length.Load()
			bytes.Index(0, idx).Store(s.Or(v, s.Scalar(uint64(0x80))).Cast(e.T.Uint8))
			i.Store(s.ShiftRight(v, s.Scalar(uint64(7))))
			length.Store(s.Add(idx, s.Scalar(uint32(1))))
		})
		bytes.Index(0, length.Load()).Store(i.Load().Cast(e.T.Uint8))
		e.AppendBuffer(s, buf, s.Add(length.Load(), s.Scalar(uint32(1))), bytes.Index(0, 0))
	})

	e.funcs.writeZigzag = e.M.Function(e.T.Void, "write_zigzag", e.T.CtxPtr, e.T.BufPtr, e.T.Uint64).
		LinkOnceODR()
	e.C.Build(e.funcs.writeZigzag, func(s *compiler.S) {
		buf, val := s.Parameter(1), s.Parameter(2)
		// (n << 1) ^ (n >> 63)
		val = val.Cast(e.T.Int64)
		lhs := s.ShiftLeft(val, s.Scalar(int64(1)))
		rhs := s.ShiftRight(val, s.Scalar(int64(63)))
		zigzag := s.Xor(lhs, rhs)
		e.writeVarint(s, buf, zigzag)
	})
}

// mgEncode returns the mangling function for an encode method on the given
// class.
func (e *encoder) mgEncode(mgClass *mangling.Class) *mangling.Function {
	return &mangling.Function{
		Name:   "encode",
		Parent: mgClass,
		Parameters: []mangling.Type{
			mangling.Pointer{To: mangling.Void},
			mangling.Bool,
		},
		Const: true,
	}
}

// buildEncoderFuncs generates the encode functions for all the API classes,
// command parameters, command calls and state.
func (e *encoder) buildEncoderFuncs() {
	e.buildClassEncodeToBufFuncs()
	e.buildClassEncodeFuncs()

	e.buildStateEncodeFunc()

	e.buildCommandEncodeFuncs()
	e.buildCommandCallEncodeFuncs()
}

// buildClassEncodeToBufFuncs builds the encode_to_buf() method for each API
// class type.
// encode_to_buf() encode the class message to a buffer.
func (e *encoder) buildClassEncodeToBufFuncs() {
	for _, api := range e.APIs {
		for _, ty := range api.Classes {
			if !e.hasEntity(ty) {
				continue
			}
			e.funcs.encodeToBuf[ty] = e.Method(false, e.T.Target(ty), e.T.Void, "encode_to_buf", e.T.CtxPtr, e.T.BufPtr).
				LinkPrivate()
		}
		for _, class := range api.Classes {
			if !e.hasEntity(class) {
				continue
			}
			e.C.Build(e.funcs.encodeToBuf[class], func(s *compiler.S) {
				this, buf := s.Parameter(0), s.Parameter(2)
				for i, f := range class.Fields {
					ptr := this.Index(0, f.Name())
					e.encodeField(s, ptr, buf, serialization.ClassFieldStart+serialization.ProtoFieldID(i), f.Type)
				}
			})
		}
	}
}

// buildClassEncodeFuncs builds the encode() method for each API class type.
// encode() will call gapil_encode_type() with the class type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) buildClassEncodeFuncs() {
	for _, api := range e.APIs {
		for _, class := range api.Classes {
			if class.Annotations.GetAnnotation("serialize") == nil {
				continue
			}
			e.C.Build(e.Method(true, e.T.Target(class), e.T.VoidPtr, "encode", e.T.CtxPtr, e.T.Bool), func(s *compiler.S) {
				this, isGroup := s.Parameter(0), s.Parameter(2)

				e.debug(s, "encoding class: '"+class.Name()+"' this: %p, ctx: %p", this, s.Ctx)

				typeID := s.Call(e.ent(class).encodeType, s.Ctx)

				buf, delBuf := e.newBuf(s)

				s.Call(e.funcs.encodeToBuf[class], this, s.Ctx, buf)

				bufOffset := buf.Index(0, compiler.BufSize).Load()
				bufData := buf.Index(0, compiler.BufData).Load()

				out := s.Call(e.callbacks.encodeObject, s.Ctx, isGroup.Cast(e.T.Uint8), typeID, bufOffset, bufData)

				delBuf()

				s.Return(out)
			})
		}
	}
}

// buildStateEncodeFunc builds the encode() method for each API state object.
// encode() will call gapil_encode_type() with the state type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) buildStateEncodeFunc() {
	for _, api := range e.APIs {
		mgState := &mangling.Class{
			Parent: e.Root,
			Name:   cases.Title(api.Name()) + "State",
		}

		stateTy := e.T.Globals.Field(api.Name()).Type
		statePtr := e.T.Pointer(stateTy)

		mgEncode := e.mgEncode(mgState)
		e.C.Build(e.M.Function(
			e.T.VoidPtr,
			e.Mangler(mgEncode),
			statePtr, e.T.CtxPtr, e.T.Bool), func(s *compiler.S) {

			this, isGroup := s.Parameter(0), s.Parameter(2)

			e.debug(s, "encoding "+api.Name()+" state: this: %p, ctx: %p", this, s.Ctx)

			typeID := s.Call(e.entities.state[api].encodeType, s.Ctx)

			buf, delBuf := e.newBuf(s)

			for i, g := range encodeableGlobals(api) {
				e.debug(s, "encoding "+api.Name()+" global '"+g.Name()+"'")
				ptr := this.Index(0, g.Name())
				e.encodeField(s, ptr, buf, serialization.StateStart+serialization.ProtoFieldID(i), g.Type)
			}

			bufOffset := buf.Index(0, compiler.BufSize).Load()
			bufData := buf.Index(0, compiler.BufData).Load()
			out := s.Call(e.callbacks.encodeObject, s.Ctx, isGroup.Cast(e.T.Uint8), typeID, bufOffset, bufData)

			delBuf()

			s.Return(out)
		})
	}
}

// buildCommandEncodeFuncs builds the encode() method for the each API command
// and the API command call (if they don't return void).
// encode() will call gapil_encode_type() with the state type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) buildCommandEncodeFuncs() {
	for _, api := range e.APIs {
		for _, cmd := range api.Functions {
			mgCmd := &mangling.Namespace{Name: "cmd", Parent: e.Root}
			mgClass := &mangling.Class{Name: cmd.Name(), Parent: mgCmd}
			mgEncode := e.mgEncode(mgClass)

			fields := []codegen.Field{
				codegen.Field{Name: "thread", Type: e.T.Uint64},
			}
			for _, p := range cmd.FullParameters {
				fields = append(fields, codegen.Field{Name: p.Name(), Type: e.T.Target(p.Type)})
			}

			thisTy := e.T.Pointer(e.T.Struct(cmd.Name()+"_params", fields...))

			e.C.Build(e.M.Function(
				e.T.VoidPtr,
				e.Mangler(mgEncode),
				thisTy, e.T.CtxPtr, e.T.Bool), func(s *compiler.S) {

				this, isGroup := s.Parameter(0), s.Parameter(2)

				e.debug(s, "encoding command: '"+cmd.Name()+"' this: %p, ctx: %p", this, s.Ctx)

				typeID := s.Call(e.entities.funcParams[cmd].encodeType, s.Ctx)

				buf, delBuf := e.newBuf(s)

				threadPtr := this.Index(0, "thread")
				e.encodeField(s, threadPtr, buf, serialization.CmdThread, semantic.Uint64Type)
				for i, p := range cmd.CallParameters() {
					ptr := this.Index(0, p.Name())
					e.encodeField(s, ptr, buf, serialization.CmdFieldStart+serialization.ProtoFieldID(i), p.Type)
				}

				bufOffset := buf.Index(0, compiler.BufSize).Load()
				bufData := buf.Index(0, compiler.BufData).Load()
				out := s.Call(e.callbacks.encodeObject, s.Ctx, isGroup.Cast(e.T.Uint8), typeID, bufOffset, bufData)

				delBuf()

				s.Return(out)
			})
		}
	}
}

// buildCommandCallEncodeFuncs builds the encode() method for the each API
// command call (for commands that don't return void).
// encode() will call gapil_encode_type() with the state type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) buildCommandCallEncodeFuncs() {
	for _, api := range e.APIs {
		for _, cmd := range api.Functions {
			if cmd.Return.Type == semantic.VoidType {
				continue
			}
			mgCmd := &mangling.Namespace{Name: "cmd", Parent: e.Root}
			mgClass := &mangling.Class{Name: cmd.Name() + "Call", Parent: mgCmd}
			mgEncode := e.mgEncode(mgClass)

			fields := []codegen.Field{{Name: "result", Type: e.T.Target(cmd.Return.Type)}}
			thisTy := e.T.Pointer(e.T.Struct(cmd.Name()+"_call", fields...))

			e.C.Build(e.M.Function(
				e.T.VoidPtr,
				e.Mangler(mgEncode),
				thisTy, e.T.CtxPtr, e.T.Bool), func(s *compiler.S) {

				this, isGroup := s.Parameter(0), s.Parameter(2)

				e.debug(s, "encoding command call: '"+cmd.Name()+"' this: %p, ctx: %p", this, s.Ctx)

				typeID := s.Call(e.entities.funcCalls[cmd].encodeType, s.Ctx)

				buf, delBuf := e.newBuf(s)

				resultPtr := this.Index(0, "result")
				e.encodeField(s, resultPtr, buf, serialization.CmdResult, cmd.Return.Type)

				bufOffset := buf.Index(0, compiler.BufSize).Load()
				bufData := buf.Index(0, compiler.BufData).Load()
				out := s.Call(e.callbacks.encodeObject, s.Ctx, isGroup.Cast(e.T.Uint8), typeID, bufOffset, bufData)

				delBuf()

				s.Return(out)
			})
		}
	}
}

// newBuf creates and returns a pointer to a new buffer stored on the stack
// along with a function that will release the buffer's data.
func (e *encoder) newBuf(s *compiler.S) (ptr *codegen.Value, del func()) {
	buf := s.Local("buffer", e.T.Buf)
	e.InitBuffer(s, buf, s.Scalar(uint32(initialBufferCapacity)))
	return buf, func() { e.TermBuffer(s, buf) }
}

// encodeField encodes a single proto field to buf.
func (e *encoder) encodeField(s *compiler.S, ptr, buf *codegen.Value, id serialization.ProtoFieldID, ty semantic.Type) {
	e.debug(s, "encoding field at %p: '%s' id: %d, ty: %s", ptr, ptr.Name(), id, ty.Name())

	ent := e.ent(ty)
	doEncode := func(s *compiler.S) {
		switch ty := semantic.Underlying(ty).(type) {
		case *semantic.StaticArray:
			if ent := e.ent(ty.ValueType); ent.isPacked() {
				e.writeWireAndTag(s, buf, proto.WireBytes, id)
				e.writeBlob(s, buf, func(buf *codegen.Value) {
					for i := uint32(0); i < ty.Size; i++ {
						e.encodeValue(s, ptr.Index(0, int(i)), buf, ty.ValueType)
					}
				})
			} else {
				for i := uint32(0); i < ty.Size; i++ {
					e.writeWireAndTag(s, buf, ent.wireTy, id)
					e.encodeValue(s, ptr.Index(0, int(i)), buf, ty.ValueType)
				}
			}
		default:
			e.writeWireAndTag(s, buf, ent.wireTy, id)
			e.encodeValue(s, ptr, buf, ty)
		}
	}

	if cond := e.shouldEncode(s, ptr, ty); cond != nil {
		s.If(cond, doEncode)
	} else {
		doEncode(s)
	}
}

// shouldEncode returns a boolean value which determines whether the proto value
// should be encoded. Fields that are zero or nil are not encoded.
func (e *encoder) shouldEncode(s *compiler.S, ptr *codegen.Value, ty semantic.Type) *codegen.Value {
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.Int8Type,
			semantic.Int16Type,
			semantic.Int32Type,
			semantic.Int64Type,
			semantic.IntType,
			semantic.Uint8Type,
			semantic.Uint16Type,
			semantic.Uint32Type,
			semantic.Uint64Type,
			semantic.UintType,
			semantic.CharType,
			semantic.SizeType,
			semantic.BoolType,
			semantic.Float32Type,
			semantic.Float64Type,
			semantic.StringType:
			return s.NotEqual(ptr.Load(), s.Zero(e.T.Target(ty)))
		}
	case *semantic.Enum,
		*semantic.Pointer,
		*semantic.Reference:
		return s.NotEqual(ptr.Load(), s.Zero(e.T.Target(ty)))

	case *semantic.StaticArray,
		*semantic.Class,
		*semantic.Map,
		*semantic.Slice:
		return nil
	}
	e.Fail("Unsupported type: %T %v", ty, ty)
	return nil
}

// encodeValue encodes the proto value to buf.
func (e *encoder) encodeValue(s *compiler.S, ptr, buf *codegen.Value, ty semantic.Type) {
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.Int8Type,
			semantic.Int16Type,
			semantic.Int32Type,
			semantic.Int64Type,
			semantic.IntType,
			semantic.CharType:
			e.writeZigzag(s, buf, ptr.Load().Cast(e.T.Int64)) // Sign-ext
			return
		case semantic.Uint8Type,
			semantic.Uint16Type,
			semantic.Uint32Type,
			semantic.Uint64Type,
			semantic.UintType,
			semantic.SizeType,
			semantic.BoolType:
			e.writeZigzag(s, buf, ptr.Load().Cast(e.T.Uint64)) // No sign-ext
			return
		case semantic.Float32Type:
			e.writeFixed32(s, buf, ptr.Load())
			return
		case semantic.Float64Type:
			e.writeFixed64(s, buf, ptr.Load())
			return
		case semantic.StringType:
			strPtr := ptr.Load()
			size := strPtr.Index(0, compiler.StringLength).Load().Cast(e.T.Uint32)
			bytes := strPtr.Index(0, compiler.StringData, 0)
			e.debug(s, "encoding string at %p: size: %d, refcount: %d, str: %s",
				strPtr, size, strPtr.Index(0, compiler.StringRefCount).Load(), bytes)
			e.writeBytes(s, buf, size, bytes)
			return
		}
	case *semantic.Enum:
		e.encodeValue(s, ptr, buf, ty.NumberType)
		return
	case *semantic.Pointer:
		e.encodeValue(s, ptr, buf, semantic.Uint64Type)
		return
	case *semantic.StaticArray:
		panic("Must be handled in encodeField")
	case *semantic.Class:
		e.writeBlob(s, buf, func(buf *codegen.Value) {
			s.Call(e.funcs.encodeToBuf[ty], ptr, s.Ctx, buf)
		})
		return
	case *semantic.Reference:
		refPtr := ptr.Load()
		s.If(s.NotEqual(refPtr, s.Zero(refPtr.Type())), func(s *compiler.S) {
			signedRefID := s.Call(e.callbacks.encodeBackref, s.Ctx, refPtr.Cast(e.T.VoidPtr))
			newRef := s.GreaterOrEqualTo(signedRefID, s.Scalar(int64(0)))
			refID := s.Select(newRef, signedRefID, s.Negate(signedRefID))

			e.writeBlob(s, buf, func(buf *codegen.Value) {
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.RefRef)
				e.writeZigzag(s, buf, refID)

				s.If(newRef, func(s *compiler.S) {
					e.writeWireAndTag(s, buf, proto.WireBytes, serialization.RefVal)
					e.encodeValue(s, refPtr.Index(0, compiler.RefValue), buf, ty.To)
				})
			})
		})
		return
	case *semantic.Map:
		mapPtr := ptr.Load()
		e.writeBlob(s, buf, func(buf *codegen.Value) {
			signedRefID := s.Call(e.callbacks.encodeBackref, s.Ctx, mapPtr.Cast(e.T.VoidPtr))
			newRef := s.GreaterOrEqualTo(signedRefID, s.Scalar(int64(0)))
			refID := s.Select(newRef, signedRefID, s.Negate(signedRefID))

			e.writeWireAndTag(s, buf, proto.WireVarint, serialization.MapRef)
			e.writeZigzag(s, buf, refID)

			s.If(newRef, func(s *compiler.S) {
				count := mapPtr.Index(0, compiler.MapCount).Load()
				e.debug(s, "encoding map '"+ty.Name()+"' at %p: cnt: %d, cap: %d, refcount: %d",
					mapPtr, count,
					mapPtr.Index(0, compiler.MapCapacity).Load(),
					mapPtr.Index(0, compiler.MapRefCount).Load())
				s.If(s.NotEqual(count, s.Zero(count.Type())), func(s *compiler.S) {
					writer := func(ty semantic.Type, id serialization.ProtoFieldID) (write func(*codegen.Value), flush func()) {
						ent := e.ent(ty)
						if ent.isPacked() {
							packBuf, delBuf := e.newBuf(s)
							return func(v *codegen.Value) {
									e.encodeValue(s, v, packBuf, ty)
								}, func() {
									e.writeWireAndTag(s, buf, proto.WireBytes, id)
									e.writeBuffer(s, buf, packBuf)
									delBuf()
								}
						}
						return func(v *codegen.Value) {
							e.writeWireAndTag(s, buf, ent.wireTy, id)
							e.encodeValue(s, v, buf, ty)
						}, func() {}
					}

					writeVal, flushVal := writer(ty.ValueType, serialization.MapVal)
					writeKey, flushKey := writer(ty.KeyType, serialization.MapKey)

					e.IterateMap(s, mapPtr, semantic.Uint32Type, func(i, k, v *codegen.Value) {
						e.debug(s, "encoding map '"+ty.Name()+"' val %d", i.Load())
						writeVal(v)
					})
					e.IterateMap(s, mapPtr, semantic.Uint32Type, func(i, k, v *codegen.Value) {
						e.debug(s, "encoding map '"+ty.Name()+"' key %d", i.Load())
						writeKey(k)
					})

					flushVal()
					flushKey()
				})
			})
		})
		return
	case *semantic.Slice:
		e.writeBlob(s, buf, func(buf *codegen.Value) {
			root := ptr.Index(0, compiler.SliceRoot).Load()
			base := ptr.Index(0, compiler.SliceBase).Load()
			size := ptr.Index(0, compiler.SliceSize).Load()
			count := ptr.Index(0, compiler.SliceCount).Load()
			pool := ptr.Index(0, compiler.SlicePool).Load()

			s.If(s.NotEqual(root, s.Zero(root.Type())), func(s *compiler.S) {
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.SliceRoot)
				e.writeVarint(s, buf, root.Cast(e.T.Uint64))
			})

			s.If(s.NotEqual(base, s.Zero(base.Type())), func(s *compiler.S) {
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.SliceBase)
				e.writeVarint(s, buf, base.Cast(e.T.Uint64))
			})

			s.If(s.NotEqual(size, s.Zero(size.Type())), func(s *compiler.S) {
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.SliceSize)
				e.writeVarint(s, buf, size)
			})

			s.If(s.NotEqual(count, s.Zero(size.Type())), func(s *compiler.S) {
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.SliceCount)
				e.writeVarint(s, buf, count)
			})

			s.If(s.Not(pool.IsNull()), func(s *compiler.S) {
				id := pool.Index(0, compiler.PoolID).Load()
				e.writeWireAndTag(s, buf, proto.WireVarint, serialization.SlicePool)
				e.writeVarint(s, buf, id)
			})
		})
		s.Call(e.callbacks.sliceEncoded, s.Ctx, ptr)
		return
	}

	e.Fail("Unsupported type: %T %v", ty, ty)
}

// writeWireAndTag writes a wire type and tag (proto field ID) to buf.
// All proto fields are prefixed with a wire and tag.
func (e *encoder) writeWireAndTag(s *compiler.S, buf *codegen.Value, wire uint64, tag serialization.ProtoFieldID) {
	if tag < 1 {
		panic(fmt.Sprintf("Illegal tag: %v"))
	}
	e.writeVarint(s, buf, s.Scalar(wire|(uint64(tag)<<3)))
}

// writeFixed32 writes a fixed size, 32-bit number to buf.
func (e *encoder) writeFixed32(s *compiler.S, buf, val *codegen.Value) {
	i := s.LocalInit("i", val.Bitcast(e.T.Uint32))
	e.AppendBuffer(s, buf, s.Scalar(uint32(4)), i.Cast(e.T.VoidPtr))
}

// writeFixed64 writes a fixed size, 64-bit number to buf.
func (e *encoder) writeFixed64(s *compiler.S, buf, val *codegen.Value) {
	i := s.LocalInit("i", val.Bitcast(e.T.Uint64))
	e.AppendBuffer(s, buf, s.Scalar(uint32(8)), i.Cast(e.T.VoidPtr))
}

// writeZigzag writes a variable length integer to buf.
func (e *encoder) writeVarint(s *compiler.S, buf, val *codegen.Value) {
	s.Call(e.funcs.writeVarint, s.Ctx, buf, val.Cast(e.T.Uint64))
}

// writeZigzag writes a zigzag encoded, variable length integer to buf.
func (e *encoder) writeZigzag(s *compiler.S, buf, val *codegen.Value) {
	s.Call(e.funcs.writeZigzag, s.Ctx, buf, val.Cast(e.T.Uint64))
}

// writeBlob writes calls inner with a new buffer. Once inner returns the buffer
// size is encoded as a varint. followed by the buffer itself.
func (e *encoder) writeBlob(s *compiler.S, buf *codegen.Value, inner func(*codegen.Value)) {
	innerBuf, delBuf := e.newBuf(s)
	defer delBuf()

	inner(innerBuf)

	e.writeBuffer(s, buf, innerBuf)
}

// writeBuffer writes the size of buffer src to dst as a varint to dst, and
// then writes the src buffer data to dst.
func (e *encoder) writeBuffer(s *compiler.S, dst, src *codegen.Value) {
	size := src.Index(0, compiler.BufSize).Load()
	bytes := src.Index(0, compiler.BufData).Load()
	e.writeBytes(s, dst, size, bytes)
}

// writeBytes writes size as a varint to buf, and then writes size bytes to buf.
func (e *encoder) writeBytes(s *compiler.S, buf, size, bytes *codegen.Value) {
	e.writeVarint(s, buf, size)
	e.AppendBuffer(s, buf, size, bytes)
}

// encodeableGlobals returns the list API globals that are encodable.
func encodeableGlobals(api *semantic.API) []*semantic.Global {
	out := make([]*semantic.Global, 0, len(api.Globals))
	for _, g := range api.Globals {
		if serialization.IsEncodable(g) {
			out = append(out, g)
		}
	}
	return out
}

// debug emits a log message if debugging is enabled (see function body).
func (e *encoder) debug(s *compiler.S, msg string, args ...interface{}) {
	const enabled = false
	if enabled {
		e.LogI(s, msg, args...)
	}
}
