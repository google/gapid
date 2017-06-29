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

package database

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/gapid/core/context/keys"
)

// resolveChain is a value that is stored in the context of a Resolve().
// It holds a pointer to the record that is currently being resolved, and a
// pointer to the parent resolve (if any exists). This forms a chain of resolves
// that can be walked for displaying resolve stack traces.
type resolveChain struct {
	record *record
	parent *resolveChain
}

type resolveChainKeyTy string

var resolveChainKey resolveChainKeyTy = "<database.resolveChain>"

func getResolveChain(ctx context.Context) *resolveChain {
	if v := ctx.Value(resolveChainKey); v != nil {
		return v.(*resolveChain)
	}
	return nil
}

func (c *resolveChain) bind(ctx context.Context) context.Context {
	return keys.WithValue(ctx, resolveChainKey, c)
}

// callstack is a stack of program counters.
type callstack []uintptr

func (c callstack) String() string {
	lines := []string{}
	frames := runtime.CallersFrames(([]uintptr)(c))
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		lines = append(lines, fmt.Sprintf("%v:%v  %v", frame.File, frame.Line, frame.Function))
	}
	return strings.Join(lines, "\n")
}

func getCallstack(skip int) callstack {
	const stackLimit = 10
	callers := make([]uintptr, stackLimit)
	count := runtime.Callers(skip, callers)
	return callstack(callers[:count])
}

type rethrownPanic string

func (p rethrownPanic) Error() string { return string(p) }

// resolvePanicHandler catches and rethrows panics to add resolve chain
// information to the error message.
func (d *memory) resolvePanicHandler(ctx context.Context) {
	err := recover()
	switch err.(type) {
	case nil:
		return
	case rethrownPanic:
		panic(err)
	default:
		d.mutex.Lock()
		defer d.mutex.Unlock()

		buf := &bytes.Buffer{}
		for c := getResolveChain(ctx); c != nil; c = c.parent {
			r := c.record
			fmt.Fprintln(buf)
			fmt.Fprintf(buf, "--- %T ---\n", r.object)
			fmt.Fprintln(buf, indent(fmt.Sprintf("%+v", r.object), 1))
			fmt.Fprintf(buf, " Store():\n")
			fmt.Fprintln(buf, indent(r.created.String(), 2))
			fmt.Fprintln(buf)
			for i, c := range r.resolveState.callstacks {
				if i >= 10 {
					fmt.Fprintf(buf, " ... %d more Build() calls (truncated)\n", len(r.resolveState.callstacks)-i-1)
					break
				}
				fmt.Fprintf(buf, " Build() #%d:\n", i)
				fmt.Fprintln(buf, indent(c.String(), 2))
			}
		}

		panic(rethrownPanic(buf.String()))
	}
}

func indent(s string, depth int) string {
	i := strings.Repeat(" ", depth)
	return i + strings.Replace(s, "\n", "\n"+i, -1)
}
