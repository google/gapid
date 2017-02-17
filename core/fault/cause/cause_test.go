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

package cause_test

import (
	"context"
	"testing"

	"strings"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/text/note"
)

const (
	anError      = fault.Const("Some message")
	anotherError = fault.Const("another")
)

func TestRoot(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	l1 := cause.Explain(ctx, anError, "Wrapped")
	l2 := cause.Explainf(ctx, l1, "And %s", "again")
	assert.For("root cause of unstructured error").ThatError(anError).HasCause(anError)
	assert.For("root cause of wrapped error").ThatError(l1).HasCause(anError)
	assert.For("root cause of nested error").ThatError(l2).HasCause(anError)
	assert.For("cause of nested error").ThatError(l2.Cause()).HasMessage(l1.Error())
}

func TestTranscribe(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = memo.Tag(ctx, "CauseTests")
	pure := cause.Explain(ctx, nil, "No underlying")
	detail := cause.Explain(ctx, anError, "with details").With("A detail", "Some value")
	l1 := cause.Explain(ctx, anError, "First wrapping")
	l2 := cause.Explain(ctx, l1, "Second wrapping")
	ctx = memo.Tag(ctx, "ATag")
	ctx = keys.WithValue(ctx, "A key", "Some details")
	complex := cause.Explainf(ctx, anotherError, "A %s explanation", "detailed")
	wrapped := cause.Explainf(ctx, complex, "wrapped")
	for _, test := range []struct {
		name       string
		err        cause.Error
		detailed   string
		canonoical string
	}{{
		name: "Pure",
		err:  pure,
		detailed: `
CauseTests
    ⦕No underlying⦖
`,
		canonoical: `Tag{Tag="CauseTests"},Cause{"No underlying"}`,
	}, {
		name: "detail",
		err:  detail,
		detailed: `
CauseTests:with details
    ⦕Some message⦖
    A detail = Some value
`,
		canonoical: `Tag{Tag="CauseTests"},Text{Text="with details"},Cause{"Some message"},Detail{A detail="Some value"}`,
	}, {
		name: "L1",
		err:  l1,
		detailed: `
CauseTests:First wrapping
    ⦕Some message⦖
`,
		canonoical: `Tag{Tag="CauseTests"},Text{Text="First wrapping"},Cause{"Some message"}`,
	}, {
		name: "L2",
		err:  l2,
		detailed: `
 CauseTests:Second wrapping
    CauseTests:First wrapping
        ⦕Some message⦖
`,
		canonoical: `Tag{Tag="CauseTests"},Text{Text="Second wrapping"},Cause{[Tag{Tag="CauseTests"},Text{Text="First wrapping"},Cause{"Some message"}]}`,
	}, {
		name: "Complex",
		err:  complex,
		detailed: `
ATag:A detailed explanation
    ⦕another⦖
    A key = Some details
`,
		canonoical: `Tag{Tag="ATag"},Text{Text="A detailed explanation"},Cause{"another"},Extra{A key="Some details"}`,
	}, {
		name: "Wrapped",
		err:  wrapped,
		detailed: `
ATag:wrapped
    ATag:A detailed explanation
        ⦕another⦖
        A key = Some details
    A key = Some details
`,
		canonoical: `Tag{Tag="ATag"},Text{Text="wrapped"},Cause{[Tag{Tag="ATag"},Text{Text="A detailed explanation"},Cause{"another"},Extra{A key="Some details"}]},Extra{A key="Some details"}`,
	}} {
		canonoical := strings.TrimSpace(test.canonoical)
		detailed := strings.TrimSpace(test.detailed)
		test.err.Page.Sort()
		assert.For("%s Canonical", test.name).That(note.Canonical.Print(test.err.Page)).Equals(canonoical)
		assert.For("%s Detail", test.name).That(note.Detailed.Print(test.err.Page)).Equals(detailed)
		if test.detailed == "" {
			continue
		}
	}
}
