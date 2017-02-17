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

package template

import "testing"

func TestBitposWithNoBitsSet(t *testing.T) {
	in := uint32(0)
	expected := -1
	got := (&Functions{}).Bitpos(in)
	if got != expected {
		t.Errorf("Bitpos(%#v) returned unexpected value. Expected: %#v, got: %#v", in, expected, got)
	}
}

func TestBitposWithOneBitSet(t *testing.T) {
	for pos := uint(0); pos < 32; pos++ {
		in := uint32(1 << pos)
		expected := int(pos)
		got := (&Functions{}).Bitpos(in)
		if got != expected {
			t.Errorf("Bitpos(%#v) returned unexpected value. Expected: %#v, got: %#v", in, expected, got)
		}
	}
}

func TestBitposWithTwoBitsSet(t *testing.T) {
	in := uint32(0x8080)
	expected := -1
	got := (&Functions{}).Bitpos(in)
	if got != expected {
		t.Errorf("Bitpos(%#v) returned unexpected value. Expected: %#v, got: %#v", in, expected, got)
	}
}

func TestAsSignedOnUint8(t *testing.T) {
	in := ^uint8(0) - 7
	expected := int8(-8)
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int8); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnUint16(t *testing.T) {
	in := ^uint16(0) - 15
	expected := int16(-16)
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int16); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnUint32(t *testing.T) {
	in := ^uint32(0) - 31
	expected := int32(-32)
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int32); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnUint64(t *testing.T) {
	in := ^uint64(0) - 63
	expected := int64(-64)
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int64); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnInt8(t *testing.T) {
	in := int8(-8)
	expected := in
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int8); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnInt16(t *testing.T) {
	in := int16(-16)
	expected := in
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int16); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnInt32(t *testing.T) {
	in := int32(-32)
	expected := in
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int32); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}

func TestAsSignedOnInt64(t *testing.T) {
	in := int64(-64)
	expected := in
	got := (&Functions{}).AsSigned(in)
	if got, ok := got.(int64); !ok || got != expected {
		t.Errorf("AsSigned(%T %#v) returned unexpected result. Expected: %T %#v, got: %T %#v", in, in, expected, expected, got, got)
	}
}
