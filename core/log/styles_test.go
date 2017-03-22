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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestStyles(t *testing.T) {
	for _, test := range testMessages {
		for _, s := range []struct {
			style    log.Style
			name     string
			expected string
		}{
			{log.Raw, "Raw", test.raw},
			{log.Brief, "Brief", test.brief},
			{log.Normal, "Normal", test.normal},
			{log.Detailed, "Detailed", test.detailed},
		} {
			w, b := log.Buffer()
			test.send(s.style.Handler(w))
			assert.To(t).
				For("%s(%s)", s.name, test.msg).
				ThatString(b.String()).Equals(s.expected)
		}
	}
}
