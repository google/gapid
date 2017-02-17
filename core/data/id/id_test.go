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

package id_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
)

var (
	sampleId = id.ID{
		0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x00,
		0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x00,
	}
	sampleIdString = "000123456789abcdef00" + "000123456789abcdef00"
	quotedSampleID = `"` + sampleIdString + `"`
)

func TestIDToString(t *testing.T) {
	ctx := assert.Context(t)
	str := sampleId.String()
	assert.With(ctx).That(str).Equals(sampleIdString)
}

func TestIDFormat(t *testing.T) {
	ctx := assert.Context(t)
	str := fmt.Sprint(sampleId)
	assert.With(ctx).That(str).Equals(sampleIdString)
}

func TestParseID(t *testing.T) {
	ctx := assert.Context(t)
	id, err := id.Parse(sampleIdString)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(id).Equals(sampleId)
}

func TestParseTooLongID(t *testing.T) {
	ctx := assert.Context(t)
	_, err := id.Parse(sampleIdString + "00")
	assert.With(ctx).ThatError(err).Failed()
}

func TestParseTruncatedID(t *testing.T) {
	ctx := assert.Context(t)
	_, err := id.Parse(sampleIdString[:len(sampleIdString)-2])
	assert.With(ctx).ThatError(err).Failed()
}

func TestParseInvalidID(t *testing.T) {
	ctx := assert.Context(t)
	_, err := id.Parse("abcdefghijklmnopqrst")
	assert.With(ctx).ThatError(err).Failed()
}

func TestValid(t *testing.T) {
	ctx := assert.Context(t)
	assert.With(ctx).That(id.ID{}.IsValid()).Equals(false)
	assert.With(ctx).That(sampleId.IsValid()).Equals(true)
}

func TestOfBytes(t *testing.T) {
	ctx := assert.Context(t)
	id := id.OfBytes([]byte{0x00, 0x01, 0x02, 0x03})
	assert.With(ctx).ThatString(id).Equals("a02a05b025b928c039cf1ae7e8ee04e7c190c0db")
}

func TestOfString(t *testing.T) {
	ctx := assert.Context(t)
	id := id.OfString("Test\n")
	assert.With(ctx).ThatString(id).Equals("1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea")
}

func TestMarshalJSON(t *testing.T) {
	ctx := assert.Context(t)
	data, err := json.Marshal(sampleId)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatString(data).Equals(quotedSampleID)
}

func TestUnarshalJSON(t *testing.T) {
	ctx := assert.Context(t)
	id := id.ID{}
	err := json.Unmarshal([]byte(quotedSampleID), &id)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(id).Equals(sampleId)
}

func TestInvalidUnarshalJSON(t *testing.T) {
	ctx := assert.Context(t)
	id := id.ID{}
	err := json.Unmarshal([]byte("0"), &id)
	assert.With(ctx).ThatError(err).Failed()
}

func TestUnique(t *testing.T) {
	ctx := assert.Context(t)
	id1 := id.Unique()
	id2 := id.Unique()
	assert.For(ctx, "id1").That(id1.IsValid()).Equals(true)
	assert.For(ctx, "id2").That(id2.IsValid()).Equals(true)
	assert.With(ctx).That(id1).DeepNotEquals(id2)
}
