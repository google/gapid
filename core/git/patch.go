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
	"bytes"
	"context"
	"fmt"
	"strings"
)

// GetPatch returns the changes between the two SHAs as a patch string.
func (g Git) GetPatch(ctx context.Context, from, to SHA) (string, error) {
	str, _, err := g.run(ctx, "format-patch", "--stdout", fmt.Sprintf("%v..%v", from, to))
	if err != nil {
		return "", err
	}
	return str, nil
}

// CanApplyPatch returns true if the specified patch can be applied to HEAD
// without conficts.
func (g Git) CanApplyPatch(ctx context.Context, patch string) (bool, error) {
	stdin := bytes.NewBuffer([]byte(patch))
	_, stderr, err := g.runWithStdin(ctx, stdin, "apply", "--check")
	if err != nil {
		for _, substr := range []string{
			"No such file or directory",
			"patch failed",
			"which does not match the current contents",
		} {
			if strings.Contains(stderr, substr) {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
