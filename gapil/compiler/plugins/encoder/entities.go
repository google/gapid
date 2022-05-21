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

	// repeated indicates that this entity is encoded as a repeated field
	repeated bool

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
func (t *entity) isPacked() bool {
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
func (t *entity) protoDescField(name string, id serialization.ProtoFieldID) *descriptor.FieldDescriptorProto {
	var typename *string
	if t.fqn != "" {
		t := "." + t.fqn
		typename = &t
	}
	label := descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	if t.repeated {
		label = descriptor.FieldDescriptorProto_LABEL_REPEATED
	}
	return &descriptor.FieldDescriptorProto{
		Name:     &name,
		JsonName: proto.String(cases.Snake(name).ToCamel()),
		Number:   proto.Int32(int32(id)),
		Label:    &label,
		Type:     &t.protoTy,
		TypeName: typename,
	}
}

// hasEntity returns true if the given type is present.
func (e *encoder) hasEntity(ty semantic.Type) bool {
	_, ok := e.entities.types[ty]
	return ok
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

// buildTypes builds all the entities.
func (e *entities) buildTypes(enc *encoder) {
	c := enc.C

	// Declare all the entities as a first pass. This is required as we may need
	// to build cyclic graphs of proto descriptors.
	e.declBuiltins(c)
	e.declEnums(c)
	e.declPointers(c)
	e.declSlices(c)

	// Build all the entities and their proto descriptors.
	e.buildState(c)
	e.buildExtras(c)
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

// declSlices creates a single entity for all slices, which is shared for each
// API slice type.
func (e *entities) declSlices(c *compiler.C) {
	u64 := entity{
		protoTy: descriptor.FieldDescriptorProto_TYPE_UINT64,
		wireTy:  proto.WireVarint,
	}
	u32 := entity{
		protoTy: descriptor.FieldDescriptorProto_TYPE_UINT32,
		wireTy:  proto.WireVarint,
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

// buildType builds the proto descriptor for the given type, if it has not been built already.
func (e *entities) buildType(c *compiler.C, api string, ty semantic.Type) *entity {
	if entity, ok := e.types[ty]; ok {
		return entity
	}

	switch ty := ty.(type) {
	case *semantic.Class:
		return e.buildClass(c, api, ty)
	case *semantic.Map:
		return e.buildMap(c, api, ty)
	case *semantic.Pseudonym:
		return e.buildPseudonym(c, api, ty)
	case *semantic.Reference:
		return e.buildReference(c, api, ty)
	case *semantic.StaticArray:
		return e.buildStaticArray(c, api, ty)
	default:
		panic(fmt.Sprintf("Unexpected semantic node type: %T", ty))
	}
}

// buildClass builds the proto descriptor for the given class and all its members.
func (e *entities) buildClass(c *compiler.C, api string, ty *semantic.Class) *entity {
	entity := &entity{
		node:    ty,
		desc:    &descriptor.DescriptorProto{},
		protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		wireTy:  proto.WireBytes,
		fqn:     fmt.Sprintf("%v.%v", api, serialization.ProtoTypeName(ty)),
	}
	// Assign type before recursing on fields to allow cycles.
	e.types[ty] = entity

	fields := []*descriptor.FieldDescriptorProto{}
	for i, f := range ty.Fields {
		fields = append(fields,
			e.buildType(c, api, f.Type).protoDescField(f.Name(), serialization.ClassFieldStart+serialization.ProtoFieldID(i)),
		)
	}
	entity.desc = &descriptor.DescriptorProto{
		Name:  proto.String(serialization.ProtoTypeName(ty)),
		Field: fields,
	}
	return entity
}

// buildMap builds the proto descriptor for the given map, as well as its key and value types.
func (e *entities) buildMap(c *compiler.C, api string, ty *semantic.Map) *entity {
	entity := &entity{
		node:    ty,
		protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		wireTy:  proto.WireBytes,
		fqn:     fmt.Sprintf("%v.%v", api, serialization.ProtoTypeName(ty)),
	}
	// Assign type before recursing to allow cycles.
	e.types[ty] = entity

	// Note that we're making copies of the entities here, since we'll encode the key and values
	// as repeated, but only for the maps, not elsewhere these types may be used.
	keyTI := *e.buildType(c, api, ty.KeyType)
	valTI := *e.buildType(c, api, ty.ValueType)
	keyTI.repeated = true
	valTI.repeated = true

	entity.desc = &descriptor.DescriptorProto{
		Name: proto.String(serialization.ProtoTypeName(ty)),
		Field: []*descriptor.FieldDescriptorProto{
			e.ty(semantic.Int64Type).protoDescField("ReferenceID", serialization.MapRef),
			valTI.protoDescField("Values", serialization.MapVal),
			keyTI.protoDescField("Keys", serialization.MapKey),
		},
	}
	return entity
}

// buildPseudonym builds the proto descriptor for the underlying type and uses it for the
// pseudonym type as well.
func (e *entities) buildPseudonym(c *compiler.C, api string, ty *semantic.Pseudonym) *entity {
	if underlying := semantic.Underlying(ty.To); underlying != semantic.VoidType {
		entity := e.buildType(c, api, underlying)
		e.types[ty] = entity
		return entity
	}
	return nil
}

// buildReference builds the proto descriptor for the given reference type.
func (e *entities) buildReference(c *compiler.C, api string, ty *semantic.Reference) *entity {
	entity := &entity{
		node:    ty,
		desc:    &descriptor.DescriptorProto{},
		protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		wireTy:  proto.WireBytes,
		fqn:     fmt.Sprintf("%v.%v", api, serialization.ProtoTypeName(ty)),
	}
	// Assign type before recursing on fields to allow cycles.
	e.types[ty] = entity

	to := e.buildType(c, api, ty.To)

	entity.desc = &descriptor.DescriptorProto{
		Name: proto.String(serialization.ProtoTypeName(ty)),
		Field: []*descriptor.FieldDescriptorProto{
			e.ty(semantic.Uint64Type).protoDescField("ReferenceID", serialization.RefRef),
			to.protoDescField("Value", serialization.RefVal),
		},
	}
	return entity
}

// buildStaticArray builds the proto descriptor for the given static array type.
func (e *entities) buildStaticArray(c *compiler.C, api string, ty *semantic.StaticArray) *entity {
	valTi := e.buildType(c, api, semantic.Underlying(ty.ValueType))
	ent := &entity{
		node:     ty,
		repeated: true,
		wireTy:   valTi.wireTy,
		protoTy:  valTi.protoTy,
		desc:     valTi.desc,
		fqn:      valTi.fqn,
	}
	if ent.isPacked() {
		ent.wireTy = proto.WireBytes
	}
	e.types[ty] = ent
	return ent
}

// buildState builds the proto descriptor for the API state and all types referenced from it.
func (e *entities) buildState(c *compiler.C) {
	for _, api := range c.APIs {
		fields := []*descriptor.FieldDescriptorProto{}
		for i, g := range encodeableGlobals(api) {
			fields = append(fields,
				e.buildType(c, api.Name(), g.Type).protoDescField(g.Name(), serialization.StateStart+serialization.ProtoFieldID(i)),
			)
		}
		e.state[api] = &entity{
			node:    api,
			protoTy: descriptor.FieldDescriptorProto_TYPE_MESSAGE,
			wireTy:  proto.WireBytes,
			fqn:     fmt.Sprintf("%v.State", api.Name()),
			desc: &descriptor.DescriptorProto{
				Name:  proto.String("State"),
				Field: fields,
			},
		}
	}
}

// buildExtras builds the proto descriptor for the serialized command extras.
func (e *entities) buildExtras(c *compiler.C) {
	for _, api := range c.APIs {
		for _, class := range api.Classes {
			if class.Annotations.GetAnnotation("serialize") != nil {
				e.buildClass(c, api.Name(), class)
			}
		}
	}
}

// buildFunctions builds the proto descriptors for all command parameter and
// call messages.
func (e *entities) buildFunctions(c *compiler.C) {
	for _, api := range c.APIs {
		for _, f := range api.Functions {
			{ // Parameters
				fields := []*descriptor.FieldDescriptorProto{
					e.ty(semantic.Uint64Type).protoDescField("thread", serialization.CmdThread),
				}
				for i, p := range f.CallParameters() {
					fields = append(fields,
						e.buildType(c, api.Name(), p.Type).protoDescField(p.Name(), serialization.CmdFieldStart+serialization.ProtoFieldID(i)),
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
				ret := e.buildType(c, api.Name(), f.Return.Type)
				name := f.Name() + "Call"
				e.funcCalls[f] = &entity{
					desc: &descriptor.DescriptorProto{
						Name: &name,
						Field: []*descriptor.FieldDescriptorProto{
							ret.protoDescField("result", serialization.CmdResult),
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
		if ent.repeated {
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
