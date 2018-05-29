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

package flags_test

import (
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/assert"
)

const (
	Pagefaultman Hero = iota
	Supergirl
	Batman
	Aquaman
)
const (
	PoisonIvy Villain = iota
	Joker
	HarleyQuinn
	Sinestro
)

type (
	Hero    uint8
	Villain int32

	MyVerbFlags struct {
		Str     string
		Hero    Hero
		Villain Villain
	}
)

var heroNames = map[Hero]string{
	Pagefaultman: "pagefaultman",
	Supergirl:    "supergirl",
	Batman:       "batman",
	Aquaman:      "aquaman",
}

func (h Hero) String() string        { return heroNames[h] }
func (h *Hero) Choose(c interface{}) { *h = c.(Hero) }
func (h *Hero) Chooser() flags.Chooser {
	c := flags.Chooser{Value: h}
	for h := range heroNames {
		c.Choices = append(c.Choices, h)
	}
	return c
}

func (v *Villain) Choose(c interface{}) { *v = c.(Villain) }
func (v Villain) String() string {
	switch v {
	case PoisonIvy:
		return "Poison Ivy"
	case Joker:
		return "Joker"
	case HarleyQuinn:
		return "Harley Quinn"
	case Sinestro:
		return "Sinestro"
	default:
		return strconv.Itoa(int(v))
	}
}

func TestChoiceParsing(t *testing.T) {
	assert := assert.To(t)

	for _, cs := range []struct {
		args []string
		ok   bool
		h    Hero
		v    Villain
		s    string
	}{
		{[]string{"-hero", "Joker"}, false, 0, 0, "-"},
		{[]string{"-hero", "AQUAMAN"}, true, Aquaman, 0, ""},
		{[]string{"-str", "foo", "-hero", "supergirl"}, true, Supergirl, 0, "foo"},
		{[]string{"-str", "bar", "-hero", "Batman"}, true, Batman, 0, "bar"},
		{[]string{}, true, Pagefaultman, 0, ""},
		{[]string{"-villain", "Joker"}, true, 0, Joker, ""},
		{[]string{"-villain", "Harley Quinn"}, true, 0, HarleyQuinn, ""},
	} {
		verb := &struct{ MyVerbFlags }{}
		flags := flags.Set{}
		flags.Raw.Usage = func() {}
		flags.Raw.SetOutput(ioutil.Discard)
		flags.Bind("", verb, "")
		err := flags.Raw.Parse(cs.args)
		if cs.ok {
			assert.For("err").ThatError(err).Succeeded()
			assert.For("str").ThatString(verb.Str).Equals(cs.s)
			assert.For("hero").That(verb.Hero).Equals(cs.h)
		} else {
			assert.For("err").ThatError(err).Failed()
		}
	}
}
