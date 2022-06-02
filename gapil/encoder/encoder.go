// Copyright (C) 2022 Google Inc.
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

// Package encoder generates C++ code to encode the API structs in the
// proto wire format to be stored in the trace file.
//
// This file creates only few abstraction so that it is easier to read,
// understand, and debug. It is in part based off the old LLVM-based
// gapil/compiler/plugins/encoder/encoder.go file, but changed to emit
// C++ code, rather than LLVM IR.
package encoder

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/text/cases"
	"github.com/google/gapid/core/text/reflow"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/serialization"
)

const (
	initialBufferCapacity = 1024
)

type Settings struct {
	Namespace string
	Out       io.Writer
}

type encoder struct {
	APIs         []*semantic.API
	namespace    string
	entities     entities
	out          *reflow.Writer
	lastBuffer   int
	lastIterator int
}

// GenerateEncoders is the main entry point and will generate the C++ code for the enoders of
// the provided APIs.
func GenerateEncoders(apis []*semantic.API, mappings *semantic.Mappings, settings Settings) error {
	encoder := &encoder{
		APIs:      apis,
		namespace: settings.Namespace,
		entities: entities{
			types:      map[semantic.Type]*entity{},
			funcParams: map[*semantic.Function]*entity{},
			funcCalls:  map[*semantic.Function]*entity{},
			state:      map[*semantic.API]*entity{},
		},
		out:          reflow.New(settings.Out),
		lastBuffer:   0,
		lastIterator: 0,
	}
	encoder.entities.buildTypes(encoder)

	encoder.emitHeader()

	encoder.Namespace("", func() {
		encoder.entities.emitEncodeTypeFuncs(encoder)
		encoder.emitClassEncodeToBufFuncs()
	})

	encoder.Namespace(settings.Namespace, func() {
		encoder.emitEncoderFuncs()
	})

	return encoder.out.Flush()
}

// empty emits an empty line.
func (e *encoder) empty() {
	e.out.EOL()
}

// Line printf formats the given arguments and writes the result with a newline to the output.
func (e *encoder) Line(format string, a ...interface{}) {
	fmt.Fprintf(e.out, format, a...)
	e.out.EOL()
}

// namespace emits a namspace of the given name and calls the callback within the namespace.
func (e *encoder) Namespace(name string, body func()) {
	e.Line("namespace %s {", name)
	e.empty()
	body()
	e.empty()
	e.Line("}  // namespace %s", name)
	e.empty()
}

// Function emits a function with the given signature and calls the callback for its body.
func (e *encoder) Function(sig string, body func()) {
	e.Line("%s {", sig)
	e.out.Increase()
	body()
	e.out.Decrease()
	e.Line("}")
	e.empty()
}

// If emits an if statement with the given condition and calls the callback for the then body.
func (e *encoder) If(cond string, body func()) {
	e.Line("if (%s) {", cond)
	e.out.Increase()
	body()
	e.out.Decrease()
	e.Line("}")
}

// IFElse emits an if statement with the given condition and calls the callbacks for the then
// and else bodies.
func (e *encoder) IfElse(cond string, then, alt func()) {
	e.Line("if (%s) {", cond)
	e.out.Increase()
	then()
	e.out.Decrease()
	e.Line("} else {")
	e.out.Increase()
	alt()
	e.out.Decrease()
	e.Line("}")
}

// ForIter emits an for loop and calls inner with a new unused iterator variable.
func (e *encoder) ForIter(collection value, inner func(it value)) {
	it := fmt.Sprintf("it%d", e.lastIterator)
	e.lastIterator++
	e.Line("for (auto %s = %s.begin(); %s != %s.end(); %s++) {", it, collection, it, collection, it)
	e.out.Increase()
	inner(value{it, true})
	e.out.Decrease()
	e.Line("}")
	e.lastIterator--
}

// emitHeader emits the source code header
func (e *encoder) emitHeader() {
	e.Line("// Auto-generated code. Do not modify!")
	e.Line("// Genereted by apic compile")
	e.empty()

	for _, api := range e.APIs {
		e.Line("#include \"%s_types.h\"", api.Name())
	}
	e.Line("#include \"gapil/runtime/cc/encoder.inc\"")
	e.empty()
}

