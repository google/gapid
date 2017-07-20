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
	"unicode"

	"github.com/google/gapid/gapil/resolver"
)

type nameOptions struct {
	PreserveSpecial   bool
	UntitleFirst      bool
	TitleFirst        bool
	TitleAfterNumber  bool
	UnderscoreToTitle bool
	Remap             map[string]string
}

var (
	specials = map[rune]rune{
		':': '_',
		'!': '_',
	}
	nonascii = map[rune]rune{
		resolver.RefRune:     'R',
		resolver.SliceRune:   'S',
		resolver.ConstRune:   'C',
		resolver.PointerRune: 'P',
		resolver.ArrayRune:   'A',
		resolver.MapRune:     'M',
		resolver.TypeRune:    'T',
	}
	goKeywords = map[string]string{
		"type": "ty",
	}
)

// NameOf returns the name of the supplied object if it has one.
func (Functions) NameOf(obj interface{}) string {
	return nameOf(obj)
}

// CommandName converts an api name to a command name.
func (Functions) CommandName(obj interface{}) string {
	return nameOptions{}.convert(nameOf(obj))
}

// GoCommandName converts an api name to the public go command name form.
func (Functions) GoCommandName(obj interface{}) string {
	return nameOptions{
		TitleFirst: true,
		Remap:      goKeywords,
	}.convert(nameOf(obj))
}

// GoPublicName converts an api name to the public go form.
func (Functions) GoPublicName(obj interface{}) string {
	return nameOptions{
		TitleFirst:        true,
		UnderscoreToTitle: true,
		PreserveSpecial:   true,
		Remap:             goKeywords,
	}.convert(nameOf(obj))
}

// GoPrivateName converts an api name to the public go form.
func (Functions) GoPrivateName(obj interface{}) string {
	return nameOptions{
		UntitleFirst:      true,
		UnderscoreToTitle: true,
		PreserveSpecial:   true,
		Remap:             goKeywords,
	}.convert(nameOf(obj))
}

// ProtoName converts an api name to the proto name.
func (Functions) ProtoName(obj interface{}) string {
	return nameOptions{}.convert(nameOf(obj))
}

// ProtoGoName converts an api name to the go name produced by the proto compiler.
func (Functions) ProtoGoName(obj interface{}) string {
	return nameOptions{
		TitleFirst:        true,
		TitleAfterNumber:  true,
		UnderscoreToTitle: true,
		Remap:             goKeywords,
	}.convert(nameOf(obj))
}

type named interface {
	Name() string
}

func nameOf(obj interface{}) string {
	switch v := obj.(type) {
	case string:
		return v
	case named:
		return v.Name()
	default:
		return fmt.Sprint(v)
	}
}

func (o nameOptions) convert(name string) string {
	out := ""
	titleNext := o.TitleFirst
	skip := false
	prefix := ""
	for i, r := range name {
		title := titleNext
		titleNext = false
		skipped := skip
		skip = false
		prefixed := prefix != ""
		prefix = ""
		if !o.PreserveSpecial {
			if sub, found := specials[r]; found {
				r = sub
			}
			if sub, found := nonascii[r]; found {
				prefix = "__"
				r = sub
			}
		}
		switch {
		case unicode.IsNumber(r):
			if o.TitleAfterNumber {
				titleNext = true
			}
		case r == '_':
			if o.UnderscoreToTitle {
				titleNext = true
				skip = true
				continue
			}
		}
		if prefix != "" && !prefixed {
			out += prefix
		}
		if i == 0 && o.UntitleFirst {
			r = unicode.ToLower(r)
		}
		if title {
			t := unicode.ToTitle(r)
			if r == t && skipped {
				out += "_"
			}
			r = t
		}
		out += string(r)
	}
	if remap, found := o.Remap[out]; found {
		return remap
	}
	return out
}
