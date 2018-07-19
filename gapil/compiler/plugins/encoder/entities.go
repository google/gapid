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

package encoder

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/text/cases"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/serialization"
)

// entity describes an encodable entity. This includes types, function parameter
// messages, function call messages and the state.
type entity struct {
	// node is the semantic node for the given entitiy.
	node semantic.Node

	// desc is the proto descriptor for this entitiy. For non-message types,
	// this is nil.
	desc *descriptor.DescriptorProto

	// protoTy is the proto type.
	protoTy descriptor.FieldDescriptorProto_Type

	// protoTy is the proto wire type.
	wireTy uint64

	// protoTy is the field label to use for this entity.
	label descriptor.FieldDescriptorProto_Label

	// fqn is the the fully qualified name for this entity.
	fqn string

	// encode is the function used to encode this entitiy.
	// Only set for message types.
	encodeType *codegen.Function // u32(encoder*)
}

// entities holds all the entities for the API.
type entities struct {
	types      map[semantic.Type]*entity
	funcParams map[*semantic.Function]*entity
	funcCalls  map[*semantic.Function]*entity
	slice      *entity
	state      map[*semantic.API]*entity
}

// isPacked returns true if this entity should be packed together into a single
// buffer when encoded as an array. If isPacked returns false then they are
// stored as separate, repeated fields.
func (t entity) isPacked() bool {
	switch t.wireTy {
	case proto.WireVarint,
		proto.WireFixed32,
		proto.WireFixed64:
		return true
	default:
		return false
	}
}

// protoDescField builds and returns a FieldDescriptorProto for the given entity
// when stored in a proto field with the given name and id.
func (t entity) protoDescField(name string, id serialization.ProtoFieldID) *descriptor.FieldDescriptorProto {
	var typename *string
	if t.fqn != "" {
		t := "." + t.fqn
		typename = &t
	}
	return &descriptor.FieldDescriptorProto{
		Name:     &name,
		JsonName: proto.String(cases.Snake(name).ToCamel()),
		Number:   proto.Int32(int32(id)),
		Label:    &t.label,
		Type:     &t.protoTy,
		TypeName: typename,
	}
}

// ent is a helper method for getting the entitiy with the given semantic type.
func (e *encoder) ent(ty semantic.Type) *entity {
	return e.entities.ty(ty)
}

// ty returns the entity for the given semantic type. ty will panic if there is
// no entitiy declared for the given type.
func (e *entities) ty(ty semantic.Type) *entity {
	ent, ok := e.types[ty]
	if !ok {
		panic(fmt.Errorf("No entity for type %T %v", ty, ty))
	}
	return ent
}

// build builds all the entities.
func (e *entities) build(enc *encoder) {
	c := enc.C

	// Declare all the entities as a first pass. This is required as we may need
	// to build cyclic graphs of proto descriptors.
	e.declBuiltins(c)
	e.declEnums(c)
	e.declPointers(c)
	e.declReferences(c)
	e.declSlices(c)
	e.declClasses(c)
	e.declMaps(c)
	e.declStaticArrays(c)
	e.declPseudonyms(c)
	e.declState(c)

	// Set labels (if they haven't specified one already) to optional.
	for _, ty := range e.types {
		if ty.label == 0 {
			ty.label = descriptor.FieldDescriptorProto_LABEL_OPTIONAL
		}
	}

	// Build all the entities and their proto descriptors.
	e.buildReferences(c)
	e.buildClasses(c)
	e.buildMaps(c)
	e.buildState(c)
	e.buildFunctions(c)

	// Build the encode functions
	e.buildEncodeTypeFuncs(c, enc.callbacks.encodeType)
}

// declBuiltins creates an entity for each semantic.Builtin type.
func (e *entities) declBuiltins(c *compiler.C) {
	for _, ty := range semantic.BuiltinTypes {
		ent := &entity{
			node: ty,
		}
		switch {
		case semantic.IsInteger(ty):
			ent.protoTy = descriptor.FieldDescriptorProto_TYPE_SINT64
			ent.wireTy = proto.WireVarint
		case ty == semantic.Float32Type:
			ent.protoTy = descriptor.FieldDescriptorProto_TYPE_FLOAT
			ent.wireTy = proto.WireFixed32
		case ty == semantic.Float64Type:
			ent.protoTy = descriptor.FieldDescriptorProto_TYPE_DOUBLE
			ent.wireTy = proto.WireFixed64
		case ty == semantic.BoolType:
			ent.protoTy = descriptor.FieldDescriptorProto_TYPE_SINT32
			ent.wireTy = proto.WireVarint
		case ty == semantic.StringType:
			ent.protoTy = descriptor.FieldDescriptorProto_TYPE_STRING
			ent.wireTy = proto.WireBytes
		case ty == semantic.InvalidType,
			ty == semantic.VoidType,
			ty == semantic.MessageType,
			ty == semantic.AnyType:
			continue
		default:
			c.Fail("Unsupported builtin type: %T %v", ty, ty)
		}
		e.types[ty] = ent
	}
}

