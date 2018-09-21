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

package assert

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/google/gapid/core/log"
)

type (
	// Output matches the logging methods of the test host types.
	Output interface {
		Fatal(...interface{})
		Error(...interface{})
		Log(...interface{})
	}

	// Manager is the root of the fluent interface.
	// It wraps an assertion output target in something that can construct
	// assertion objects.
	// The output object is normally a testing.T
	Manager struct {
		out Output
	}

	ctxOutput struct{ ctx context.Context }
	stdOutput struct{}
)

// To creates an assertion manager using the target t for logging.
// t can be a context.Context, Output or nil to log to stdout.
func To(t interface{}) Manager {
	switch t := t.(type) {
	case nil:
		return Manager{stdOutput{}}
	case context.Context:
		return Manager{ctxOutput{t}}
	case Output:
		return Manager{t}
	default:
		panic(fmt.Errorf("Unsupported assertion target type %T", t))
	}
}

// For is shorthand for assert.To(t).For(msg, args...).
func For(t interface{}, msg string, args ...interface{}) *Assertion {
	return To(t).For(msg, args...)
}

// For starts a new assertion with the supplied title.
func (ctx Manager) For(msg string, args ...interface{}) *Assertion {
	a := &Assertion{
		to:    ctx.out,
		out:   &bytes.Buffer{},
		level: Error,
	}
	a.Printf(msg, args...)
	a.Println()
	return a
}

func (o ctxOutput) Fatal(args ...interface{}) {
	log.F(o.ctx, true, "%v", fmt.Sprint(args...))
}

func (o ctxOutput) Error(args ...interface{}) {
	log.E(o.ctx, "%v", fmt.Sprint(args...))
}

func (o ctxOutput) Log(args ...interface{}) {
	log.I(o.ctx, "%v", fmt.Sprint(args...))
}

func (stdOutput) Fatal(args ...interface{}) {
	fmt.Fprintln(os.Stdout, args...)
	panic("Fatal error without test context")
}

func (stdOutput) Error(args ...interface{}) {
	fmt.Fprintln(os.Stdout, args...)
}

func (stdOutput) Log(args ...interface{}) {
	fmt.Fprintln(os.Stdout, args...)
}
