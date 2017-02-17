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

package output_test

import (
	"context"
	"testing"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/note"
)

func TestLog(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	l := &logger{}
	ctx = output.NewContext(ctx, output.Log(l, note.Normal))
	for _, test := range tests {
		testHandler(ctx, test)
		assert.For(test.severity.String()).That(l.out).Equals(test.expect)
		l.out = ""
	}
}

type logger struct {
	out string
}

func (l *logger) Output(calldepth int, s string) error {
	l.out += s
	return nil
}
