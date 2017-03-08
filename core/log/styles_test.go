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

package log_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestStyles(t *testing.T) {
	var buf bytes.Buffer

	for _, test := range testMessages {
		for _, w := range []struct {
			handler  log.Handler
			name     string
			expected string
		}{
			{log.Raw.Handler(&buf, &buf), "Raw", test.raw},
			{log.Brief.Handler(&buf, &buf), "Brief", test.brief},
			{log.Normal.Handler(&buf, &buf), "Normal", test.normal},
			{log.Detailed.Handler(&buf, &buf), "Detailed", test.detailed},
		} {
			buf.Reset()
			test.send(w.handler)
			assert.To(t).
				For("%s(%s)", w.name, test.msg).
				ThatString(buf.String()).Equals(w.expected)
		}
	}
}
