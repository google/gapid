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

package text

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/math/sint"
)

// LineNumber prefixes returns s prefixed with a line number for each line,
// starting from 1. Useful for adding line numbers to source.
func LineNumber(s string) string {
	lines := strings.Split(s, "\n")
	width := sint.Log10(len(lines)) + 1
	fa := fmt.Sprintf("%%%dd:", width)
	fb := fmt.Sprintf("%%%dd: %%s", width)
	for i, l := range lines {
		if len(l) == 0 {
			lines[i] = fmt.Sprintf(fa, i+1)
		} else {
			lines[i] = fmt.Sprintf(fb, i+1, l)
		}
	}
	return strings.Join(lines, "\n")
}
