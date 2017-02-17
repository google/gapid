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

package binaryxml

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestBinaryXmlDecodingAndEncoding(t *testing.T) {
	ctx := assert.Context(t)
	for _, fn := range []string{
		"testdata/manifest1.binxml",
		"testdata/manifest2.binxml",
		"testdata/manifest3.binxml",
		"testdata/manifest4.binxml",
		"testdata/manifest5.binxml",
		"testdata/manifest6.binxml",
	} {
		originalData, err := ioutil.ReadFile(fn)
		assert.With(ctx).ThatError(err).Succeeded()

		tree, err := decodeXmlTree(bytes.NewReader(originalData))
		assert.With(ctx).ThatError(err).Succeeded()
		assert.With(ctx).ThatSlice(tree.encode()).Equals(originalData)
	}
}
