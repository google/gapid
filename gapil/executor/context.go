// Copyright (C) 2018 Google Inc.
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

package executor

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

type contextEnvKeyTy string

const contextEnvKey = contextEnvKeyTy("env")

// PutEnv attaches a Env to a Context.
func PutEnv(ctx context.Context, e *Env) context.Context {
	return keys.WithValue(ctx, contextEnvKey, e)
}

// GetEnv retrieves the Env from a context previously annotated by PutEnv.
func GetEnv(ctx context.Context) *Env {
	val := ctx.Value(contextEnvKey)
	if val == nil {
		panic(string(contextEnvKey) + " not present")
	}
	return val.(*Env)
}
