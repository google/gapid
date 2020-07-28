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

package git

import (
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestParseLog(t *testing.T) {
	assert := assert.To(t)
	str := `ǁa5eaebbe90d60b560e08a6750958755a58e6f999ǀBen Clayton <bclayton@google.com>ǀMerge pull request #157 from google/fix_freetypeǀUpdate freetype path and fix for breaking changes.
  ǁf3cdfb4d9b662e0d66b5779d02370115e136a008ǀBen ClaytonǀUpdate freetype path and fix for breaking changes.ǀ
  ǁ92ee22e53df7b361224670cf0f5ce4b12cdbc040ǀBen Clayton <bclayton@google.com>ǀMerge pull request #154 from google/recreate_ctrlsǀAdd recreateControls flag to the OnDataChanged list/tree event.
  ǁ73c5a5ba67edfd06d20f9f52249b06984a324d0cǀMr4xǀUpdated Grid Layoutǀ
  ǁ1d039c122e95122e68e5862aab3580bba15bd1adǀMr4xǀMerge pull request #1 from google/fooǀUpdate`
	expected := []ChangeList{
		{
			SHA:         SHA{0xa5, 0xea, 0xeb, 0xbe, 0x90, 0xd6, 0x0b, 0x56, 0x0e, 0x08, 0xa6, 0x75, 0x09, 0x58, 0x75, 0x5a, 0x58, 0xe6, 0xf9, 0x99},
			Author:      "Ben Clayton <bclayton@google.com>",
			Subject:     "Merge pull request #157 from google/fix_freetype",
			Description: "Update freetype path and fix for breaking changes.",
		},
		{
			SHA:     SHA{0xf3, 0xcd, 0xfb, 0x4d, 0x9b, 0x66, 0x2e, 0x0d, 0x66, 0xb5, 0x77, 0x9d, 0x02, 0x37, 0x01, 0x15, 0xe1, 0x36, 0xa0, 0x08},
			Author:  "Ben Clayton",
			Subject: "Update freetype path and fix for breaking changes.",
		},
		{
			SHA:         SHA{0x92, 0xee, 0x22, 0xe5, 0x3d, 0xf7, 0xb3, 0x61, 0x22, 0x46, 0x70, 0xcf, 0x0f, 0x5c, 0xe4, 0xb1, 0x2c, 0xdb, 0xc0, 0x40},
			Author:      "Ben Clayton <bclayton@google.com>",
			Subject:     "Merge pull request #154 from google/recreate_ctrls",
			Description: "Add recreateControls flag to the OnDataChanged list/tree event.",
		},
		{
			SHA:     SHA{0x73, 0xc5, 0xa5, 0xba, 0x67, 0xed, 0xfd, 0x06, 0xd2, 0x0f, 0x9f, 0x52, 0x24, 0x9b, 0x06, 0x98, 0x4a, 0x32, 0x4d, 0x0c},
			Author:  "Mr4x",
			Subject: "Updated Grid Layout",
		},
		{
			SHA:         SHA{0x1d, 0x03, 0x9c, 0x12, 0x2e, 0x95, 0x12, 0x2e, 0x68, 0xe5, 0x86, 0x2a, 0xab, 0x35, 0x80, 0xbb, 0xa1, 0x5b, 0xd1, 0xad},
			Author:      "Mr4x",
			Subject:     "Merge pull request #1 from google/foo",
			Description: "Update",
		},
	}
	cls, err := parseLog(str)
	assert.For("err").ThatError(err).Succeeded()
	assert.For("cls").ThatSlice(cls).Equals(expected)
}
