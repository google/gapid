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

package schema

import "github.com/google/gapid/framework/binary"

type ObjectClass binary.Entity

func (c *ObjectClass) Name() string { return (*binary.Entity)(c).Name() }

func (c *ObjectClass) New() binary.Object { return &Object{Type: c} }

func (c *ObjectClass) Schema() *binary.Entity { return (*binary.Entity)(c) }

func (c *ObjectClass) Encode(e binary.Encoder, object binary.Object) {
	o := object.(*Object)
	for i, f := range c.Fields {
		f.Type.EncodeValue(e, o.Fields[i])
	}
}

func (c *ObjectClass) doDecode(d binary.Decoder, o *Object) {
	o.Fields = make([]interface{}, len(c.Fields))
	for i, f := range c.Fields {
		o.Fields[i] = f.Type.DecodeValue(d)
	}
}

func (c *ObjectClass) Decode(d binary.Decoder) binary.Object {
	o := &Object{Type: c}
	c.doDecode(d, o)
	return o
}

func (c *ObjectClass) DecodeTo(d binary.Decoder, object binary.Object) {
	c.doDecode(d, object.(*Object))
}
