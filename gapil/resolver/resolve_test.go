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

package resolver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/semantic/printer"
)

func err(msg string) parse.ErrorList {
	return parse.ErrorList{parse.Error{Message: msg}}
}

type test struct {
	name   string
	source string
	output string
	errors parse.ErrorList
	opts   resolver.Options
}

func (t test) check(ctx context.Context) *semantic.API {
	log.I(ctx, "-------%s-------", t.name)
	m := &semantic.Mappings{}
	astAPI, errs := parser.Parse("resolve_test.api", t.source, &m.AST)
	assert.For(ctx, "parse errors").That(errs).IsNil()
	api, errs := resolver.Resolve([]*ast.API{astAPI}, m, t.opts)
	assert.For(ctx, "resolve errors").That(errs).DeepEquals(t.errors)
	if len(t.output) > 0 && len(errs) == 0 {
		output := printer.New().WriteFunction(api.Functions[0]).String()
		assert.For(ctx, "output").ThatString(output).Equals(t.output)
	}
	return api
}

func TestCyclicDependencies(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "Cyclic pseudonym declaration",
			source: `type c b
type a c
type b a`,
			errors: err("cyclic type declaration: a -> b -> c -> a"),
		}, {
			name:   "Cyclic define declaration",
			source: `define x x`,
			errors: err("cyclic define declaration: x -> x"),
		}, {
			name:   "Cyclic define declaration 2",
			source: `cmd void f() { y := x } define x x`,
			errors: err("cyclic define declaration: x -> x"),
		},
	} {
		test.check(ctx)
	}
}

func TestGlobalInitOrder(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "Global init order",
			source: `
s32 b = c - 10
s32 a = 4 + b * 5
s32 c = 1`,
		},
	} {
		test.check(ctx)
	}
}

func TestStaticArrays(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "StaticArray initializer",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3) }`,
		}, {
			name: "StaticArray initializer. Error: too many values",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3,4) }`,
			errors: err("expected 3 values, got 4"),
		}, {
			name: "StaticArray initializer. Error: too few values",
			source: `type u32[3] A
cmd void foo() { a := A(1,2) }`,
			errors: err("expected 3 values, got 2"),
		}, {
			name: "StaticArray index read",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3) i := a[1] }`,
		}, {
			name: "StaticArray index write",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3) a[2] = 2 }`,
		}, {
			name: "StaticArray index read. Error: out of bounds",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3) i := a[3] }`,
			errors: err("array index 3 is out of bounds for u32[3]"),
		}, {
			name: "StaticArray sliced. Error: cannot slice static arrays",
			source: `type u32[3] A
cmd void foo() { a := A(1,2,3) i := a[1:2] }`,
			errors: err("cannot slice static arrays"),
		},
	} {
		test.check(ctx)
	}
}

func TestMaps(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name:   "Map delete",
			source: `cmd void foo(map!(u32, string) m) { delete(m, 1) }`,
			output: `// order: Contains-fence
