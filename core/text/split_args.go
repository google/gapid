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

package text

import (
	"bytes"
)

// SplitArgs splits and returns the string s separated by non-quoted whitespace.
// Useful for splitting a string containing multiple command-line parameters.
func SplitArgs(s string) []string {
	out := []string{}
	b := bytes.Buffer{}
	runes := ([]rune)(s)
	var escaping, inQuote bool
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range runes {
		switch {
		case !escaping && r == '\\':
			escaping = true
		case !escaping && !inQuote && r == '"':
			inQuote, escaping = true, false
		case !escaping && inQuote && r == '"':
			inQuote, escaping = false, false
		case !inQuote && r == ' ':
			flush()
		default:
			b.WriteRune(r)
			escaping = false
		}
	}
	flush()
	return out
}

// Quote takes a slice of strings, and returns a slice where each
// string has been quoted and escaped to pass down to the command-line.
func Quote(s []string) []string {
	out := []string{}

	for _, str := range s {
		hasSpace := false
		outStr := ""
		for _, c := range str {
			if c == '"' {
				outStr += "\\\""
			} else {
				outStr += string(c)
			}
			if c == '\\' {
				outStr += "\\\\"
			}

			if c == ' ' {
				hasSpace = true
			}
		}

		if hasSpace {
			out = append(out, "\""+outStr+"\"")
		} else {
			out = append(out, outStr)
		}
	}
	return out
}
