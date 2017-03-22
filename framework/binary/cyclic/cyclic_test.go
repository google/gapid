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

package cyclic_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/test"
	"github.com/google/gapid/framework/binary/vle"
)

const (
	aa = uint32(123)
	bb = uint32(321)
	cc = uint32(456)
)

var testObjects = []test.Entry{
	{
		Name:   "Nil",
		Values: []binary.Object{nil},
		Data:   []byte{0},
	},
	{
		Name:   "One",
		Values: []binary.Object{test.ObjectA},
		Data: test.Bytes{}.Add(
			0x03, // object sid + encoded
			0x03, // type sid + encoded
		).Add(test.EntityA...).Add(
			0x07,
			'O', 'b', 'j', 'e', 'c', 't', 'A',
		).Data,
	},
	{
		Name:   "Repeat",
		Values: []binary.Object{test.ObjectA, test.ObjectA},
		Data: test.Bytes{}.Add(
			0x03, // object sid + encoded
			0x03, // type sid + encoded
		).Add(test.EntityA...).Add(
			0x07,
			'O', 'b', 'j', 'e', 'c', 't', 'A',

			0x02, // repeated object sid
		).Data,
	},
	{
		Name:   "Simple",
		Values: []binary.Object{test.ObjectC},
		Data: test.Bytes{}.Add(
			0x03, // object sid + encoded
			0x03, // type sid + encoded
		).Add(test.EntityC...).Add(
			0x03,
		).Data,
	},
	{
		Name:   "Many",
		Values: []binary.Object{test.ObjectA, test.ObjectB, test.ObjectA, nil},
		Data: test.Bytes{}.Add(
			0x03, // object sid + encoded
			0x03, // type sid + encoded
		).Add(test.EntityA...).Add(
			0x07,
			'O', 'b', 'j', 'e', 'c', 't', 'A',

			0x05, // object sid + encoded
			0x05, // type sid + encoded
		).Add(test.EntityB...).Add(
			0x07,
			'O', 'b', 'j', 'e', 'c', 't', 'B',

			0x02, // repeated object sid

			0x00, // nil object sid
		).Data,
	},
}

var testFull = test.Entry{
	Name:   "Full",
	Values: []binary.Object{test.ObjectA},
	Data: test.Bytes{}.Add(
		0x01, // control object
		0x00, // control version
		0x01, // control data
		0x03, // object sid + encoded
		0x03, // type sid + encoded
	).Add(test.EntityAFull...).Add(
		0x07,
		'O', 'b', 'j', 'e', 'c', 't', 'A',
	).Data,
}

func EncodeObject(ctx context.Context, entry test.Entry, e binary.Encoder, buf *bytes.Buffer) {
	ctx = log.Enter(ctx, entry.Name)
	e.SetMode(binary.Compact)
	for _, o := range entry.Values {
		e.Object(o)
		if e.Error() != nil {
			log.F(ctx, "Object gave unexpected error: %v", e.Error())
		}
	}
	test.VerifyData(ctx, entry, buf)
}

func DecodeObject(ctx context.Context, entry test.Entry, d binary.Decoder, reader *bytes.Reader) {
	ctx = log.Enter(ctx, entry.Name)
	for i, o := range entry.Values {
		ctx := log.V{"index": i}.Bind(ctx)
		got := d.Object()
		if d.Error() != nil {
			log.F(ctx, "Object gave unexpected error: %v", d.Error())
		}
		assert.With(ctx).That(got).DeepEquals(o)
	}
}

func EncodeAndDecode(ctx context.Context, obj binary.Object) {
	b := &bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(b))
	e.Object(obj)
	if e.Error() != nil {
		log.F(ctx, "%v", e.Error())
	}
	d := cyclic.Decoder(vle.Reader(b))
	got := d.Object()
	if d.Error() != nil {
		log.F(ctx, "%v", d.Error())
	}
	assert.With(ctx).That(got).DeepEquals(obj)
}

func TestObject(t *testing.T) {
	ctx := log.Testing(t)
	for _, entry := range testObjects {
		b := &bytes.Buffer{}
		EncodeObject(ctx, entry, cyclic.Encoder(vle.Writer(b)), b)
		r := bytes.NewReader(entry.Data)
		DecodeObject(ctx, entry, cyclic.Decoder(vle.Reader(r)), r)
	}
}

func TestFull(t *testing.T) {
	ctx := log.Testing(t)
	b := &bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(b))
	e.SetMode(binary.Full)
	for i, o := range testFull.Values {
		ctx := log.V{"index": i}.Bind(ctx)
		e.Object(o)
		assert.For(ctx, "encode").ThatError(e.Error()).Succeeded()
	}
	test.VerifyData(ctx, testFull, b)
	d := cyclic.Decoder(vle.Reader(b))
	for i, o := range testFull.Values {
		ctx := log.V{"index": i}.Bind(ctx)
		got := d.Object()
		assert.For(ctx, "decode").ThatError(d.Error()).Succeeded()
		assert.For(ctx, "decode").That(got).DeepEquals(o)
	}
}

func TestError(t *testing.T) {
	ctx := log.Testing(t)
	for _, entry := range testObjects {
		if len(entry.Data) <= 2 {
			continue
		}
		ctx := log.Enter(ctx, entry.Name)
		e := cyclic.Encoder(vle.Writer(&test.LimitedWriter{Limit: 1}))
		e.SetMode(binary.Compact)
		for _, o := range entry.Values {
			e.Object(o)
		}
		assert.For(ctx, "short write").ThatError(e.Error()).Equals(io.ErrShortWrite)
	}
}

