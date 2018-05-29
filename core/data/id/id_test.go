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
	sampleID = id.ID{
		0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x00,
		0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x00,
	}
	sampleIDString = "000123456789abcdef00" + "000123456789abcdef00"
	quotedSampleID = `"` + sampleIDString + `"`
)

func TestIDToString(t *testing.T) {
	assert := assert.To(t)
	str := sampleID.String()
	assert.For("str").That(str).Equals(sampleIDString)
}

func TestIDFormat(t *testing.T) {
	assert := assert.To(t)
	str := fmt.Sprint(sampleID)
	assert.For("id").That(str).Equals(sampleIDString)
}

func TestParseID(t *testing.T) {
	assert := assert.To(t)
	id, err := id.Parse(sampleIDString)
	assert.For("err").ThatError(err).Succeeded()
	assert.For("id").That(id).Equals(sampleID)
}

func TestParseTooLongID(t *testing.T) {
	assert := assert.To(t)
	_, err := id.Parse(sampleIDString + "00")
	assert.For("err").ThatError(err).Failed()
}

func TestParseTruncatedID(t *testing.T) {
	assert := assert.To(t)
	_, err := id.Parse(sampleIDString[:len(sampleIDString)-2])
	assert.For("err").ThatError(err).Failed()
}

func TestParseInvalidID(t *testing.T) {
	assert := assert.To(t)
	_, err := id.Parse("abcdefghijklmnopqrs")
	assert.For("err").ThatError(err).Failed()
}

func TestValid(t *testing.T) {
	assert := assert.To(t)
	assert.For("ID{}").That(id.ID{}.IsValid()).Equals(false)
	assert.For("sampleID").That(sampleID.IsValid()).Equals(true)
}

func TestOfBytes(t *testing.T) {
	assert := assert.To(t)
	id := id.OfBytes([]byte{0x00, 0x01, 0x02, 0x03})
	assert.For("id").ThatString(id).Equals("a02a05b025b928c039cf1ae7e8ee04e7c190c0db")
}

func TestOfString(t *testing.T) {
	assert := assert.To(t)
	id := id.OfString("Test\n")
	assert.For("id").ThatString(id).Equals("1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea")
}

func TestMarshalJSON(t *testing.T) {
	assert := assert.To(t)
	data, err := json.Marshal(sampleID)
	assert.For("err").ThatError(err).Succeeded()
	assert.For("data").ThatString(data).Equals(quotedSampleID)
}

func TestUnarshalJSON(t *testing.T) {
	assert := assert.To(t)
	id := id.ID{}
	err := json.Unmarshal([]byte(quotedSampleID), &id)
	assert.For("err").ThatError(err).Succeeded()
	assert.For("id").That(id).Equals(sampleID)
}

func TestInvalidUnarshalJSON(t *testing.T) {
	assert := assert.To(t)
	id := id.ID{}
	err := json.Unmarshal([]byte("0"), &id)
	assert.For("err").ThatError(err).Failed()
}

func TestUnique(t *testing.T) {
	assert := assert.To(t)
	id1 := id.Unique()
	id2 := id.Unique()
	assert.For("id1").That(id1.IsValid()).Equals(true)
	assert.For("id2").That(id2.IsValid()).Equals(true)
	assert.For("not-eq").That(id1).DeepNotEquals(id2)
}