cmd void foo(map!(u32, string) m) {
  delete(m, 1)
  fence
}`,
		}, {
			name:   "Map delete - bad key",
			source: `cmd void foo(map!(u32, string) m) { delete(m, "blah") }`,
			errors: err("Cannot use string as key to map!(u32, string)"),
		}, {
			name:   "Map delete - bad map",
			source: `cmd void foo(int m) { delete(m, "blah") }`,
			errors: err("delete's first argument must be a map, got int"),
		},
	} {
		test.check(ctx)
	}
}

func TestMessages(t *testing.T) {
	ctx := log.Testing(t)
	boilerplate := `
		extern void foo(message bar)

		type u32   GLuint // 32 bit, unsigned binary integer.
		type u64   GLuint64 // 64 bit, unsigned binary integer.

		class ERR_FAKE {
			u32 big
			u64 bigger
		}

		class ERR_FAKE_PSEUDO {
			GLuint big
			GLuint64 bigger
		}
		`
	for _, test := range []test{
		{
			name: "ref!class -> message implicit cast",
			source: boilerplate + `
			cmd void fake() {
				foo(new!ERR_FAKE(big: 0, bigger: 1))
			}`,
		}, {
			name: "Actual -> pseudonym resolution as an argument in constructor",
			source: boilerplate + `
			cmd void fake() {
				foo(new!ERR_FAKE_PSEUDO(big: 0, bigger: 1))
			}`,
		}, {
			name: "Pseudonym -> actual resolution as an argument in constructor",
			source: boilerplate + `
			cmd void fake() {
				a := as!GLuint(-10)
				b := as!GLuint64(-1)
				foo(new!ERR_FAKE_PSEUDO(big: a, bigger: b))
			}`,
		}, {
			name: "Mixed pseudonym and actual resolution as an argument in constructor",
			source: boilerplate + `
			cmd void fake() {
				b := as!GLuint64(-1)
				foo(new!ERR_FAKE_PSEUDO(big: 10, bigger: b))
				}`,
		}, {
			name: "Pseudonym -> actual resolution as an argument in constructor - wrong pseudonym",
			source: boilerplate + `
			type s64   GLint64 // 64 bit, signed, two's complement binary integer.

			cmd void fake() {
				a := as!GLuint(-10)
				b := as!GLint64(-1)
				foo(new!ERR_FAKE_PSEUDO(big: a, bigger: b))
			}`,
			errors: err("cannot assign GLint64 to field 'bigger' of type GLuint64"),
		}, {
			name: "Accept only create statements as message - pass local variable",
			source: boilerplate + `
			cmd void fake() {
				msg := new!ERR_FAKE(big: 1, bigger: 10)
				foo(msg)
			}`,
			errors: err("Message arguments require a new class instance or forwarded message parameter, got: *semantic.Local"),
		},
	} {
		test.check(ctx)
	}
}

func TestExtractCalls(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "Call statement",
			source: `
sub void S() { }
cmd void C() { S() }`,
			output: `// order: Contains-fence
cmd void C() {
  S()
  fence
}`}, {
			name: "Call in local decl",
			source: `
sub s32 S() { return 1 }
cmd void C() { a := S() }`,
			output: `// order: Contains-fence
cmd void C() {
  a := S()
  fence
}`}, {
			name: "Call as arguments",
			source: `
sub s32 X() { return 1 }
sub s32 Y() { return 1 }
sub s32 Z() { return 1 }
sub s32 S(s32 x, s32 y, s32 z) { return x + y + z }
cmd void C() { a := S(X(), Y(), Z()) }`,
			output: `// order: Contains-fence
cmd void C() {
  _res_0 := X()
  _res_1 := Y()
  _res_2 := Z()
  a := S(_res_0, _res_1, _res_2)
  fence
}`}, {
			name: "Call in expression",
			source: `
sub u32 S() { return 1 }
cmd void C(u8* ptr, u32 count) { read(ptr[0:count * S()]) }`,
			output: `// order: Contains-fence
cmd void C(u8* ptr, u32 count) {
  _res_0 := S()
  read(ptr[as!u64(0):as!u64(count*_res_0)])
  fence
}`}, {
			name:   "Call switch expression",
			errors: err("Cannot call subroutines inside select expressions."),
			source: `
sub u32 S() { return 1 }
cmd void C(u32 a) {
  x := switch a {
    case 1: { S() }
  }
}`,
		},
	} {
		log.I(ctx, "-------%s-------", test.name)
		m := &semantic.Mappings{}
		astAPI, errs := parser.Parse("resolve_test.api", test.source, &m.AST)
		assert.For(ctx, "parse errors").That(errs).IsNil()
		api, errs := resolver.Resolve([]*ast.API{astAPI}, m, resolver.Options{
			ExtractCalls: true,
		})
		assert.For(ctx, "resolve errors").That(errs).DeepEquals(test.errors)
		if len(errs) == 0 {
			output := printer.New().WriteFunction(api.Functions[0]).String()
			assert.For(ctx, "output").ThatString(output).Equals(test.output)
		}
	}
}

func TestRemoveDeadCode(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "if false { S() }",
			source: `
sub void S() { }
cmd void C() { if false { S() } }`,
			output: `// order: Contains-fence
cmd void C() {
  fence
}`}, {
			name: "if false { S1() } else { S2() }",
			source: `
sub void S1() { }
sub void S2() { }
cmd void C() { if false { S1() } else { S2() } }`,
			output: `// order: Contains-fence
cmd void C() {
  {
    S2()
  }
  fence
}`}, {
			name: "if true { S() }",
			source: `
sub void S1() { }
sub void S2() { }
cmd void C() { if true { S1() } else { S2() } }`,
			output: `// order: Contains-fence
cmd void C() {
  {
    S1()
  }
  fence
}`}, {
			name: "kill unused locals",
			source: `