func TestBad(t *testing.T) {
	ctx := log.Testing(t)
	b := &bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(b))
	e.SetMode(binary.Compact)
	defer func() {
		assert.With(ctx).That(recover()).Equals("Class() not implemented")
	}()
	e.Object(test.BadObject)
}

func TestInvalidControl(t *testing.T) {
	ctx := log.Testing(t)
	b := &test.Bytes{Data: []byte{0x01, 0x7f, 0x00}}
	d := cyclic.Decoder(vle.Reader(b))
	d.Object()
	assert.With(ctx).ThatError(d.Error()).HasMessage("⦕Invalid control block version⦖")
}

func TestNested(t *testing.T) {
	ctx := log.Testing(t)
	leaf := test.Leaf{A: aa}
	EncodeAndDecode(ctx, &leaf)
	anon := test.Anonymous{Leaf: test.Leaf{A: aa}}
	EncodeAndDecode(ctx, &anon)
	contains := test.Contains{LeafField: test.Leaf{A: aa}}
	EncodeAndDecode(ctx, &contains)
	leafSlice := []test.Leaf{{A: aa}, {A: bb}}
	slice := test.Slice{Leaves: leafSlice}
	EncodeAndDecode(ctx, &slice)
	emptySlice := test.Slice{}
	EncodeAndDecode(ctx, &emptySlice)
	emptyMapKey := test.MapKey{}
	EncodeAndDecode(ctx, &emptyMapKey)
	mapKey := test.MapKey{M: map[test.Leaf]uint32{test.Leaf{A: aa}: bb}}
	EncodeAndDecode(ctx, &mapKey)
	mapKeyValue := test.MapKeyValue{
		M: map[test.Leaf]test.Leaf{test.Leaf{A: aa}: {A: bb}}}
	EncodeAndDecode(ctx, &mapKeyValue)
	sliceInMap := test.SliceInMap{M: map[uint32][]test.Leaf{aa: leafSlice}}
	EncodeAndDecode(ctx, &sliceInMap)
	emptySliceInMap := test.SliceInMap{M: map[uint32][]test.Leaf{bb: nil}}
	EncodeAndDecode(ctx, &emptySliceInMap)
	sliceOfSlices := test.SliceOfSlices{
		Slice: [][]test.Leaf{leafSlice, {{A: cc}}, nil, leafSlice}}
	EncodeAndDecode(ctx, &sliceOfSlices)
	mapInSlice := test.MapInSlice{
		Slice: []test.Uint32ːuint32{{cc: bb}, {aa: bb, bb: cc}}}
	EncodeAndDecode(ctx, &mapInSlice)
	mapOfMaps := test.MapOfMaps{M: test.Uint32ːLeafːLeaf{
		aa: {leaf: leaf}, cc: {test.Leaf{A: bb}: test.Leaf{A: aa}}}}
	EncodeAndDecode(ctx, &mapOfMaps)
	leafArray := [3]test.Leaf{{A: aa}, {A: bb}, {A: cc}}
	array := test.Array{Leaves: leafArray}
	EncodeAndDecode(ctx, &array)
	mapInArray :=
		test.MapInArray{Array: [2]test.Uint32ːuint32{{
			cc: bb, aa: cc}, {bb: cc}}}
	EncodeAndDecode(ctx, &mapInArray)
	arrayInMap := test.ArrayInMap{M: test.Uint32ː3_Leaf{bb: leafArray}}
	EncodeAndDecode(ctx, &arrayInMap)
	arrayOfArrays := test.ArrayOfArrays{
		Array: [2][3]test.Leaf{leafArray, leafArray}}
	EncodeAndDecode(ctx, &arrayOfArrays)
	ca := test.Contains{LeafField: test.Leaf{A: aa}}
	cb := test.Contains{LeafField: test.Leaf{A: bb}}
	cc := test.Contains{LeafField: test.Leaf{A: cc}}

	complex := test.Complex{
		SliceMapArray: []test.Containsː3_Contains{
			{ca: {cb, cc, ca}},
		},
		SliceArrayMap: [][3]test.ContainsːContains{
			{{ca: cb}, {cb: cc, ca: cc}, {cb: cc}},
		},
		ArraySliceMap: [3][]test.ContainsːContains{
			{{ca: cb}, {cb: cc, ca: cc}, {cb: cc}},
			{{cb: cc, ca: cc}, {ca: cb}, {cb: cc}},
			{{ca: cb}, {cb: cc, ca: cc}, {cb: cc}},
		},
		ArrayMapSlice: [3]test.ContainsːSliceContains{
			{ca: {ca, cb, cc}},
			{cb: {cc, cb}},
			{cc: {cb, ca}},
		},
		MapArraySlice: test.Containsː3_SliceContains{
			ca: {{ca, cb, cc}, {ca, cb}, {cc}},
		},
		MapSliceArray: test.ContainsːSlice3_Contains{
			ca: {{ca, cb, cc}, {cc, cb, ca}},
		},
	}
	EncodeAndDecode(ctx, &complex)
}

func TestUnknownTypeError(t *testing.T) {
	ctx := log.Testing(t)
	d := cyclic.Decoder(vle.Reader(bytes.NewBuffer([]byte{
		0x03, // object sid + encoded
		0x03, // type sid + encoded
		0x0C, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x09,
		'B', 'a', 'd', 'O', 'b', 'j', 'e', 'c', 't',
	})))
	d.AllowDynamic = false
	d.Object()
	assert.With(ctx).ThatError(d.Error()).Failed()
}
