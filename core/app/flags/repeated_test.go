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

package flags_test

import (
	"io/ioutil"
	"testing"

	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/assert"
)

type MyFlag string

func (f *MyFlag) String() string     { return string(*f) }
func (f *MyFlag) Set(v string) error { *f = MyFlag(v); return nil }

type MyFlags struct {
	Str    string
	Bools  []bool
	Strs   []string
	Ints   []int
	Uints  []uint
	Floats []float64
	Mine   []MyFlag
}

var (
	args1 = []string{
		"-str", "foo",
		"-bools", "true",
		"-strs", "one",
		"-ints", "1",
		"-uints", "2",
		"-floats", "3.5",
		"-mine", "yours",
	}
	args2 = []string{
		"-str", "bar",
		"-bools", "true", "-bools", "false",
		"-strs", "one", "-strs", "two",
		"-ints", "1", "-ints", "4",
		"-uints", "2", "-uints", "5",
		"-floats", "3.5", "-floats", "6.7",
		"-mine", "yours", "-mine", "ours",
	}
	args3 = []string{
		"-str", "baz",
		"-bools", "true", "-bools", "false", "-bools", "false",
		"-strs", "one", "-strs", "two", "-strs", "three",
		"-ints", "1", "-ints", "4", "-ints", "7",
		"-uints", "2", "-uints", "5", "-uints", "8",
		"-floats", "3.5", "-floats", "6.7", "-floats", "9.2",
		"-mine", "yours", "-mine", "ours", "-mine", "theirs",
	}
)

func b(v ...bool) []bool       { return v }
func s(v ...string) []string   { return v }
func i(v ...int) []int         { return v }
func u(v ...uint) []uint       { return v }
func f(v ...float64) []float64 { return v }
func m(v ...string) (r []MyFlag) {
	for _, s := range v {
		r = append(r, MyFlag(s))
	}
	return
}

func TestRepeatedParsing(t *testing.T) {
	assert := assert.To(t)

	for _, cs := range []struct {
		args []string
		exp  MyFlags
	}{
		{args1, MyFlags{"foo", b(true), s("one"), i(1), u(2), f(3.5), m("yours")}},
		{args2, MyFlags{"bar", b(true, false), s("one", "two"), i(1, 4), u(2, 5), f(3.5, 6.7), m("yours", "ours")}},
		{args3, MyFlags{"baz", b(true, false, false), s("one", "two", "three"), i(1, 4, 7), u(2, 5, 8), f(3.5, 6.7, 9.2), m("yours", "ours", "theirs")}},
	} {
		verb := &struct{ MyFlags }{}
		flags := flags.Set{}
		flags.Raw.Usage = func() {}
		flags.Raw.SetOutput(ioutil.Discard)
		flags.Bind("", verb, "")
		err := flags.Raw.Parse(cs.args)
		assert.For("err").ThatError(err).Succeeded()
		assert.For("str").ThatString(verb.Str).Equals(cs.exp.Str)
		assert.For("bools").ThatSlice(verb.Bools).Equals(cs.exp.Bools)
		assert.For("strs").ThatSlice(verb.Strs).Equals(cs.exp.Strs)
		assert.For("ints").ThatSlice(verb.Ints).Equals(cs.exp.Ints)
		assert.For("uints").ThatSlice(verb.Uints).Equals(cs.exp.Uints)
		assert.For("floats").ThatSlice(verb.Floats).Equals(cs.exp.Floats)
		assert.For("mine").ThatSlice(verb.Mine).Equals(cs.exp.Mine)
	}
}