// declEnums creates an entity for each API enum type.
// declEnums needs to be called after declBuiltins.
func (e *entities) declEnums(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Enums {
			e.types[ty] = e.ty(ty.NumberType)
		}
	}
}

// declPointers creates an entity for each API pointer type.
func (e *entities) declPointers(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Pointers {
			e.types[ty] = &entity{
				node:    ty,
				protoTy: descriptor.FieldDescriptorProto_TYPE_SINT64,
				wireTy:  proto.WireVarint,
			}
		}
	}
}

// declPointers stubs an entity for each API reference type.
func (e *entities) declReferences(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.References {
			e.types[ty] = &entity{
				node:    ty,
				desc:    &descriptor.DescriptorProto{},
				protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
				wireTy:  proto.WireBytes,
				fqn:     fmt.Sprintf("%v.%v", api.Name(), serialization.ProtoTypeName(ty)),
			}
		}
	}
}

// declSlices creates a single entity for all slices, which is shared for each
// API slice type.
func (e *entities) declSlices(c *compiler.C) {
	u64 := entity{
		protoTy: descriptor.FieldDescriptorProto_TYPE_UINT64,
		wireTy:  proto.WireVarint,
		label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL,
	}
	u32 := entity{
		protoTy: descriptor.FieldDescriptorProto_TYPE_UINT32,
		wireTy:  proto.WireVarint,
		label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL,
	}
	e.slice = &entity{
		node: &semantic.Slice{},
		desc: &descriptor.DescriptorProto{
			Name: proto.String("Slice"),
			Field: []*descriptor.FieldDescriptorProto{
				u64.protoDescField("root", 1),
				u64.protoDescField("base", 2),
				u64.protoDescField("size", 3),
				u64.protoDescField("count", 4),
				u32.protoDescField("pool", 5),
			},
		},
		protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		wireTy:  proto.WireBytes,
		fqn:     "memory.Slice",
	}
	for _, api := range c.APIs {
		for _, ty := range api.Slices {
			e.types[ty] = e.slice
		}
	}
}

// declClasses stubs an entity for each API class type.
func (e *entities) declClasses(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Classes {
			e.types[ty] = &entity{
				node:    ty,
				desc:    &descriptor.DescriptorProto{},
				protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
				wireTy:  proto.WireBytes,
				fqn:     fmt.Sprintf("%v.%v", api.Name(), serialization.ProtoTypeName(ty)),
			}
		}
	}
}

// declMaps stubs an entity for each API map type.
func (e *entities) declMaps(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Maps {
			e.types[ty] = &entity{
				node:    ty,
				protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
				wireTy:  proto.WireBytes,
				fqn:     fmt.Sprintf("%v.%v", api.Name(), serialization.ProtoTypeName(ty)),
			}
		}
	}
}

// declStaticArrays creates an entity for each API static array type.
// declStaticArrays must be called after all other API types are declared
// (except for pseudonyms).
func (e *entities) declStaticArrays(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.StaticArrays {
			valTi := e.ty(semantic.Underlying(ty.ValueType))
			ent := &entity{
				node:    ty,
				label:   descriptor.FieldDescriptorProto_LABEL_REPEATED,
				wireTy:  valTi.wireTy,
				protoTy: valTi.protoTy,
				desc:    valTi.desc,
				fqn:     valTi.fqn,
			}
			if ent.isPacked() {
				ent.wireTy = proto.WireBytes
			}
			e.types[ty] = ent
		}
	}
}

// declPseudonyms adds an entity reference to the underlying datatype for each
// API pseudonyms.
// declPseudonyms must be called after all other API types are declared.
func (e *entities) declPseudonyms(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Pseudonyms {
			if underlying := semantic.Underlying(ty.To); underlying != semantic.VoidType {
				e.types[ty] = e.ty(underlying)
			}
		}
	}
}

