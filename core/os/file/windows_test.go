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

//go:build windows

package file_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/file"
)

func TestJunction(t *testing.T) {
	assert := assert.To(t)
	tmp, err := ioutil.TempDir("", "junction")
	if !assert.For("TempDir").ThatError(err).Succeeded() {
		return
	}
	defer os.RemoveAll(tmp)
	lnk := file.Abs(tmp).Join("lnk")
	tgt := file.Abs(tmp).Join("tgt")
	if err := file.Mkdir(tgt); !assert.For("Mkdir(tgt)").ThatError(err).Succeeded() {
		return
	}
	assert.For("Non-existant isJunction").That(file.IsJunction(lnk)).Equals(false)
	if err := file.Junction(lnk, tgt); !assert.For("Junction").ThatError(err).Succeeded() {
		return
	}
	assert.For("New junction isJunction").That(file.IsJunction(lnk)).Equals(true)

	testDataA := []byte("test data A")
	testDataB := []byte("test data B")

	err = ioutil.WriteFile(tgt.Join("made-in.tgt").System(), testDataA, 0666)
	assert.For("Create file in tgt").ThatError(err).Succeeded()
	err = ioutil.WriteFile(lnk.Join("made-in.lnk").System(), testDataB, 0666)
	assert.For("Create file in lnk").ThatError(err).Succeeded()

	var data []byte
	data, err = ioutil.ReadFile(tgt.Join("made-in.lnk").System())
	assert.For("File in lnk err").ThatError(err).Succeeded()
	assert.For("File in lnk data").ThatSlice(data).Equals(testDataB)
	data, err = ioutil.ReadFile(lnk.Join("made-in.tgt").System())
	assert.For("File in tgt err").ThatError(err).Succeeded()
	assert.For("File in tgt data").ThatSlice(data).Equals(testDataA)
}
