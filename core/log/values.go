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

// values is a linked-list of key-value pairs.
type values struct {
	v      V
	parent *values
}

type valuesKeyTy string

const valuesKey valuesKeyTy = "log.valuesKey"

func getValues(ctx context.Context) *values {
	out, _ := ctx.Value(valuesKey).(*values)
	return out
}

// V is a map of key-value pairs.
// It can be associated with a context with Bind().
type V map[string]interface{}

// Bind returns a new context with V attached.
func (v V) Bind(ctx context.Context) context.Context {
	return keys.WithValue(ctx, valuesKey, &values{
		v:      v,
		parent: getValues(ctx),
	})
}
