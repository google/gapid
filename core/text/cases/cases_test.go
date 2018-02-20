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

package cases

import (
	"fmt"
	"reflect"
	"testing"
)

func ExampleSnake() {
	fmt.Println(Snake("cat_says_meow"), Words{"cat", "says", "meow"}.ToSnake())
	// Output:
	// [cat says meow] cat_says_meow
}

func ExamplePascal() {
	fmt.Println(Pascal("CatSaysMeow"), Words{"cat", "says", "meow"}.ToPascal())
	// Output:
	// [Cat Says Meow] CatSaysMeow
}

func ExampleCamel() {
	fmt.Println(Camel("catSaysMeow"), Words{"cat", "says", "meow"}.ToCamel())
	// Output:
	// [cat Says Meow] catSaysMeow
}

func TestSnake(t *testing.T) {
	for str, want := range map[string]Words{
		"a_simple_case":     {"a", "simple", "case"},
		"case_Is_PRESERVED": {"case", "Is", "PRESERVED"},
		"no-underscores":    {"no-underscores"},
		"":                  {},
	} {
		if got := Snake(str); !reflect.DeepEqual(got, want) {
			t.Errorf("Snake(%v) - Got: %v, Want: %v", str, got, want)
		}
		if got := want.ToSnake(); got != str {
			t.Errorf("%v.ToSnake() - Got: %v, Want: %v", want, got, str)
		}
	}
}

func TestPascal(t *testing.T) {
	for _, test := range []struct {
		str       string // input string
		wantWords Words  // expected Pascal(pre)
		wantStr   string // expected w.ToPascal()
	}{
		{"ASimpleCase", Words{"A", "Simple", "Case"}, "ASimpleCase"},
		{"startsWithLower", Words{"starts", "With", "Lower"}, "StartsWithLower"},
		{"ABC_DEF", Words{"A", "B", "C_D", "E", "F"}, "ABC_DEF"},
		{"", Words{}, ""},
	} {
		if got := Pascal(test.str); !reflect.DeepEqual(got, test.wantWords) {
			t.Errorf("Pascal(%v) - Got: %v, Want: %v", test.str, got, test.wantWords)
		}
		if got := test.wantWords.ToPascal(); got != test.wantStr {
			t.Errorf("%v.ToPascal() - Got: %v, Want: %v", test.wantWords, got, test.wantStr)
		}
	}
}

func TestCamel(t *testing.T) {
	for _, test := range []struct {
		str       string // input string
		wantWords Words  // expected Camel(pre)
		wantStr   string // expected w.ToCamel()
	}{
		{"aSimpleCase", Words{"a", "Simple", "Case"}, "aSimpleCase"},
		{"StartsWithUpper", Words{"Starts", "With", "Upper"}, "StartsWithUpper"},
		{"ABC_DEF", Words{"A", "B", "C_D", "E", "F"}, "ABC_DEF"},
		{"", Words{}, ""},
	} {
		if got := Camel(test.str); !reflect.DeepEqual(got, test.wantWords) {
			t.Errorf("Camel(%v) - Got: %v, Want: %v", test.str, got, test.wantWords)
		}
		if got := test.wantWords.ToCamel(); got != test.wantStr {
			t.Errorf("%v.ToCamel() - Got: %v, Want: %v", test.wantWords, got, test.wantStr)
		}
	}
}

func TestToUpperToLower(t *testing.T) {
	for _, test := range []struct {
		words Words // input words
		upper Words // expected
		lower Words // expected
	}{
		{Words{"Hello", "There", "How", "Are", "YOU?"},
			Words{"HELLO", "THERE", "HOW", "ARE", "YOU?"},
			Words{"hello", "there", "how", "are", "you?"}},
		{Words{"Numbers", "are", "not", "affected", "12345"},
			Words{"NUMBERS", "ARE", "NOT", "AFFECTED", "12345"},
			Words{"numbers", "are", "not", "affected", "12345"}},
	} {
		if got := test.words.ToUpper(); !reflect.DeepEqual(got, test.upper) {
			t.Errorf("ToUpper(%v) - Got: %v, Want: %v", test.words, got, test.upper)
		}
		if got := test.words.ToLower(); !reflect.DeepEqual(got, test.lower) {
			t.Errorf("ToLower(%v) - Got: %v, Want: %v", test.words, got, test.lower)
		}
	}
}

func TestTitleUntitle(t *testing.T) {
	for _, test := range []struct {
		words   Words // input words
		title   Words // expected
		untitle Words // expected
	}{
		{Words{"Hello", "there", "HOW", "are", "YOU?"},
			Words{"Hello", "There", "HOW", "Are", "YOU?"},
			Words{"hello", "there", "hOW", "are", "yOU?"}},
		{Words{"Numbers", "are", "not", "affected", "12345"},
			Words{"Numbers", "Are", "Not", "Affected", "12345"},
			Words{"numbers", "are", "not", "affected", "12345"}},
	} {
		if got := test.words.Title(); !reflect.DeepEqual(got, test.title) {
			t.Errorf("Title(%v) - Got: %v, Want: %v", test.words, got, test.title)
		}
		if got := test.words.Untitle(); !reflect.DeepEqual(got, test.untitle) {
			t.Errorf("Untitle(%v) - Got: %v, Want: %v", test.words, got, test.untitle)
		}
	}
}
