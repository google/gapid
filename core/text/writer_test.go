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
	"io"
	"testing"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text"
)

func TestWriter(t *testing.T) {
	got := []string{}
	w := text.Writer(func(s string) error {
		got = append(got, s)
		return nil
	})
	input := []string{
		"A short part",
		" of a long line\n",
		"And now",
		" a split line\nAnd another one too\n",
		"And finally",
		" fragments",
		" with no",
		" newlines",
	}
	expect := []string{
		"A short part of a long line",
		"And now a split line",
		"And another one too",
		"And finally fragments with no newlines",
	}
	for _, in := range input {
		io.WriteString(w, in)
	}
	w.Close()
	if len(got) != len(expect) {
		t.Errorf("Incorrect number of lines, got %d expected %d", len(got), len(expect))
	}
	for i, expect := range expect {
		if got[i] != expect {
			t.Errorf("Got %q expected %q", got[i], expect)
		}
	}
}

func TestFailWriter(t *testing.T) {
	limit := 2
	w := text.Writer(func(s string) error {
		limit--
		if limit < 0 {
			return fault.Const("Failed")
		}
		return nil
	})
	_, err := io.WriteString(w, "A two part string\nThat should be fine\n")
	if err != nil {
		t.Errorf("First write failed")
	}
	_, err = io.WriteString(w, "But the next one should fail\n")
	if err == nil {
		t.Errorf("Second write should have failed")
	}
}
