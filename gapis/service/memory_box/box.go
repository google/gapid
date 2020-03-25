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

package memory_box

import (
	"context"
	"reflect"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/service/types"
)

var (
	tyLinkable = reflect.TypeOf((*path.Linker)(nil)).Elem()
)

func Box(ctx context.Context, d *memory.Decoder, t *types.Type, p path.Node, rc *path.ResolveConfig) (*Value, error) {
	a, err := t.Alignment(ctx, d.MemoryLayout())
	if err != nil {
		return nil, err
	}
	d.Align(uint64(a))
	typeId := t.TypeId
	switch t := t.Ty.(type) {
	case *types.Type_Pod:
		switch t.Pod {
		case pod.Type_uint:
			v := d.Uint()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint{
							Uint: uint64(v),
						},
					},
				}}, d.Error()
		case pod.Type_sint:
			v := d.Int()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint{
							Sint: int64(v),
						},
					},
				}}, d.Error()
		case pod.Type_uint8:
			v := d.U8()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint8{
							Uint8: uint32(v),
						},
					},
				}}, d.Error()
		case pod.Type_sint8:
			v := d.I8()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint8{
							Sint8: int32(v),
						},
					},
				}}, d.Error()
		case pod.Type_uint16:
			v := d.U16()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint16{
							Uint16: uint32(v),
						},
					},
				}}, d.Error()
		case pod.Type_sint16:
			v := d.I16()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint16{
							Sint16: int32(v),
						},
					},
				}}, d.Error()
		case pod.Type_uint32:
			v := d.U32()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint32{
							Uint32: v,
						},
					},
				}}, d.Error()
		case pod.Type_sint32:
			v := d.I32()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint32{
							Sint32: v,
						},
					},
				}}, d.Error()
		case pod.Type_uint64:
			v := d.U64()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint64{
							Uint64: v,
						},
					},
				}}, d.Error()
		case pod.Type_sint64:
			v := d.I64()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint64{
							Sint64: v,
						},
					},
				}}, d.Error()
		case pod.Type_bool:
			v := d.Bool()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Bool{
							Bool: v,
						},
					},
				}}, d.Error()
		case pod.Type_string:
			v := d.String()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_String_{
							String_: v,
						},
					},
				}}, d.Error()
		case pod.Type_float32:
			v := d.F32()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Float32{
							Float32: v,
						},
					},
				}}, d.Error()
		case pod.Type_float64:
			v := d.F64()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Float64{
							Float64: v,
						},
					},
				}}, d.Error()
		}
	case *types.Type_Pointer:
		v := d.Pointer()
		return &Value{
			Val: &Value_Pointer{
				Pointer: &Pointer{
					Address: v,
				},
			}}, d.Error()
	case *types.Type_Struct:
		s := &Struct{
			Fields: []*Value{},
		}

		for _, f := range t.Struct.Fields {
			elem, ok := types.TryGetType(f.Type)
			if !ok {
				return nil, log.Err(ctx, nil, "Incomplete type in struct box")
			}

			v, err := Box(ctx, d, elem, p, rc)
			if err != nil {
				return nil, err
			}
			s.Fields = append(s.Fields, v)
		}
		d.Align(uint64(a))
		return &Value{
			Val: &Value_Struct{
				Struct: s,
			}}, nil
	case *types.Type_Sized:
		switch t.Sized {
		case types.SizedType_sized_int:
			v := d.Int()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Sint64{
							Sint64: int64(v),
						},
					},
				}}, d.Error()
		case types.SizedType_sized_uint:
			v := d.Uint()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint64{
							Uint64: uint64(v),
						},
					},
				}}, d.Error()
		case types.SizedType_sized_size:
			v := d.Size()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint64{
							Uint64: uint64(v),
						},
					},
				}}, d.Error()
		case types.SizedType_sized_char:
			v := d.Char()
			return &Value{
				Val: &Value_Pod{
					Pod: &pod.Value{
						Val: &pod.Value_Uint8{
							Uint8: uint32(v),
						},
					},
				}}, d.Error()
		}
	case *types.Type_Pseudonym:
		if elem, ok := types.TryGetType(t.Pseudonym.Underlying); ok {
			b, err := Box(ctx, d, elem, p, rc)
			if err != nil {
				return nil, err
			}

			reflType, err := types.GetReflectedType(typeId)
			if err != nil {
				return nil, err
			}

			if reflType.Implements(tyLinkable) {
				// reflectively convert the underlying value back to what its
				// api type should have been, then attempt to link through it.
				v := reflect.New(reflType)
				switch vv := b.Val.(*Value_Pod).Pod.Val.(type) {
				case (*pod.Value_Uint64):
					v.Elem().SetUint(vv.Uint64)
				case (*pod.Value_Uint32):
					v.Elem().SetUint(uint64(vv.Uint32))
				}

				if res, err := v.Interface().(path.Linker).Link(ctx, p, rc); err == nil {
					b.Link = res.Path()
				}
			}

			return b, nil
		}
	case *types.Type_Array:
		if elem, ok := types.TryGetType(t.Array.ElementType); ok {
			s := &Array{
				Entries: []*Value{},
			}
			for i := uint64(0); i < t.Array.Size; i++ {
				v, err := Box(ctx, d, elem, p, rc)
				if err != nil {
					return nil, err
				}
				s.Entries = append(s.Entries, v)
			}
			return &Value{
				Val: &Value_Array{
					Array: s,
				},
			}, nil
		}
	case *types.Type_Enum:
		if elem, ok := types.TryGetType(t.Enum.Underlying); ok {
			return Box(ctx, d, elem, p, rc)
		}
	case *types.Type_Map:
		return nil, log.Err(ctx, nil, "Cannot decode map from memory")
	case *types.Type_Reference:
		return nil, log.Err(ctx, nil, "Cannot decode refs from memory")
	case *types.Type_Slice:
		return nil, log.Err(ctx, nil, "Cannot decode slices from memory")
	}
	return nil, log.Err(ctx, nil, "Unhandled box type")
}
