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
	"context"
	"fmt"
	"strings"
)

// Log returns the top count ChangeList at HEAD.
func (g Git) Log(ctx context.Context, count int) ([]ChangeList, error) {
	return g.LogFrom(ctx, "HEAD", count)
}

// LogFrom returns the top count ChangeList starting from at.
func (g Git) LogFrom(ctx context.Context, at string, count int) ([]ChangeList, error) {
	if at == "" {
		at = "HEAD"
	}
	str, _, err := g.run(ctx, "log", at, "--pretty=format:ǁ%Hǀ%an <%ae>ǀ%sǀ%b", fmt.Sprintf("-%d", count), g.wd)
	if err != nil {
		return nil, err
	}
	return parseLog(str)
}

// Parent returns the parent ChangeList for cl.
func (g Git) Parent(ctx context.Context, cl ChangeList) (ChangeList, error) {
	str, _, err := g.run(ctx, "log", "--pretty=format:ǁ%Hǀ%an <%ae>ǀ%sǀ%b", fmt.Sprintf("%v^", cl.SHA))
	if err != nil {
		return ChangeList{}, err
	}
	cls, err := parseLog(str)
	if err != nil {
		return ChangeList{}, err
	}
	if len(cls) == 0 {
		return ChangeList{}, fmt.Errorf("Unexpected output")
	}
	return cls[0], nil
}

// HeadCL returns the HEAD ChangeList at the given commit/tag/branch.
func (g Git) HeadCL(ctx context.Context, at string) (ChangeList, error) {
	if at == "" {
		at = "HEAD"
	}
	cls, err := g.LogFrom(ctx, at, 1)
	if err != nil {
		return ChangeList{}, err
	}
	if len(cls) == 0 {
		return ChangeList{}, fmt.Errorf("No commits found")
	}
	return cls[0], nil
}

func parseLog(str string) ([]ChangeList, error) {
	msgs := strings.Split(str, "ǁ")
	cls := make([]ChangeList, 0, len(msgs))
	for _, s := range msgs {
		if parts := strings.Split(s, "ǀ"); len(parts) == 4 {
			cl := ChangeList{
				Author:      strings.TrimSpace(parts[1]),
				Subject:     strings.TrimSpace(parts[2]),
				Description: strings.TrimSpace(parts[3]),
			}
			if err := cl.SHA.Parse(parts[0]); err != nil {
				return nil, err
			}
			cls = append(cls, cl)
		}
	}
	return cls, nil
}
