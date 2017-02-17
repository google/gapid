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

package file_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/file"
)

func TestPathContains(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		dir      file.Path
		file     file.Path
		expected bool
	}{
		{
			dir:      file.Abs("foo").Join("bar"),
			file:     file.Abs("foo").Join("bar", "cat"),
			expected: true,
		}, {
			dir:      file.Abs("foo").Join("bar"),
			file:     file.Abs("foo").Join("bar"),
			expected: false,
		}, {
			dir:      file.Abs("foo").Join("bar"),
			file:     file.Abs("foo").Join("nom"),
			expected: false,
		}, {
			dir:      file.Abs("foo").Join("bar", "cat"),
			file:     file.Abs("foo").Join("bar"),
			expected: false,
		},
	} {
		assert.For("").Compare(test.dir, "contains", test.file).
			Test(test.dir.Contains(test.file) == test.expected)
	}
}
