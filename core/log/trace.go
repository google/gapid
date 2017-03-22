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

// trace is a single entry in a stack of Enter()s.
type trace struct {
	name   string
	parent *trace
}

type traceKeyTy string

const traceKey traceKeyTy = "log.traceKey"

// Enter returns a new context with the trace-stack pushed by name.
func Enter(ctx context.Context, name string) context.Context {
	return keys.WithValue(ctx, traceKey, &trace{
		name:   name,
		parent: getTrace(ctx),
	})
}

func getTrace(ctx context.Context) *trace {
	out, _ := ctx.Value(traceKey).(*trace)
	return out
}

// GetTrace returns the trace-stack.
func GetTrace(ctx context.Context) []string {
	t := getTrace(ctx)
	if t == nil {
		return nil
	}
	out := []string{}
	for t != nil {
		out = append(out, t.name)
		t = t.parent
	}
	return out
}