// emitEncoderFuncs generates the encode functions for all the API classes,
// command parameters, command calls and state.
func (e *encoder) emitEncoderFuncs() {
	e.emitClassEncodeFuncs()
	e.emitStateEncodeFunc()
	e.emitCommandEncodeFuncs()
}

// emitClassEncodeToBufFuncs builds the encode_to_buf() method for each API
// class type.
// encode_to_buf() encode the class message to a buffer.
func (e *encoder) emitClassEncodeToBufFuncs() {
	for _, api := range e.APIs {
		for _, class := range api.Classes {
			if e.hasEntity(class) {
				e.Line("void %s(gapil::Encoder* enc, buffer* buf, const %s::%s* v);",
					e.ent(class).encodeToBuf, e.namespace, class.Name())
			}
		}
	}
	e.empty()

	// Note: This is intentionally split into two passes to allow cyclic encodes.

	for _, api := range e.APIs {
		for _, class := range api.Classes {
			if e.hasEntity(class) {
				sig := fmt.Sprintf("void %s(gapil::Encoder* enc, buffer* buf, const %s::%s* v)", e.ent(class).encodeToBuf, e.namespace, class.Name())
				e.Function(sig, func() {
					buf := buffer{"buf", true}
					val := value{"v", true}
					for i, f := range class.Fields {
						e.Line("// encoding %s.%s", class.Name(), f.Name())
						e.encodeField(buf, val.child("m"+f.Name()), serialization.ClassFieldStart+serialization.ProtoFieldID(i), f.Type)
					}
				})
			}
		}
	}
}

// emitClassEncodeFuncs builds the encode() method for each API class type.
// encode() will call gapil_encode_type() with the class type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) emitClassEncodeFuncs() {
	for _, api := range e.APIs {
		for _, class := range api.Classes {
			if class.Annotations.GetAnnotation("serialize") == nil {
				continue
			}
			e.Function(fmt.Sprintf("void* %s::encode(gapil::Encoder* enc, bool is_group) const", class.Name()), func() {
				e.Line("int32_t typeId = %s(enc);", e.ent(class).encodeType)
				e.Line("void* result;")
				e.withBuffer(func(buf buffer) {
					e.Line("%s(enc, %P, this);", e.ent(class).encodeToBuf, buf)
					e.Line("result = enc->encodeObject(is_group, typeId, %s.size, %s.data);", buf, buf)
				})
				e.Line("return result;")
			})
		}
	}
}

// emitStateEncodeFunc builds the encode() method for each API state object.
// encode() will call gapil_encode_type() with the state type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) emitStateEncodeFunc() {
	for _, api := range e.APIs {
		e.Function(fmt.Sprintf("void* %sState::encode(gapil::Encoder* enc, bool is_group) const", cases.Title(api.Name())), func() {
			e.Line("int32_t typeId = %s(enc);", e.state(api).encodeType)
			e.Line("void* result;")
			e.withBuffer(func(buf buffer) {
				for i, g := range encodeableGlobals(api) {
					e.Line("// encoding %sState.%s", api.Name(), g.Name())
					e.encodeField(buf, value{g.Name(), false}, serialization.StateStart+serialization.ProtoFieldID(i), g.Type)
				}
				e.Line("result = enc->encodeObject(is_group, typeId, %s.size, %s.data);", buf, buf)
			})
			e.Line("return result;")
		})
	}
}

