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

package text_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/gapid/core/text"
)

func TestLimit(t *testing.T) {
	buf := &bytes.Buffer{}
	for _, test := range []struct {
		limit  int
		writes string
		expect string
	}{
		{8, "123", "123"},
		{8, "1234", "1234"},
		{8, "12345", "12345"},
		{8, "123456", "123456"},
		{8, "1234567", "1234567"},
		{8, "12345678", "12345678"},
		{8, "123456789", "12345abc"},
		{8, "1234567890123", "12345abc"},
		{8, "12|34", "1234"},
		{8, "123|45", "12345"},
		{8, "1234|56", "123456"},
		{8, "12345|67", "1234567"},
		{8, "12345|", "12345"},
		{8, "123456|78", "12345678"},
		{8, "1234567|89", "12345abc"},
		{8, "12345678|90123", "12345abc"},
		{8, "123456789|0123", "12345abc"},
	} {
		buf.Reset()
		writer := text.NewLimitWriter(buf, 8, "abc")
		for _, v := range strings.Split(test.writes, "|") {
			writer.Write(([]byte)(v))
		}
		writer.Flush()
		got := buf.String()
		if got != test.expect {
			t.Errorf("Expected %q got %q", test.expect, got)
		}
	}
}
