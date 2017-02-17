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
	"fmt"
	"strings"

	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/core/data/id"
)

type variable struct {
	Name   string
	Type   interface{}
	Unique string
	Extra  interface{}
}

func (v*variable) SetExtra(arg interface{}) *variable {
	v.Extra = arg
	return v
}

func (*Templates) Var(t binary.Type, args ...interface{}) *variable {
	all := append([]interface{}{t, "ǂ"}, args...)
	return &variable{
		Name:   fmt.Sprint(args...),
		Type:   t,
		Unique: id.OfString(fmt.Sprint(all...)).String(),
	}
}

func (t *Templates) Call(prefix string, arg interface{}) (string, error) {
	tmpl, err := t.getTemplate(prefix, arg)
	if err != nil {
		return "", err
	}
	return "", tmpl.Execute(t.writer, arg)
}

func (*Templates) Lower(s interface{}) string {
	return strings.ToLower(fmt.Sprint(s))
}

func (*Templates) Upper(s interface{}) string {
	return strings.ToUpper(fmt.Sprint(s))
}

func (*Templates) Contains(test, s interface{}) bool {
	return strings.Contains(fmt.Sprint(s), fmt.Sprint(test))
}

func (*Templates) ToS8(val byte) string {
	return fmt.Sprint(int8(val))
}

func (*Templates) TrimPackage(n string) string {
	i := strings.LastIndex(n, ".")
	if i < 0 {
		return n
	}
	return n[i+1:]
}

func (*Templates) BraceIfNeeded(s string) string {
	r, _ := utf8.DecodeRuneInString(s)
	if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '(' {
		return s
	}
	return fmt.Sprintf("(%s)", s)
}

func (*Templates) Error(format string, args ...interface{}) (string, error) {
	return "", fmt.Errorf(format, args...)
}

func (*Templates) NewStringSet(args ...interface{}) map[string]bool {
	return map[string]bool{}
}

func (*Templates) StringSetContains(set map[string]bool, args ...interface{}) bool {
	s := fmt.Sprint(args...)
	_, found := set[s]
	return found
}

func (*Templates) StringSetAdd(set map[string]bool, args ...interface{}) bool {
	s :=  fmt.Sprint(args...)
	_, found := set[s]
	if !found {
		set[s] = true
	}
	return !found
}

