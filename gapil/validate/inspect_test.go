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

package validate_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/validate"
)

const maxErrors = 10

func compile(ctx context.Context, source string) (*semantic.API, *semantic.Mappings, error) {
	m := &semantic.Mappings{}
	parsed, errs := parser.Parse("no_unreachables_test.api", source, &m.AST)
	if err := gapil.CheckErrors(source, errs, maxErrors); err != nil {
		return nil, nil, err
	}
	compiled, errs := resolver.Resolve([]*ast.API{parsed}, m, resolver.Options{
		RemoveDeadCode: true,
	})
	if err := gapil.CheckErrors(source, errs, maxErrors); err != nil {
		return nil, nil, err
	}
	return compiled, m, nil
}

type test struct {
	source   string
	expected string
}

func TestSimpleUnreachable(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []test{
		///////////////////////////////////////////////////////
		// Bool tests
		///////////////////////////////////////////////////////
		{`s32 x = 0
      cmd void f() {
        if (true) { x = 1 }
      }`, ""}, // Handled by resolver.removeDeadCode()

		{`s32 x = 0
      cmd void f() {
        if (true) { x = 1 } else {}
      }`, ""}, // Handled by resolver.removeDeadCode()

		{`s32 x = 0
      cmd void f() {
        if (false) {} else { x = 2 }
      }`, ""}, // Handled by resolver.removeDeadCode()

		{`s32 x = 0
      cmd void f() {
        a := false
        if (a) { x = 1 } else { x = 2 }
      }`, ""}, // Handled by resolver.removeDeadCode()

		{`s32 x = 0
      cmd void f() {
        a := true
        if (a) { x = 1 } else { x = 2 }
      }`, ""}, // Handled by resolver.removeDeadCode()

		{`s32 x = 0
      cmd void f() {
        a := true == false
        if (a) { x = 1 } else { x = 2 }
      }`, `no_unreachables_test.api:4:16 Unreachable block`},

		{`s32 x = 0
      cmd void f() {
        a := true && false
        if (a) { x = 1 } else { x = 2 }
      }`, `no_unreachables_test.api:4:16 Unreachable block`},

		{`s32 x = 0
      cmd void f() {
        a := true || false
        if (a) { x = 1 } else { x = 2 }
      }`, `no_unreachables_test.api:4:31 Unreachable block`},

		{`s32 x = 0
      cmd void f(bool a) {
        if (a) { if (!a) { x = 1 } }
        if (!a) { if (a) { x = 1 } }
        if (a) {} else if (a) { x = 1 }
        if (!a) {} else if (!a) { x = 1 }
      }`, `
no_unreachables_test.api:3:26 Unreachable block
no_unreachables_test.api:4:26 Unreachable block
no_unreachables_test.api:5:31 Unreachable block
no_unreachables_test.api:6:33 Unreachable block
`},

		{`s32 x = 0
      cmd void f(bool v) {
        assert(v)
        if (v) { x = 1 } else { x = 2 }
      }`, `no_unreachables_test.api:4:31 Unreachable block`},

		///////////////////////////////////////////////////////
		// Uint tests
		///////////////////////////////////////////////////////
		{`s32 x = 0
      cmd void f(u32 a) {
        if (a > 5) { x = 1 } else { x = 2 }
      }`, ""},

		{`s32 x = 0
      cmd void f() {
        a := as!u32(5)
        if (a > 4) { x = 1 } else { x = 2 }
        if (a > 5) { x = 1 } else { x = 2 }
        if (a > 6) { x = 1 } else { x = 2 }
        if (4 > a) { x = 1 } else { x = 2 }
        if (5 > a) { x = 1 } else { x = 2 }
        if (6 > a) { x = 1 } else { x = 2 }
      }`, `
no_unreachables_test.api:4:35 Unreachable block
no_unreachables_test.api:5:20 Unreachable block
no_unreachables_test.api:6:20 Unreachable block
no_unreachables_test.api:7:20 Unreachable block
no_unreachables_test.api:8:20 Unreachable block
no_unreachables_test.api:9:35 Unreachable block
`},

		{`s32 x = 0
      cmd void f() {
        a := as!u32(5)
        if (a >= 4) { x = 1 } else { x = 2 }
        if (a >= 5) { x = 1 } else { x = 2 }
        if (a >= 6) { x = 1 } else { x = 2 }
        if (4 >= a) { x = 1 } else { x = 2 }
        if (5 >= a) { x = 1 } else { x = 2 }
        if (6 >= a) { x = 1 } else { x = 2 }
      }`, `
no_unreachables_test.api:4:36 Unreachable block
no_unreachables_test.api:5:36 Unreachable block
no_unreachables_test.api:6:21 Unreachable block
no_unreachables_test.api:7:21 Unreachable block
no_unreachables_test.api:8:36 Unreachable block
no_unreachables_test.api:9:36 Unreachable block
`},

		{`s32 x = 0
      cmd void f() {
        a := as!u32(5)
        if (a < 4) { x = 1 } else { x = 2 }
        if (a < 5) { x = 1 } else { x = 2 }
        if (a < 6) { x = 1 } else { x = 2 }
        if (4 < a) { x = 1 } else { x = 2 }
        if (5 < a) { x = 1 } else { x = 2 }
        if (6 < a) { x = 1 } else { x = 2 }
      }`, `
no_unreachables_test.api:4:20 Unreachable block
no_unreachables_test.api:5:20 Unreachable block
no_unreachables_test.api:6:35 Unreachable block
no_unreachables_test.api:7:35 Unreachable block
no_unreachables_test.api:8:20 Unreachable block
no_unreachables_test.api:9:20 Unreachable block
`},

		{`s32 x = 0
      cmd void f() {
        a := as!u32(5)
        if (a <= 4) { x = 1 } else { x = 2 }
        if (a <= 5) { x = 1 } else { x = 2 }
        if (a <= 6) { x = 1 } else { x = 2 }
        if (4 <= a) { x = 1 } else { x = 2 }
        if (5 <= a) { x = 1 } else { x = 2 }
        if (6 <= a) { x = 1 } else { x = 2 }
      }`, `
no_unreachables_test.api:4:21 Unreachable block
no_unreachables_test.api:5:36 Unreachable block
no_unreachables_test.api:6:36 Unreachable block
no_unreachables_test.api:7:36 Unreachable block
no_unreachables_test.api:8:36 Unreachable block
no_unreachables_test.api:9:21 Unreachable block
`},

		{`s32 x = 0
      cmd void f() {
        a := as!u32(5)
        if (a == 4) { x = 1 } else { x = 2 }
        if (a == 5) { x = 1 } else { x = 2 }
        if (a == 6) { x = 1 } else { x = 2 }
      }`, `
no_unreachables_test.api:4:21 Unreachable block
no_unreachables_test.api:5:36 Unreachable block
no_unreachables_test.api:6:21 Unreachable block
`},

		///////////////////////////////////////////////////////
		// Abort tests
		///////////////////////////////////////////////////////
		{`cmd void f() {
        a := 1
        abort
        b := 2
        c := 3
      }`, `no_unreachables_test.api:4:9 Unreachable statement`},

		{`s32 x = 0
      cmd void f(bool v) {
        if v {
          abort
        }
        if v {
          x = 1
        }
      }`, "no_unreachables_test.api:6:14 Unreachable block"},

		///////////////////////////////////////////////////////
		// Subroutine tests
		///////////////////////////////////////////////////////
		{`s32 x = 0
      sub void S(u32 i) {
        if i < 3 {
          x = 1
        }
      }
      cmd void f() {
        S(4)
      }
      `, `no_unreachables_test.api:3:18 Unreachable block`},

		///////////////////////////////////////////////////////
		// Map indexing tests
		///////////////////////////////////////////////////////
		/* TODO: Map tests.
				{
					`map!(u32, f32) M
		       f32 V
		       cmd void foo() {
		         if (5 in M) && true {
		           V = M[5]
		         }
		       }`, ``,
				}, {
					`map!(u32, f32) M
		       f32 V
		       cmd void foo() {
		         if !(5 in M) {
		           abort
		         }
		         V = M[5]
		       }`, ``,
				}, {
					`map!(u32, f32) M
		       f32 V
		       cmd void foo() {
		         M[5] = 1
		         V = M[5]
		       }`, ``,
				}, {
					`map!(u32, f32) M
		       f32 V
		       cmd void foo() {
		         V = M[5]
		       }`, ``,
				}, {
					`map!(u32, f32) M
		       f32 V
		       cmd void foo() {
		         M[5] = 1
		         delete(M, 5)
		         V = M[5]
		       }`, ``,
				},
		*/
	} {
		api, mappings, err := compile(ctx, test.source)
		ok := true
		ok = assert.For(ctx, "err").ThatError(err).Succeeded() && ok
		ok = assert.For(ctx, "api").Critical().That(api).IsNotNil() && ok
		got := fmt.Sprint(validate.Inspect(api, mappings))
		expected := strings.TrimSpace(test.expected)
		ok = assert.For(ctx, "got").ThatString(got).Equals(expected) && ok
		if !ok {
			log.E(ctx, "test failed.\n  source: %v\n  got:  %v", test.source, got)
		}
	}
}