s32 X = 0
cmd void C() {
  unreferenced := 1 // dead-code stripper didn't remove reference
	used := 2
  stripped := 3
  if (false) {
    X = stripped
  }
	switch X {
	case 1:
		if (false) {
			X = used
		}
	case 2:
		foo := used
	}
}`,
			output: `// order: Contains-fence
cmd void C() {
  unreferenced := 1
  used := 2
  switch X {
    case 1: {
    }
    case 2: {
      foo := used
    }
  }
  fence
}`},
	} {
		test.opts.RemoveDeadCode = true
		test.check(ctx)
	}
}

func TestInsertFence(t *testing.T) {
	ctx := log.Testing(t)
	countFences := func(n semantic.Node) int {
		count := 0
		var f func(semantic.Node)
		f = func(n semantic.Node) {
			switch n.(type) {
			case *semantic.Fence:
				count++
			case semantic.Node:
				semantic.Visit(n, f)
			}
		}
		semantic.Visit(n, f)
		return count
	}
	for _, test := range []test{
		{
			name: "Forward declaration",
			source: `
sub void R(u8[] x) { read(x) }
sub void W(u8[] x) { write(x) }
sub void RW(u8[] x) { R(x) W(x) }
cmd void C(u8* x) { RW(x[0:1]) }`,
		}, {
			name: "Reverse declaration",
			source: `
cmd void C(u8* x) { RW(x[0:1]) }
sub void RW(u8[] x) { R(x) W(x) }
sub void W(u8[] x) { write(x) }
sub void R(u8[] x) { read(x) }`,
		}, {
			name: "Local, fence, call",
			source: `
sub void S(u8 a, u8[] x) { write(x) }
cmd void C(u8* x) {
  l := x[0]
  S(l, x[0:1])
}`,
		}, {
			name: "Impossible fence",
			source: `
sub void S(u8 a, u8[] x) { write(x) }
cmd void C(u8* x) { S(x[0], x[0:1]) }
`,
			errors: err("Only copy statements can be pre and post fence."),
		}, {
			name: "Explicit fence in subroutine",
			source: `
sub void S() { fence }
cmd void C() { S() }
`,
		}, {
			name: "Valid pre_fence annotation",
			source: `
@pre_fence sub void S() { }
cmd void C(u8* x) { S() write(x[0:1]) }
`,
		}, {
			name: "Conflicting pre_fence annotation",
			source: `
@pre_fence sub void S() { }
cmd void C(u8* x) { write(x[0:1]) S() }
`,
			errors: err("pre-statement after fence"),
		}, {
			name: "Valid post_fence annotation",
			source: `
@post_fence sub void S() { }
cmd void C(u8* x) { read(x[0:1]) S() }
`,
		}, {
			name: "Conflicting post_fence annotation",
			source: `
@post_fence sub void S() { }
cmd void C(u8* x) { S() read(x[0:1]) }
`,
			errors: err("pre-statement after fence"),
		},
	} {
		api := test.check(ctx)
		if len(test.errors) == 0 {
			expectedFences := 1
			gotFences := countFences(api.Functions[0])
			if !assert.For(ctx, "number of fences").ThatInteger(gotFences).Equals(expectedFences) {
				fmt.Println(printer.New().WriteFunction(api.Functions[0]).String())
				for _, s := range api.Subroutines {
					fmt.Println(printer.New().WriteFunction(s).String())
				}
			}
		}
	}
}

func TestInternal(t *testing.T) {
	ctx := log.Testing(t)
	common := `
u8[] slice
u8* pointer
@internal u8[] internalslice
@internal u8* internalpointer

sub void subPtr(u8* p) {}
sub void subSlice(u8[] s) {}

sub void subInternalPtr(@internal u8* p) {}
sub void subInternalSlice(@internal u8[] s) {}
`
	for _, test := range []test{
		{
			name: "Check valid usage of @internal annotation on slices and pointers",
			source: common + `