// declState stubs an entity for each API state object.
// declState must be called after all API types are declared.
func (e *entities) declState(c *compiler.C) {
	for _, api := range c.APIs {
		e.state[api] = &entity{
			node:    api,
			protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
			wireTy:  proto.WireBytes,
			fqn:     fmt.Sprintf("%v.State", api.Name()),
		}
	}
}

// buildReferences builds the proto descriptor for every API reference type.
// buildReferences must be called after all entities are declared.
func (e *entities) buildReferences(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.References {
			*e.ty(ty).desc = descriptor.DescriptorProto{
				Name: proto.String(serialization.ProtoTypeName(ty)),
				Field: []*descriptor.FieldDescriptorProto{
					e.ty(semantic.Uint64Type).protoDescField("ReferenceID", serialization.RefRef),
					e.ty(ty.To).protoDescField("Value", serialization.RefVal),
				},
			}
		}
	}
}

// buildClasses builds the proto descriptor for every API class type.
// buildClasses must be called after all entities are declared.
func (e *entities) buildClasses(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Classes {
			fields := []*descriptor.FieldDescriptorProto{}
			for i, f := range ty.Fields {
				fields = append(fields,
					e.ty(f.Type).protoDescField(f.Name(), serialization.ClassFieldStart+serialization.ProtoFieldID(i)),
				)
			}
			*e.ty(ty).desc = descriptor.DescriptorProto{
				Name:  proto.String(serialization.ProtoTypeName(ty)),
				Field: fields,
			}
		}
	}
}

// buildMaps builds the proto descriptor for every API map type.
// buildMaps must be called after all entities are declared.
func (e *entities) buildMaps(c *compiler.C) {
	for _, api := range c.APIs {
		for _, ty := range api.Maps {
			valTI := *e.ty(ty.ValueType)
			valTI.label = descriptor.FieldDescriptorProto_LABEL_REPEATED
			keyTI := *e.ty(ty.KeyType)
			keyTI.label = descriptor.FieldDescriptorProto_LABEL_REPEATED
			e.ty(ty).desc = &descriptor.DescriptorProto{
				Name: proto.String(serialization.ProtoTypeName(ty)),
				Field: []*descriptor.FieldDescriptorProto{
					e.ty(semantic.Int64Type).protoDescField("ReferenceID", serialization.MapRef),
					valTI.protoDescField("Values", serialization.MapVal),
					keyTI.protoDescField("Keys", serialization.MapKey),
				},
			}
		}
	}
}

// buildState builds the proto descriptor for the API state object.
// buildState must be called after all entities are declared.
func (e *entities) buildState(c *compiler.C) {
	for _, api := range c.APIs {
		fields := []*descriptor.FieldDescriptorProto{}
		for i, g := range encodeableGlobals(api) {
			fields = append(fields,
				e.ty(g.Type).protoDescField(g.Name(), serialization.StateStart+serialization.ProtoFieldID(i)),
			)
		}
		e.state[api].desc = &descriptor.DescriptorProto{
			Name:  proto.String("State"),
			Field: fields,
		}
	}
}

// buildFunctions builds the proto descriptors for all command parameter and
// call messages.
// buildFunctions must be called after all entities are declared.
func (e *entities) buildFunctions(c *compiler.C) {
	for _, api := range c.APIs {
		for _, f := range api.Functions {
			{ // Parameters
				fields := []*descriptor.FieldDescriptorProto{
					e.ty(semantic.Uint64Type).protoDescField("thread", serialization.CmdThread),
				}
				for i, p := range f.CallParameters() {
					fields = append(fields,
						e.ty(p.Type).protoDescField(p.Name(), serialization.CmdFieldStart+serialization.ProtoFieldID(i)),
					)
				}
				e.funcParams[f] = &entity{
					desc: &descriptor.DescriptorProto{
						Name:  proto.String(f.Name()),
						Field: fields,
					},
					fqn: fmt.Sprintf("%v.%v", api.Name(), f.Name()),
				}
			}
			if f.Return.Type != semantic.VoidType {
				name := f.Name() + "Call"
				e.funcCalls[f] = &entity{
					desc: &descriptor.DescriptorProto{
						Name: &name,
						Field: []*descriptor.FieldDescriptorProto{
							e.ty(f.Return.Type).protoDescField("result", serialization.CmdResult),
						},
					},
					fqn: fmt.Sprintf("%v.%v", api.Name(), name),
				}
			}
		}
	}
}