// emitCommandEncodeFuncs builds the encode() method for the each API command
// and the API command call (if they don't return void).
// encode() will call gapil_encode_type() with the state type before encoding
// the proto message with gapil_encode_object.
func (e *encoder) emitCommandEncodeFuncs() {
	for _, api := range e.APIs {
		for _, cmd := range api.Functions {
			if cmd.Annotations.GetAnnotation("pfn") != nil {
				continue
			}
			e.Function(fmt.Sprintf("void* cmd::%s::encode(gapil::Encoder* enc, bool is_group) const", cmd.Name()), func() {
				e.Line("int32_t typeId = %s(enc);", e.command(cmd).encodeType)
				e.Line("void* result;")
				e.withBuffer(func(buf buffer) {
					e.encodeField(buf, value{"thread", false}, serialization.CmdThread, semantic.Uint64Type)
					for i, p := range cmd.CallParameters() {
						e.encodeField(buf, value{p.Name(), false}, serialization.CmdFieldStart+serialization.ProtoFieldID(i), p.Type)
					}
					e.Line("result = enc->encodeObject(is_group, typeId, %s.size, %s.data);", buf, buf)
				})
				e.Line("return result;")
			})

			if cmd.Return.Type != semantic.VoidType {
				e.Function(fmt.Sprintf("void* cmd::%sCall::encode(gapil::Encoder* enc, bool is_group) const", cmd.Name()), func() {
					e.Line("int32_t typeId = %s(enc);", e.commandCall(cmd).encodeType)
					e.Line("void* _result;")
					e.withBuffer(func(buf buffer) {
						e.encodeField(buf, value{"result", false}, serialization.CmdResult, cmd.Return.Type)
						e.Line("_result = enc->encodeObject(is_group, typeId, %s.size, %s.data);", buf, buf)
					})
					e.Line("return _result;")
				})
			}
		}
	}
}

// encodeField emits the code to encode a single proto field.
func (e *encoder) encodeField(buf buffer, val value, id serialization.ProtoFieldID, ty semantic.Type) {
	e.shouldEncodeGuard(val, ty, func() {
		switch ty := semantic.Underlying(ty).(type) {
		case *semantic.StaticArray:
			if ent := e.ent(ty.ValueType); ent.isPacked() {
				e.writeWireAndTag(buf, proto.WireBytes, id)
				e.writeBlob(buf, func(buf buffer) {
					for i := uint32(0); i < ty.Size; i++ {
						e.encodeValue(buf, val.element(i), ty.ValueType)
					}
				})
			} else {
				for i := uint32(0); i < ty.Size; i++ {
					e.writeWireAndTag(buf, ent.wireTy, id)
					e.encodeValue(buf, val.element(i), ty.ValueType)
				}
			}
		default:
			e.writeWireAndTag(buf, e.ent(ty).wireTy, id)
			e.encodeValue(buf, val, ty)
		}
	})
}

// shouldEncodeGuard emits an if statement if required for the type to check whether the given value
// should be encoded. The inner function is called either within the true block or wihtout the if.
func (e *encoder) shouldEncodeGuard(val value, ty semantic.Type, inner func()) {
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
			e.If(val.String(), inner)
			return
		}
	case *semantic.Enum,
		*semantic.Pointer,
		*semantic.Reference:
		e.If(val.String(), inner)
		return
	case *semantic.StaticArray,
		*semantic.Class,
		*semantic.Map,
		*semantic.Slice:
		inner()
		return
	}
	panic(fmt.Sprintf("Unsupported type: %T %v", ty, ty))
}

