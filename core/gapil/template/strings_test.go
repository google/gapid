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

package template

import (
	"reflect"
	"testing"
)

func TestJoinWith(t *testing.T) {
	in := stringify("a", "b", "cd", "ef", "g")
	expected := "a•b•cd•ef•g"
	got := Functions{}.JoinWith("•", in)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("JoinWith(\"•\", %#v) returned unexpected value. Expected: %#v, got: %#v",
			in, expected, got)
	}
}

func TestSplitOn(t *testing.T) {
	for _, v := range []struct {
		in       string
		expected stringList
	}{
		{
			"a•single•bullet•delimited•string",
			stringList{"a", "single", "bullet", "delimited", "string"},
		}, {
			"•a•few•bullet•••delimited••strings••",
			stringList{"a", "few", "bullet", "delimited", "strings"},
		},
	} {
		got := Functions{}.SplitOn("•", v.in)
		if !reflect.DeepEqual(v.expected, got) {
			t.Errorf("Split(\"•\", %#v) returned unexpected value. Expected: %#v, got: %#v",
				v.in, v.expected, got)
		}
	}
}

func TestSplitUpperCase(t *testing.T) {
	for _, v := range []struct {
		in       string
		expected stringList
	}{
		{
			"abcDEFgh",
			stringList{"abc", "D", "E", "F", "gh"},
		}, {
			"ABC_DEF",
			stringList{"A", "B", "C", "_", "D", "E", "F"},
		}, {
			"xxx123ABC456DEf789ghi",
			stringList{"xxx123", "A", "B", "C", "456", "D", "E", "f789ghi"},
		},
	} {
		got := Functions{}.SplitUpperCase(v.in)
		if !reflect.DeepEqual(v.expected, got) {
			t.Errorf("SplitUpperCase(%#v) returned unexpected value. Expected: %#v, got: %#v",
				v.in, v.expected, got)
		}
	}
}

func TestSplitPascalCase(t *testing.T) {
	for _, v := range []struct {
		in       string
		expected stringList
	}{
		{
			"abcDEFgh",
			stringList{"abc", "D", "E", "Fgh"},
		}, {
			"ABC_DEF",
			stringList{"A", "B", "C_D", "E", "F"},
		}, {
			"ATitledString",
			stringList{"A", "Titled", "String"},
		}, {
			"anUntitledString",
			stringList{"an", "Untitled", "String"},
		}, {
			"xxx123ABC456DEf789ghi",
			stringList{"xxx123A", "B", "C456D", "Ef789ghi"},
		},
	} {
		got := Functions{}.SplitPascalCase(v.in)
		if !reflect.DeepEqual(v.expected, got) {
			t.Errorf("SplitPascalCase(%#v) returned unexpected value. Expected: %#v, got: %#v",
				v.in, v.expected, got)
		}
	}
}

func TestTitle(t *testing.T) {
	in := stringList{"abc", "dE f", "G hi", "JKL"}
	expected := stringList{"Abc", "DE f", "G hi", "JKL"}
	got := Functions{}.Title(in)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Title(%#v) returned unexpected value. Expected: %#v, got: %#v",
			in, expected, got)
	}
}

func TestUntitle(t *testing.T) {
	in := stringList{"abc", "dE F", "G hi", "JKL"}
	expected := stringList{"abc", "dE F", "g hi", "jKL"}
	got := Functions{}.Untitle(in)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Untitle(%#v) returned unexpected value. Expected: %#v, got: %#v",
			in, expected, got)
	}
}

func TestLower(t *testing.T) {
	in := stringList{"abc", "dE F", "G hi", "JKL"}
	expected := stringList{"abc", "de f", "g hi", "jkl"}
	got := Functions{}.Lower(in)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Lower(%#v) returned unexpected value. Expected: %#v, got: %#v",
			in, expected, got)
	}
}

func TestUpper(t *testing.T) {
	in := stringList{"abc", "dE F", "G hi", "JKL"}
	expected := stringList{"ABC", "DE F", "G HI", "JKL"}
	got := Functions{}.Upper(in)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Upper(%#v) returned unexpected value. Expected: %#v, got: %#v",
			in, expected, got)
	}
}

func TestContains(t *testing.T) {
	for _, v := range []struct {
		in       stringList
		substr   string
		expected bool
	}{
		{stringList{"abc", "def", "ghi", "jkl", "mno"}, "ghi", true},
		{stringList{"abc", "def", "ghi", "jkl", "mno"}, "efg", false},
	} {
		got := Functions{}.Contains(v.substr, v.in)
		if !reflect.DeepEqual(v.expected, got) {
			t.Errorf("Contains(%#v) returned unexpected value. Expected: %#v, got: %#v",
				v.in, v.expected, got)
		}
	}
}

func TestReplace(t *testing.T) {
	for _, v := range []struct {
		in       stringList
		old      string
		new      string
		expected stringList
	}{
		{
			stringList{"abc", "def", "ghi", "jkl", "mno"},
			"ghi", "123",
			stringList{"abc", "def", "123", "jkl", "mno"},
		}, {
			stringList{"abc", "def", "ghi", "jkl", "mno"},
			"jkl", "456",
			stringList{"abc", "def", "ghi", "456", "mno"},
		}, {
			stringList{"aba", "bab", "aaa", "bbb", "abb"},
			"a", "u",
			stringList{"ubu", "bub", "uuu", "bbb", "ubb"},
		},
	} {
		got := Functions{}.Replace(v.old, v.new, v.in)
		if !reflect.DeepEqual(v.expected, got) {
			t.Errorf("Replace(%#v, %#v, %#v) returned unexpected value. Expected: %#v, got: %#v",
				v.old, v.new, v.in, v.expected, got)
		}
	}
}
