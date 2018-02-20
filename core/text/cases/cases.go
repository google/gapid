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

// Package cases contains functions for Maping strings between various
// cases (snake, pascal, etc).
package cases

import (
	"bytes"
	"strings"
	"unicode"
)

// Words are a list of strings.
type Words []string

// Snake separates and returns the words in s by underscore.
func Snake(s string) Words {
	if s == "" {
		// strings.Split() of an empty string returns a list containing a single
		// empty string. This is undesirable.
		return Words{}
	}
	return Words(strings.Split(s, "_"))
}

// Pascal separates and returns the words in s, where each word begins with a
// uppercase letter.
func Pascal(s string) Words {
	out := Words{}
	buf := bytes.Buffer{}
	wasLetter := false
	for _, r := range []rune(s) {
		if wasLetter && unicode.IsUpper(r) {
			// Start of new word. Flush buf.
			out = append(out, buf.String())
			buf.Reset()
		}
		buf.WriteRune(r)
		wasLetter = unicode.IsLetter(r)
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
		buf.Reset()
	}
	return out
}

// Camel separates and returns the words in s, where each new word begins with
// an uppercase letter.
// Camel parses identically to Pascal(), but is declared for symmetry with
// Words.ToCamel().
func Camel(s string) Words {
	return Pascal(s)
}

// ToSnake returns all the words concatenated with an underscore.
func (w Words) ToSnake() string {
	return strings.Join([]string(w), "_")
}

// ToPascal returns all the words concatenated with each word beginning with an
// uppercase letter.
func (w Words) ToPascal() string {
	buf := bytes.Buffer{}
	for _, word := range w {
		for i, r := range []rune(word) {
			if i == 0 {
				r = unicode.ToUpper(r)
			}
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// ToCamel returns all the words concatenated with each word beginning with
// an uppercase letter, except for the first.
func (w Words) ToCamel() string {
	buf := bytes.Buffer{}
	for i, word := range w {
		for j, r := range []rune(word) {
			if i > 0 && j == 0 {
				r = unicode.ToUpper(r)
			}
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// ToLower converts all the words to lowercase.
func (w Words) ToLower() Words {
	for i, word := range w {
		w[i] = strings.ToLower(word)
	}
	return w
}

// ToUpper converts all the words to uppercase.
func (w Words) ToUpper() Words {
	for i, word := range w {
		w[i] = strings.ToUpper(word)
	}
	return w
}

// Map returns a new list of words, each mapped by f.
func (w Words) Map(f func(string) string) Words {
	out := make(Words, len(w))
	for i, word := range w {
		out[i] = f(word)
	}
	return out
}

// Title capitalizes the first letter of each word.
func (w Words) Title() Words {
	return w.Map(Title)
}

// Untitle lower-cases the first letter of each word.
func (w Words) Untitle() Words {
	return w.Map(Untitle)
}

// Title capitalizes the first letter of the string.
func Title(s string) string {
	first := true
	return strings.Map(func(r rune) rune {
		if first {
			first = false
			return unicode.ToTitle(r)
		}
		return r
	}, s)
}

// Untitle lower-cases the first letter of the string.
func Untitle(s string) string {
	first := true
	return strings.Map(func(r rune) rune {
		if first {
			first = false
			return unicode.ToLower(r)
		}
		return r
	}, s)
}
