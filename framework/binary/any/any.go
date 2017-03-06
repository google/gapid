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

// Package any contains Object wrappers for Plain-Old-Data types.
package any

import (
	"github.com/google/gapid/core/data/id"
	_ "github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/framework/binary"
	_ "github.com/google/gapid/framework/binary/registry"
	_ "github.com/google/gapid/framework/binary/schema"
)

// binary: java.source = rpclib
// binary: java.package = com.google.gapid.rpclib.any
// binary: java.indent = "    "
// binary: java.member_prefix = m

type object_ struct {
	binary.Generate `java:"ObjectBox"`
	Value           binary.Object
}

type id_ struct {
	binary.Generate
	Value id.ID
}

type bool_ struct {
	binary.Generate
	Value bool
}

type uint8_ struct {
	binary.Generate
	Value uint8
}

type int8_ struct {
	binary.Generate
	Value int8
}

type uint16_ struct {
	binary.Generate
	Value uint16
}

type int16_ struct {
	binary.Generate
	Value int16
}

type float32_ struct {
	binary.Generate
	Value float32
}

type uint32_ struct {
	binary.Generate
	Value uint32
}

type int32_ struct {
	binary.Generate
	Value int32
}

type float64_ struct {
	binary.Generate
	Value float64
}

type uint64_ struct {
	binary.Generate
	Value uint64
}

type int64_ struct {
	binary.Generate
	Value int64
}

type string_ struct {
	binary.Generate `java:"StringBox"`
	Value           string
}

func (v object_) Unbox() interface{}  { return v.Value }
func (v bool_) Unbox() interface{}    { return v.Value }
func (v uint8_) Unbox() interface{}   { return v.Value }
func (v int8_) Unbox() interface{}    { return v.Value }
func (v uint16_) Unbox() interface{}  { return v.Value }
func (v int16_) Unbox() interface{}   { return v.Value }
func (v float32_) Unbox() interface{} { return v.Value }
func (v uint32_) Unbox() interface{}  { return v.Value }
func (v int32_) Unbox() interface{}   { return v.Value }
func (v float64_) Unbox() interface{} { return v.Value }
func (v uint64_) Unbox() interface{}  { return v.Value }
func (v int64_) Unbox() interface{}   { return v.Value }
func (v string_) Unbox() interface{}  { return v.Value }

type objectSlice struct {
	binary.Generate
	Value []binary.Object
}

type idSlice struct {
	binary.Generate
	Value []id.ID
}

type boolSlice struct {
	binary.Generate
	Value []bool
}

type uint8Slice struct {
	binary.Generate
	Value []uint8
}

type int8Slice struct {
	binary.Generate
	Value []int8
}

type uint16Slice struct {
	binary.Generate
	Value []uint16
}

type int16Slice struct {
	binary.Generate
	Value []int16
}

type float32Slice struct {
	binary.Generate
	Value []float32
}

type uint32Slice struct {
	binary.Generate
	Value []uint32
}

type int32Slice struct {
	binary.Generate
	Value []int32
}

type float64Slice struct {
	binary.Generate
	Value []float64
}

type uint64Slice struct {
	binary.Generate
	Value []uint64
}

type int64Slice struct {
	binary.Generate
	Value []int64
}

type stringSlice struct {
	binary.Generate
	Value []string
}

func (v objectSlice) Unbox() interface{}  { return v.Value }
func (v boolSlice) Unbox() interface{}    { return v.Value }
func (v uint8Slice) Unbox() interface{}   { return v.Value }
func (v int8Slice) Unbox() interface{}    { return v.Value }
func (v uint16Slice) Unbox() interface{}  { return v.Value }
func (v int16Slice) Unbox() interface{}   { return v.Value }
func (v float32Slice) Unbox() interface{} { return v.Value }
func (v uint32Slice) Unbox() interface{}  { return v.Value }
func (v int32Slice) Unbox() interface{}   { return v.Value }
func (v float64Slice) Unbox() interface{} { return v.Value }
func (v uint64Slice) Unbox() interface{}  { return v.Value }
func (v int64Slice) Unbox() interface{}   { return v.Value }
func (v stringSlice) Unbox() interface{}  { return v.Value }

func boxer(v interface{}) binary.Object {
	switch v := v.(type) {
	case binary.Object:
		return &object_{Value: v}
	case id.ID:
		return &id_{Value: v}
	case bool:
		return &bool_{Value: v}
	case uint8:
		return &uint8_{Value: v}
	case int8:
		return &int8_{Value: v}
	case uint16:
		return &uint16_{Value: v}
	case int16:
		return &int16_{Value: v}
	case float32:
		return &float32_{Value: v}
	case uint32:
		return &uint32_{Value: v}
	case int32:
		return &int32_{Value: v}
	case float64:
		return &float64_{Value: v}
	case uint64:
		return &uint64_{Value: v}
	case int64:
		return &int64_{Value: v}
	case string:
		return &string_{Value: v}

	case []binary.Object:
		return &objectSlice{Value: v}
	case []id.ID:
		return &idSlice{Value: v}
	case []bool:
		return &boolSlice{Value: v}
	case []uint8:
		return &uint8Slice{Value: v}
	case []int8:
		return &int8Slice{Value: v}
	case []uint16:
		return &uint16Slice{Value: v}
	case []int16:
		return &int16Slice{Value: v}
	case []float32:
		return &float32Slice{Value: v}
	case []uint32:
		return &uint32Slice{Value: v}
	case []int32:
		return &int32Slice{Value: v}
	case []float64:
		return &float64Slice{Value: v}
	case []uint64:
		return &uint64Slice{Value: v}
	case []int64:
		return &int64Slice{Value: v}
	case []string:
		return &stringSlice{Value: v}
	default:
		return nil
	}
}

func init() {
	binary.RegisterBoxer(boxer)
}
