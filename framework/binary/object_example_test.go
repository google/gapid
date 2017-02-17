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

package binary_test

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/framework/binary/vle"
	"github.com/google/gapid/core/data/id"
)

type ExampleObject struct{ Data string }
type ExampleClass struct{}

var ExampleID = id.ID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14}
var ExampleEntity = &binary.Entity{
	Package:  "binary_test",
	Identity: "ExampleObject",
	Fields: []binary.Field{
		{Declared: "Data", Type: &schema.Primitive{Name: "string", Method: schema.String}},
	},
}

func (*ExampleObject) Class() binary.Class {
	return (*ExampleClass)(nil)
}

func (*ExampleClass) Schema() *binary.Entity {
	return ExampleEntity
}

func (*ExampleClass) ID() id.ID {
	return ExampleID
}

func (*ExampleClass) New() binary.Object {
	return &ExampleObject{}
}

func (*ExampleClass) Encode(e binary.Encoder, obj binary.Object) {
	o := obj.(*ExampleObject)
	e.String(o.Data)
}

func (*ExampleClass) Decode(d binary.Decoder) binary.Object {
	o := &ExampleObject{}
	o.Data = d.String()
	return o
}

func (*ExampleClass) DecodeTo(d binary.Decoder, obj binary.Object) {
	o := obj.(*ExampleObject)
	o.Data = d.String()
}

func init() {
	registry.Global.Add((*ExampleObject)(nil).Class())
}

// This example shows how to write a type with custom encode and decode
// methods, and send it over a "connection"
func Example_object() {
	// Create a connected input and output stream for example purposes.
	in := io.Reader(&bytes.Buffer{})
	out := in.(io.Writer)

	// Build an encoder and decoder on top of the stream
	e := cyclic.Encoder(vle.Writer(out))
	d := cyclic.Decoder(vle.Reader(in))

	// Encode an object onto the stream
	e.Object(&ExampleObject{"MyObject"})
	if e.Error() != nil {
		log.Fatalf("Encode gave unexpected error: %v", e.Error())
	}

	// Read the object back
	o := d.Object()
	if d.Error() == nil {
		fmt.Printf("read %q\n", o.(*ExampleObject).Data)
	} else {
		log.Fatalf("Decode gave unexpected error: %v", d.Error())
	}

	// Output:
	// read "MyObject"
}
