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

package jot_test

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/text/note"
)

func TestJotter(t *testing.T) {
	assert := assert.To(t)
	j := jot.To(context.Background())
	assert.For("jot").
		That(note.Normal.Print(j.Clone().
			Jot("testing note").Page)).
		Equals("testing note")
	/*
				assert.For("wrap").
					That(note.Normal.Print(j.Clone().
						Wrap(fault.Const("testing wrap")).Page)).
					Equals("⦕testing wrap⦖")
		assert.For("explain").
			That(note.Normal.Print(j.Clone().
				Explain(fault.Const("testing explain"), "explanation").Page)).
			Equals("explanation:⦕testing explain⦖")
		assert.For("explainf").
			That(note.Normal.Print(j.Clone().
				Explainf(fault.Const("testing explainf"), "explanation %d", 15).Page)).
			Equals("explanation 15:⦕testing explainf⦖")
	*/
}