cmd void foo(u8* p) {
    slice = p[0:10]
    internalslice = internalslice
    internalpointer = as!u8*(internalslice)
    internalslice = internalslice[1:2]
    internalpointer = as!u8*(internalslice[1:2])
    copy(internalslice, slice)
    copy(slice, internalslice)
    n := make!u8(5)
    internalslice = n
    subPtr(p)
    subSlice(p[0:5])
}`,
		}, {
			name:   "non @internal slice assigned to @internal slice",
			source: common + `cmd void foo(u8* p) { internalslice = p[0:10] }`,
			errors: err("Assigning from a non-internal to an internal"),
		}, {
			name:   "non @internal pointer assigned to @internal pointer",
			source: common + `cmd void foo(u8* p) { internalpointer = p }`,
			errors: err("Assigning from a non-internal to an internal"),
		}, {
			name:   "non @internal pointer assigned to @internal pointer via slice",
			source: common + `cmd void foo(u8* p) { internalpointer = as!u8*(p[0:10]) }`,
			errors: err("Assigning from a non-internal to an internal"),
		}, {
			name:   "non @internal pointer passed to @internal parameter",
			source: common + `cmd void foo(u8* p) { subInternalPtr(p) }`,
			errors: err("Passing a non-internal argument to an internal parameter"),
		}, {
			name:   "non @internal slice passed to @internal slice",
			source: common + `cmd void foo(u8* p) { subInternalSlice(p[0:5]) }`,
			errors: err("Passing a non-internal argument to an internal parameter"),
		},
	} {
		test.check(ctx)
	}
}

func TestApiIndex(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name:   "Negative api index",
			source: `api_index -1`,
			errors: err("cannot convert API index \"-1\" into 4-bit unsigned integer"),
		},
		{
			name:   "Api index exceeding 4 bits",
			source: `api_index 16`,
			errors: err("cannot convert API index \"16\" into 4-bit unsigned integer"),
		},
	} {
		test.check(ctx)
	}
}

func TestRecursive(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []test{
		{
			name: "Simple Recursive Function",
			source: `
sub void recursive(bool b) {
	if (b) {
		recursive(!b)
	}
}
cmd void test() {
	recursive(true)
}
`,
		},
		{
			name: "Invalid Recursive Function",
			source: `
sub void recursive(bool b) {
	if (b) {
		fence
		recursive(!b)
	}
 }
 cmd void test() {
	recursive(true)
}
`,
			errors: err("Fence in recursive function"),
		},
		{
			name: "Implicit Recursive Function PostFence",
			source: `
sub void recursive(bool b) {
	if (b) {
		recursive(!b)
	}
	x := ?
 }
 cmd void test() {
	recursive(true)
}
`,
		},
		{
			name: "Implicit Recursive Function PostFence",
			source: `
sub void recursive(bool b) {
	x := ?
	if (b) {
		recursive(!b)
	}
 }
 cmd void test() {
	recursive(true)
}
`,
		},
		{
			name: "Allowed outer Fence Recursive Function",
			source: `
sub void recursive(bool b) {
	if (b) {
		recursive(!b)
	}
 }
 cmd void test() {
	fence
	recursive(true)
}
`,
		},
		{
			name: "Allowed Outer Pre-Fence Recursive Function",
			source: `
sub void recursive(bool b) {
	if (b) {
		recursive(!b)
	}
 }
 cmd void test() {
	fence
	recursive(true)
}
`,
		},
		{
			name: "Doubly Recursive Function",
			source: `
sub void recursive2(bool b) {
	if (b) {
		recursive(!b)
	}
}
sub void recursive(bool b) {
	if (b) {
		recursive2(!b)
	}
 }
 cmd void test() {
	recursive(true)
}
`,
		},
		{
			name: "Invalid Doubly Recursive Function",
			source: `
sub void recursive2(bool b) {
	fence
	if (b) {
		recursive(!b)
	}
}
sub void recursive(bool b) {
	recursive2(!b)
}

cmd void test() {
	recursive(true)
}
`,
			errors: err("Fence in recursive function"),
		},
		{
			name: "Implicit PreFence Recursive Function",
			source: `

sub void recursive2(u8* dat) {
	read(dat[0:1])
	recursive(dat)
}
sub void recursive(u8* dat) {
	recursive2(dat)
 }
 cmd void test(u8* pData) {
	recursive(pData)
}
`,
		},
		{
			name: "Invalid Implicit Fence Recursive Function",
			source: `

sub void recursive2(u8* dat) {
	read(dat[0:1])
	recursive(dat)
	write(dat[0:1])
}
sub void recursive(u8* dat) {
	recursive2(dat)
 }
 cmd void test(u8* pData) {
	recursive(pData)
}
`,
			errors: err("Fence in recursive function"),
		},
	} {
		test.check(ctx)
	}
}
