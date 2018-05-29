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

package shell_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/shell"
)

func TestEmptyEnv(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	assert.For("Vars").ThatSlice(env.Vars()).Equals([]string{})
}

func TestEnvSet(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	env.Set("cat", "meow").
		Set("dog", "woof").
		Set("fox", "").
		Set("bird", "tweet")
	assert.For("Vars").ThatSlice(env.Vars()).Equals([]string{
		"cat=meow",
		"dog=woof",
		"fox",
		"bird=tweet",
	})
}

func TestEnvGet(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	env.Set("cat", "meow").
		Set("dog", "woof").
		Set("fox", "").
		Set("bird", "tweet")
	assert.For("Vars").ThatString(env.Get("cat")).Equals("meow")
	assert.For("Vars").ThatString(env.Get("dog")).Equals("woof")
	assert.For("Vars").ThatString(env.Get("bird")).Equals("tweet")
	assert.For("Vars").ThatString(env.Get("fox")).Equals("")
}

func TestEnvExists(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	env.Set("cat", "meow").
		Set("dog", "woof").
		Set("fox", "").
		Set("bird", "tweet")
	assert.For("Vars").That(env.Exists("cat")).Equals(true)
	assert.For("Vars").That(env.Exists("dog")).Equals(true)
	assert.For("Vars").That(env.Exists("fox")).Equals(true)
	assert.For("Vars").That(env.Exists("bird")).Equals(true)
	assert.For("Vars").That(env.Exists("fish")).Equals(false)
}

func TestEnvAddPathStart(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	env.PathListSeparator = ':'
	env.Set("aaa", "xxx").
		Set("bbb", "").
		Set("ccc", "yyy")

	env.AddPathStart("aaa", "/blah", "/halb").
		AddPathStart("bbb", "/blah", "/halb").
		AddPathStart("ccc", "/blah", "/halb").
		AddPathStart("ddd", "/blah", "/halb")

	assert.For("Vars").ThatSlice(env.Vars()).Equals([]string{
		"aaa=/blah:/halb:xxx",
		"bbb=/blah:/halb",
		"ccc=/blah:/halb:yyy",
		"ddd=/blah:/halb",
	})
}

func TestEnvAddPathEnd(t *testing.T) {
	assert, env := assert.To(t), shell.NewEnv()
	env.PathListSeparator = ':'
	env.Set("aaa", "xxx").
		Set("bbb", "").
		Set("ccc", "yyy")

	env.AddPathEnd("aaa", "/blah", "/halb").
		AddPathEnd("bbb", "/blah", "/halb").
		AddPathEnd("ccc", "/blah", "/halb").
		AddPathEnd("ddd", "/blah", "/halb")

	assert.For("Vars").ThatSlice(env.Vars()).Equals([]string{
		"aaa=xxx:/blah:/halb",
		"bbb=/blah:/halb",
		"ccc=yyy:/blah:/halb",
		"ddd=/blah:/halb",
	})
}