// allWithDescriptor returns all the entities that have a proto descriptor.
func (e *entities) allWithDescriptor(apis []*semantic.API) []*entity {
	out := []*entity{}
	seen := map[*entity]bool{}
	add := func(ent *entity) {
		if !seen[ent] {
			seen[ent] = true
			if ent.desc != nil {
				out = append(out, ent)
			}
		}
	}
	for _, api := range apis {
		add(e.state[api])
	}
	for _, ent := range e.types {
		add(ent)
	}
	for _, ent := range e.funcParams {
		add(ent)
	}
	for _, ent := range e.funcCalls {
		add(ent)
	}
	return out
}

type slice struct {
	offset int
	size   int
}

// createDescriptors marshals the proto descriptors and stores these into a
// single, packed global byte buffer. createDescriptors returns the global
// holding all the marshalled proto descriptors and a map that locates the
// entity's descriptor in the packed global.
func (e *entities) createDescriptors(c *compiler.C, l []*entity) (codegen.Global, map[*entity]slice) {
	slices := make(map[*entity]slice, len(l))

	datas := [][]byte{}
	for _, ent := range l {
		data, err := proto.Marshal(ent.desc)
		if err != nil {
			c.Fail("Could not encode proto desc %v: %v", ent.desc.Name, err)
		}
		slices[ent] = slice{size: len(data)}
		datas = append(datas, data)
	}

	packed, indices := data.Dedupe(datas)

	for _, ent := range l {
		s := slices[ent]
		s.offset = indices[0]
		slices[ent] = s
		indices = indices[1:]
	}

	global := c.M.Global("proto-descriptors", c.M.Scalar(packed)).
		LinkPrivate()

	return global, slices
}

// buildEncodeTypeFuncs declares all the functions used to encode entity types.
// The encode function will declare the entity's type and all transitive
// entity types.
func (e *entities) buildEncodeTypeFuncs(c *compiler.C, encodeType *codegen.Function) {
	l := e.allWithDescriptor(c.APIs)

	descs, descSlices := e.createDescriptors(c, l)

	// impls is a map of type mangled name to the public implementation of the
	// encode_type function.
	// This is used to deduplicate types that have the same underlying type when
	// lowered.
	impls := map[string]*entity{}
	for _, ent := range l {
		// Arrays share an entity with the element. Seperate the function names.
		ext := ""
		if ent.label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			ext = "_array"
		}
		name := fmt.Sprintf("encode_type_%s%s", ent.fqn, ext)

		// Check whether this is the first time we've seen this lowered type.
		if impl, seen := impls[name]; seen {
			ent.encodeType = impl.encodeType // Reuse the encodeType impl.
		} else {
			// First time we've seen this lowered type. Declare the encode_type
			// function.
			f := c.M.Function(c.T.Uint32, name, c.T.CtxPtr).LinkPrivate()
			ent.encodeType = f
			impls[name] = ent
		}
	}

	// Note: This is intentionally split into two passes to allow cyclic
	// encodes.

	for _, ent := range impls {
		c.Build(ent.encodeType, func(s *compiler.S) {
			encoder := s.Parameter(0)

			descSlice := descSlices[ent]
			ptr := descs.Value(s.Builder).Index(0, descSlice.offset)
			signedTypeID := s.Call(encodeType,
				encoder,
				s.Scalar(ent.fqn),
				s.Scalar(uint32(descSlice.size)),
				ptr)

			newType := s.GreaterOrEqualTo(signedTypeID, s.Scalar(int64(0)))
			typeID := s.Select(newType, signedTypeID, s.Negate(signedTypeID))

			s.If(newType, func(s *compiler.S) {
				// Encode dependent types.
				deps := []*entity{}
				seen := map[*entity]bool{}
				consider := func(ent *entity) {
					if seen[ent] {
						return
					}
					seen[ent] = true
					if ent.desc != nil {
						deps = append(deps, ent)
					}
				}

				switch ty := ent.node.(type) {
				case *semantic.API:
					for _, g := range encodeableGlobals(ty) {
						consider(e.ty(g.Type))
					}

				case *semantic.Class:
					for _, f := range ty.Fields {
						consider(e.ty(f.Type))
					}
				case *semantic.Reference:
					consider(e.ty(ty.To))
				case *semantic.StaticArray:
					consider(e.ty(ty.ValueType))
				case *semantic.Map:
					consider(e.ty(ty.KeyType))
					consider(e.ty(ty.ValueType))
				case *semantic.Function:
					for _, p := range ty.FullParameters {
						consider(e.ty(p.Type))
					}
				}

				for _, ent := range deps {
					s.Call(ent.encodeType, encoder)
				}
			})

			s.Return(typeID.Cast(c.T.Uint32))
		})
	}
}