// encodeValue encodes the proto value to buf.
func (e *encoder) encodeValue(buf buffer, val value, ty semantic.Type) {
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.Int8Type,
			semantic.Int16Type,
			semantic.Int32Type,
			semantic.Int64Type,
			semantic.IntType,
			semantic.CharType,
			semantic.Uint8Type,
			semantic.Uint16Type,
			semantic.Uint32Type,
			semantic.Uint64Type,
			semantic.UintType,
			semantic.SizeType,
			semantic.BoolType:
			e.Line("write_zig_zag(%P, %s);", buf, val)
			return
		case semantic.Float32Type:
			e.Line("gapil_buffer_append(%P, 4, %P);", buf, val)
			return
		case semantic.Float64Type:
			e.Line("gapil_buffer_append(%P, 8, %P);", buf, val)
			return
		case semantic.StringType:
			e.Line("write_var_int(%P, %s.length());", buf, val)
			e.Line("gapil_buffer_append(%P, %s.length(), %s.c_str());", buf, val, val)
			return
		}
	case *semantic.Enum:
		e.encodeValue(buf, val, ty.NumberType)
		return
	case *semantic.Pointer:
		e.Line("write_zig_zag(%P, reinterpret_cast<intptr_t>(%s));", buf, val)
		return
	case *semantic.StaticArray:
		panic("Must be handled in encodeField")
	case *semantic.Class:
		e.writeBlob(buf, func(buf buffer) {
			e.Line("%s(enc, %P, %P);", e.ent(ty).encodeToBuf, buf, val)
		})
		return
	case *semantic.Reference:
		e.IfElse(val.String(), func() {
			e.Line("int64_t refId = enc->encodeBackref(%s.get());", val)
			e.writeBlob(buf, func(buf buffer) {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.RefRef)
				e.IfElse("refId > 0", func() {
					e.Line("write_zig_zag(%P, refId);", buf)
					e.writeWireAndTag(buf, proto.WireBytes, serialization.RefVal)
					e.encodeValue(buf, value{fmt.Sprintf("%s.get()", val), true}, ty.To)
				}, func() {
					e.Line("write_zig_zag(%P, -refId);", buf)
				})
			})
		}, func() {
			e.Line("write_zig_zag(%P, 0);", buf)
		})
		return
	case *semantic.Map:
		e.writeBlob(buf, func(buf buffer) {
			e.Line("int64_t refId = enc->encodeBackref(%s.instance_ptr());", val)
			e.writeWireAndTag(buf, proto.WireVarint, serialization.MapRef)
			e.IfElse("refId > 0", func() {
				e.Line("write_zig_zag(%P, refId);", buf)
				e.If(fmt.Sprintf("!%s.empty()", val), func() {
					keyEntity := e.ent(ty.KeyType)
					valEntity := e.ent(ty.ValueType)
					if keyEntity.isPacked() {
						e.Line("buffer_t keyBuf;")
						e.Line("gapil_buffer_init(enc->arena(), &keyBuf, %d);", initialBufferCapacity)
					}
					if valEntity.isPacked() {
						e.Line("buffer_t valBuf;")
						e.Line("gapil_buffer_init(enc->arena(), &valBuf, %d);", initialBufferCapacity)
					}
					e.ForIter(val, func(it value) {
						if keyEntity.isPacked() {
							e.encodeValue(buffer{"keyBuf", false}, it.child("first"), ty.KeyType)
						} else {
							e.writeWireAndTag(buf, keyEntity.wireTy, serialization.MapKey)
							e.encodeValue(buf, it.child("first"), ty.KeyType)
						}
						if valEntity.isPacked() {
							e.encodeValue(buffer{"valBuf", false}, it.child("second"), ty.ValueType)
						} else {
							e.writeWireAndTag(buf, valEntity.wireTy, serialization.MapVal)
							e.encodeValue(buf, it.child("second"), ty.ValueType)
						}
					})
					if keyEntity.isPacked() {
						e.writeWireAndTag(buf, proto.WireBytes, serialization.MapKey)
						e.Line("write_var_int(%P, keyBuf.size);", buf)
						e.Line("gapil_buffer_append(%P, keyBuf.size, keyBuf.data);", buf)
						e.Line("enc->arena()->free(keyBuf.data);")
					}
					if valEntity.isPacked() {
						e.writeWireAndTag(buf, proto.WireBytes, serialization.MapVal)
						e.Line("write_var_int(%P, valBuf.size);", buf)
						e.Line("gapil_buffer_append(%P, valBuf.size, valBuf.data);", buf)
						e.Line("enc->arena()->free(valBuf.data);")
					}
				})
			}, func() {
				e.Line("write_zig_zag(%P, -refId);", buf)
			})
		})
		return
	case *semantic.Slice:
		e.writeBlob(buf, func(buf buffer) {
			e.If(fmt.Sprintf("%s.root() != 0", val), func() {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.SliceRoot)
				e.Line("write_var_int(%P, %s.root());", buf, val)
			})
			e.If(fmt.Sprintf("%s.base() != 0", val), func() {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.SliceBase)
				e.Line("write_var_int(%P, %s.base());", buf, val)
			})
			e.If(fmt.Sprintf("%s.size() != 0", val), func() {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.SliceSize)
				e.Line("write_var_int(%P, %s.size());", buf, val)
			})
			e.If(fmt.Sprintf("%s.count() != 0", val), func() {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.SliceCount)
				e.Line("write_var_int(%P, %s.count());", buf, val)
			})
			e.If(fmt.Sprintf("!%s.is_app_pool()", val), func() {
				e.writeWireAndTag(buf, proto.WireVarint, serialization.SlicePool)
				e.Line("write_var_int(%P, %s.pool_id());", buf, val)
			})
		})
		e.Line("enc->sliceEncoded(%s.instance_ptr());", val)
		return
	}

	panic(fmt.Sprintf("Unsupported type: %T %v", ty, ty))
}

