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

package pack

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// Dynamic is a dynamic proto message.
type Dynamic struct {
	Desc   *descriptor.DescriptorProto
	Fields map[string]interface{}
	types  *types
}

func (Dynamic) ProtoMessage() {}

func (d *Dynamic) Reset() {
	d.Fields = map[string]interface{}{}
}

func (d Dynamic) String() string {
	return fmt.Sprintf("%+v", d)
}

// Format implements fmt.Formatter to print the full value chain.
func (d Dynamic) Format(f fmt.State, r rune) {
	switch r {
	case 'v':
		fields := []string{}
		for n, v := range d.Fields {
			suffix := ""
			if bytes, ok := v.([]byte); ok && len(bytes) > 32 {
				suffix = fmt.Sprintf(" (truncated %v bytes)", len(bytes))
				v = bytes[:32]
			}
			if f.Flag('+') {
				fields = append(fields, fmt.Sprintf("%v: %+v%v", n, v, suffix))
			} else {
				fields = append(fields, fmt.Sprintf("%v%v", v, suffix))
			}
		}
		// TODO: we should sort the fields before formatting, so that we can exit out early, as soon
		// as the requested number of chars has been reached.
		sort.Strings(fields)
		joined := strings.Join(fields, ", ")
		if width, hasWidth := f.Width(); hasWidth && len(joined) > width {
			joined = joined[0:width] + "..."
		}
		fmt.Fprintf(f, "%vᵈ{%s}", d.Desc.GetName(), joined)
	}
}

func newDynamic(desc *descriptor.DescriptorProto, types *types) *Dynamic {
	return &Dynamic{
		Desc:   desc,
		Fields: map[string]interface{}{},
		types:  types,
	}
}

func (d *Dynamic) Unmarshal(data []byte) error {
	buf := proto.NewBuffer(data)
	fields := map[int]*descriptor.FieldDescriptorProto{}
	for _, f := range d.Desc.GetField() {
		fields[int(f.GetNumber())] = f
	}

	for {
		u, err := buf.DecodeVarint()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
		wire, tag := int(u&0x7), int(u>>3)
		if tag <= 0 {
			return fmt.Errorf("illegal tag %d (wire type %d)", tag, wire)
		}

		val, err := d.read(wire, buf)
		if err != nil {
			return err
		}

		f, ok := fields[tag]
		if !ok {
			continue
		}

		wire = d.wireType(f.GetType())
		if f.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED && wire != proto.WireBytes {
			buf := proto.NewBuffer(val.([]byte))
			arr := []interface{}{}
			for {
				val, err := d.read(wire, buf)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}
				if err != nil {
					return err
				}
				el, err := d.unpack(val, f)
				if err != nil {
					return err
				}
				arr = append(arr, el)
			}
			val = arr
		} else if f.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			arr, _ := d.Fields[f.GetName()].([]interface{})
			if val, err = d.unpack(val, f); err != nil {
				return err
			}
			val = append(arr, val)
		} else {
			if val, err = d.unpack(val, f); err != nil {
				return err
			}
		}

		d.Fields[f.GetName()] = val
	}
	return nil
}

func (d *Dynamic) wireType(t descriptor.FieldDescriptorProto_Type) int {
	switch t {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return proto.WireFixed64

	case descriptor.FieldDescriptorProto_TYPE_FLOAT,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return proto.WireFixed32

	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_BOOL,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		return proto.WireVarint

	default:
		return proto.WireBytes
	}
}

func (d *Dynamic) read(wire int, buf *proto.Buffer) (interface{}, error) {
	switch wire {
	case proto.WireVarint:
		return buf.DecodeVarint()
	case proto.WireFixed64:
		return buf.DecodeFixed64()
	case proto.WireFixed32:
		return buf.DecodeFixed32()
	case proto.WireBytes:
		return buf.DecodeRawBytes(true)
	case proto.WireStartGroup, proto.WireEndGroup:
		return nil, fmt.Errorf("wire groups are deprecated and not supported by dynamic")
	default:
		return nil, fmt.Errorf("unknown wire type %v", wire)
	}
}

func (d *Dynamic) unpack(data interface{}, f *descriptor.FieldDescriptorProto) (interface{}, error) {
	switch f.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return math.Float64frombits(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return math.Float32frombits((uint32)(data.(uint64))), nil
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return (int64)(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return data, nil
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return (int32)(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return data, nil
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return (uint32)(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return data, nil
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return string(data.([]byte)), nil
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		return nil, fmt.Errorf("wire groups are deprecated and not supported by dynamic")
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return data, nil
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return uint32(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return data, nil
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return (int32)(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return (int64)(data.(uint64)), nil
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return (int32)(decodeZigzag(data.(uint64))), nil
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return (int64)(decodeZigzag(data.(uint64))), nil
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		tyname := strings.TrimLeft(f.GetTypeName(), ".")
		if subty, ok := d.types.byName[tyname]; ok {
			msg := subty.create()
			if err := proto.Unmarshal(data.([]byte), msg); err != nil {
				return nil, err
			}
			return msg, nil
		}
		return nil, fmt.Errorf("Unknown type: %v", tyname)
	}
	return nil, fmt.Errorf("Unknown field type: %v", f.GetType())
}

func decodeZigzag(x uint64) uint64 {
	return (x >> 1) ^ uint64((int64(x&1)<<63)>>63)
}
