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

package ia64_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/compiler/mangling"
	"github.com/google/gapid/gapil/compiler/mangling/ia64"
)

func TestIA64(t *testing.T) {
	ctx := log.Testing(t)

	/*
		namespace food {
		namespace fruit {

		class Apple {
		public:
			int yummy(int, char*);
			bool eat(void* person);
			int calories();
			bool looks_like(Apple*);
			static bool healthy();
			static int compare(Apple* a, Apple* b);
			template <typename T> bool same_as(T other); (T: int)
			template <typename X, typename Y> static void juice(); (X: int, Y: bool)
		};

		template <typename F>
		class Smoothie {
		public:
			void slurp(F);  (F: fruit)
		};

		} // namespace fruit
		} // namespace food
	*/
	food := &mangling.Namespace{Name: "food"}
	fruit := &mangling.Namespace{Name: "fruit", Parent: food}
	apple := &mangling.Class{Name: "Apple", Parent: fruit}
	smoothie := &mangling.Class{Name: "Smoothie", Parent: fruit, TemplateArgs: []mangling.Type{apple}}

	yummy := &mangling.Function{
		Name:       "yummy",
		Return:     mangling.Int,
		Parameters: []mangling.Type{mangling.Int, mangling.Pointer{To: mangling.Char}},
		Parent:     apple,
	}

	eat := &mangling.Function{
		Name:       "eat",
		Return:     mangling.Bool,
		Parameters: []mangling.Type{mangling.Pointer{To: mangling.Void}},
		Parent:     apple,
	}

	calories := &mangling.Function{
		Name:   "calories",
		Return: mangling.Int,
		Parent: apple,
		Const:  true,
	}

	looksLike := &mangling.Function{
		Name:       "looks_like",
		Return:     mangling.Bool,
		Parameters: []mangling.Type{mangling.Pointer{To: apple}},
		Parent:     apple,
	}

	healthy := &mangling.Function{
		Name:   "healthy",
		Return: mangling.Bool,
		Static: true,
		Parent: apple,
	}

	compare := &mangling.Function{
		Name:       "compare",
		Return:     mangling.Int,
		Parameters: []mangling.Type{mangling.Pointer{To: apple}, mangling.Pointer{To: apple}},
		Parent:     apple,
	}

	sameAs := &mangling.Function{
		Name:         "same_as",
		Return:       mangling.Bool,
		Parameters:   []mangling.Type{mangling.TemplateParameter(0)},
		TemplateArgs: []mangling.Type{mangling.Int},
		Parent:       apple,
	}

	juice := &mangling.Function{
		Name:         "juice",
		Return:       mangling.Void,
		TemplateArgs: []mangling.Type{mangling.Int, mangling.Bool},
		Parent:       apple,
	}

	slurp := &mangling.Function{
		Name:       "slurp",
		Return:     mangling.Void,
		Parameters: []mangling.Type{apple},
		Parent:     smoothie,
	}

	for _, t := range []struct {
		name     string
		sym      mangling.Entity
		expected string
	}{
		{"food::fruit::Apple", apple, "_ZN4food5fruit5AppleE"},
		{"food::fruit::Apple::yummy", yummy, "_ZN4food5fruit5Apple5yummyEiPc"},
		{"food::fruit::Apple::eat", eat, "_ZN4food5fruit5Apple3eatEPv"},
		{"food::fruit::Apple::calories", calories, "_ZNK4food5fruit5Apple8caloriesEv"},
		{"food::fruit::Apple::looks_like", looksLike, "_ZN4food5fruit5Apple10looks_likeEPS1_"},
		{"food::fruit::Apple::healthy", healthy, "_ZN4food5fruit5Apple7healthyEv"},
		{"food::fruit::Apple::compare", compare, "_ZN4food5fruit5Apple7compareEPS1_S2_"},
		{"food::fruit::Apple::same_as", sameAs, "_ZN4food5fruit5Apple7same_asIiEEbT_"},
		{"food::fruit::Apple::juice", juice, "_ZN4food5fruit5Apple5juiceIibEEvv"},
		{"food::fruit::Smoothie::slurp", slurp, "_ZN4food5fruit8SmoothieINS0_5AppleEE5slurpES2_"},
	} {
		assert.For(ctx, t.name).ThatString(ia64.Mangle(t.sym)).Equals(t.expected)
	}
}