// writeWireAndTag writes a wire type and tag (proto field ID) to buf.
// All proto fields are prefixed with a wire and tag.
func (e *encoder) writeWireAndTag(buf buffer, wire uint64, tag serialization.ProtoFieldID) {
	if tag < 1 {
		panic(fmt.Sprintf("Illegal tag: %v"))
	}
	e.Line("write_var_int(%P, %d|(%d<<3));", buf, wire, tag)
}

// withBuffer calls inner with a new buffer and de-allocates the buffer after.
func (e *encoder) withBuffer(inner func(buf buffer)) {
	buf := fmt.Sprintf("buf%d", e.lastBuffer)
	e.lastBuffer++
	e.Line("{")
	e.out.Increase()
	e.Line("buffer_t %s;", buf)
	e.Line("gapil_buffer_init(enc->arena(), &%s, %d);", buf, initialBufferCapacity)
	inner(buffer{buf, false})
	e.Line("enc->arena()->free(%s.data);", buf)
	e.out.Decrease()
	e.Line("}")
	e.lastBuffer--
}

// writeBlob calls inner with a new buffer. Once inner returns the buffer
// size is encoded as a varint followed by the buffer itself to the target buffer.
func (e *encoder) writeBlob(target buffer, inner func(buf buffer)) {
	e.withBuffer(func(buf buffer) {
		inner(buf)
		e.Line("write_var_int(%P, %s.size);", target, buf)
		e.Line("gapil_buffer_append(%P, %s.size, %s.data);", target, buf, buf)
	})
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

// value is a C++ epression, but tracks if it's a pointer or not.
type value struct {
	name string
	ptr  bool
}

// Format implements the fmt.Formatter interface for value, formatting the given value
// either as pointer (for %P) or as a value (%s).
func (v value) Format(f fmt.State, verb rune) {
	switch verb {
	case 'P':
		if v.ptr {
			fmt.Fprint(f, v.name)
		} else {
			fmt.Fprintf(f, "&%s", v.name)
		}
	default:
		if v.ptr {
			fmt.Fprintf(f, "*%s", v.name)
		} else {
			fmt.Fprint(f, v.name)
		}
	}
}

// child returns a new value that referrs to a field of v by the given name.
func (v value) child(name string) value {
	if v.ptr {
		return value{fmt.Sprintf("%s->%s", v.name, name), false}
	} else {
		return value{fmt.Sprintf("%s.%s", v.name, name), false}
	}
}

// element returns a new value that refers to the idx'th element of array v.
func (v value) element(idx uint32) value {
	if v.ptr {
		return value{fmt.Sprintf("(*%s)[%d]", v.name, idx), false}
	} else {
		return value{fmt.Sprintf("%s[%d]", v.name, idx), false}
	}
}

// String implements the fmt.Stringer interface.
func (v value) String() string {
	if v.ptr {
		return "*" + v.name
	} else {
		return v.name
	}
}

// buffer is a value, but for gapil's buffer_t type.
type buffer value

// Format implements the fmt.Formatter interface for buffer.
func (b buffer) Format(f fmt.State, verb rune) {
	value(b).Format(f, verb)
}
