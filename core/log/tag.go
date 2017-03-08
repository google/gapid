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

package log

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

type tagKeyTy string

const tagKey tagKeyTy = "log.tagKey"

// PutTag returns a new context with the tag assigned to w.
func PutTag(ctx context.Context, w string) context.Context {
	return keys.WithValue(ctx, tagKey, w)
}

// GetTag returns the Tag assigned to ctx.
func GetTag(ctx context.Context) string {
	out, _ := ctx.Value(tagKey).(string)
	return out
}
