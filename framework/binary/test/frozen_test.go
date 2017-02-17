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

package test_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/framework/binary/test"
	"github.com/google/gapid/framework/binary/vle"
)

const (
	aa    = 10
	bb    = 20
	cc    = "Hello"
	begin = "Begin"
	end   = "End"
)

// encodeX_V1_Struct encodes X_V1{aa, bb} into the stream. As we have
// disabled the encoder for X_V1 this is a hand coded encoder used
// for the test. Should do the same as e.Struct(x1) if that was allowed.
func encodeX_V1_Struct(t *testing.T, e binary.Encoder) {
	e.Int32(aa)
	e.Int32(bb)
}

// encodeX_V1_Variant encodes X_V1{aa, bb} into the stream, as a variant.
// As we have disabled the encoder for X_V1 this is a hand coded encoder used
// for the test. Should do the same as e.Variant(x1) if that was allowed.
func encodeX_V1_Variant(t *testing.T, e binary.Encoder) {
	x1 := test.X_V1{}
	// TODO (should we allow encoding of a schema of a frozen class).
	e.Entity(x1.Class().Schema())
	encodeX_V1_Struct(t, e)
}

// encodeY_V1_Variant encodes a hypothetical past version of Y which contained
// an X_V1 subtype, as a variant. Shuold do the same as e.Variant(y1) if that
// was allowed.
func encodeY_V1_Variant(t *testing.T, e binary.Encoder) {
	// Build a schema object for a Y containing X1.
	schemaY1 := &binary.Entity{
		Package:  "test",
		Identity: "Y",
		Fields: []binary.Field{
			{Declared: "begin", Type: &schema.Primitive{Name: "string", Method: schema.String}},
			{Declared: "x", Type: &schema.Struct{Entity: (*test.X_V1)(nil).Class().Schema()}},
			{Declared: "end", Type: &schema.Primitive{Name: "string", Method: schema.String}},
		},
	}
	e.Entity(schemaY1)
	e.String(begin)
	encodeX_V1_Struct(t, e)
	e.String(end)
}

// The signature of the above encoder functions.
type EncoderFunc func(t *testing.T, e binary.Encoder)

// makeDecoderFromEncoder makes a decoder which can be used to decode
// the stream created by calling the given encoder function.
func makeDecoderFromEncoder(t *testing.T, encode EncoderFunc) binary.Decoder {
	buf := bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(&buf))
	encode(t, e)
	if err := e.Error(); err != nil {
		t.Fatal(err)
	}
	return cyclic.Decoder(vle.Reader(&buf))
}

// decoderForX make a decoder for a stream which can be upgraded into type X
func decoderForX(t *testing.T) binary.Decoder {
	return makeDecoderFromEncoder(t, encodeX_V1_Variant)
}

// decoderForX make a decoder for a stream which can be upgraded into type Y
func decoderForY(t *testing.T) binary.Decoder {
	return makeDecoderFromEncoder(t, encodeY_V1_Variant)
}

// TestEncodeX_V1 test that it is not possible to encode a frozen type
func TestEncodeX_V1(t *testing.T) {
	x1 := test.X_V1{A: aa, B: bb}
	buf := bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(&buf))
	e.Variant(&x1)
	if err := e.Error(); err == nil {
		t.Errorf("Expected error encoding X_V1")
	}
}

// TestDecodeX test that it is possible to decode a frozen type in a
// stream into the current version of the type.
func TestDecodeX(t *testing.T) {
	d := decoderForX(t)
	o := d.Variant()
	if err := d.Error(); err != nil {
		t.Fatalf("Error decoding X: %v", err)
	}
	x := o.(*test.X)
	expect := test.X{A: aa, B: bb, C: cc}
	if *x != expect {
		t.Errorf("Got %v Expected %v", x, expect)
	}
}

// TestDecodeY test that it is possible to decode a frozen type in a
// stream into the current version of a type which contains the current
// version of the type.
func TestDecodeY(t *testing.T) {
	d := decoderForY(t)
	o := d.Variant()
	if err := d.Error(); err != nil {
		t.Fatalf("Error decoding Y: %v", err)
	}
	y := o.(*test.Y)
	expect := test.Y{Begin: begin, X: test.X{A: aa, B: bb, C: cc}, End: end}
	if *y != expect {
		t.Errorf("Got %v Expected %v", y, expect)
	}
}
