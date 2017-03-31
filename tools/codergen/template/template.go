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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/text/reflow"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/tools/codergen/generate"
)

// Templates manages the loaded templates and executes them on demand.
// It exposes all its public methods and fields to the templates.
type Templates struct {
	templates *template.Template
	funcs     template.FuncMap
	active    *template.Template
	writer    io.Writer
	File      interface{}
}

func isPublic(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func installMethods(v reflect.Value, funcs template.FuncMap) {
	ty := v.Type()
	for i := 0; i < ty.NumMethod(); i++ {
		m := ty.Method(i)
		if isPublic(m.Name) {
			funcs[m.Name] = v.Method(i).Interface()
		}
	}
}

func installFields(v reflect.Value, funcs template.FuncMap) {
	ty := v.Type()
	for i := 0; i < ty.NumField(); i++ {
		m := ty.Field(i)
		if isPublic(m.Name) {
			funcs[m.Name] = v.Field(i).Interface
		}
	}
}

// New constructs and returns a new template set.
// This parses all embedded templates automatically.
func New() *Templates {
	f := &Templates{
		templates: template.New("FunctionHolder"),
		funcs: template.FuncMap{
			// fake builtin functions
			"add":    func(a, b int) int { return a + b },
			"sub":    func(a, b int) int { return a - b },
			"concat": func(a, b string) string { return a + b },
		},
	}
	v := reflect.ValueOf(f)
	installMethods(v, f.funcs)
	installFields(v.Elem(), f.funcs)
	f.templates.Funcs(f.funcs)
	for name, content := range embedded {
		template.Must(f.templates.New(name).Parse(content))
	}
	return f
}

// Generate is an implementation of generate.Generator
func (t *Templates) Generate(g generate.Generate) (bool, error) {
	t.File = g.Arg
	defer func() { t.File = nil }()
	old, _ := ioutil.ReadFile(g.Output)
	buf := &bytes.Buffer{}
	out := reflow.New(buf)
	out.Indent = g.Indent

	sections, err := SectionSplit(old)
	if err != nil {
		return false, err
	}
	if len(sections) > 0 {
		// we are doing a partial update...
		for _, s := range sections {
			if s.Name == "" {
				// copy the non template section straight to the underlying buf
				buf.Write(s.Body)
			} else {
				// Write the start marker straight to the underlying buf
				buf.Write(s.StartMarker)
				out.Depth = s.Indentation
				// Execute the template to generate a new body
				if err := t.execute(s.Name, out, g.Arg); err != nil {
					return false, err
				}
				// write the end marker out through the formatting writer
				out.Write(s.EndMarker)
				out.Flush()
			}
		}
	} else {
		if err := t.execute(g.Name, out, g.Arg); err != nil {
			return false, err
		}
		out.Flush()
	}
	data := buf.Bytes()
	if g.Output == "" || bytes.Equal(data, old) {
		return false, nil
	}
	dir, _ := filepath.Split(g.Output)
	if len(dir) > 0 {
		os.MkdirAll(dir, os.ModePerm)
	}
	return true, ioutil.WriteFile(g.Output, data, 0666)
}

func appendTypeMatches(try []string, prefix string, node binary.Type) []string {
	try = append(try, fmt.Sprint(prefix, "#", node.String()))
	if node.String() != node.Representation() {
		try = append(try, fmt.Sprint(prefix, "#", node.Representation()))
	}
	return try
}

func (t *Templates) getTemplate(prefix string, node interface{}) (*template.Template, error) {
	try := []string{}
	switch node := node.(type) {
	case *schema.Slice:
		try = appendTypeMatches(try, prefix, node)
		if node.Alias != "" {
			try = append(try, fmt.Sprint(prefix, ".Slice.Alias"))
		}
		try = append(try, fmt.Sprint(prefix, ".Slice#", node.ValueType.String()))
	case *schema.Array:
		try = appendTypeMatches(try, prefix, node)
		if node.Alias != "" {
			try = append(try, fmt.Sprint(prefix, ".Array.Alias"))
		}
		try = append(try, fmt.Sprint(prefix, ".Array#", node.ValueType.String()))
	case *schema.Primitive:
		try = appendTypeMatches(try, prefix, node)
		if node.String() != node.Representation() {
			try = append(try, fmt.Sprint(prefix, ".Alias"))
		}
	case binary.Type:
		try = appendTypeMatches(try, prefix, node)
	case *variable:
		return t.getTemplate(prefix, node.Type)
	case string:
		try = append(try, prefix+node)
	case schema.Method:
		try = append(try, fmt.Sprint(prefix, "#", node.String()))
	case *generate.Struct:
		try = append(try, fmt.Sprint(prefix, "#", node.Package, ".", node.Name()))
	case nil:
	default:
		panic(fmt.Sprintf("Unknown %T", node))
	}
	if node != nil {
		r := reflect.TypeOf(node)
		// using the reflected typename
		if r.Name() != "" {
			try = append(try, fmt.Sprint(prefix, ".", r.Name()))
		}
		if r.Kind() == reflect.Ptr {
			try = append(try, fmt.Sprint(prefix, ".", r.Elem().Name()))
		}
	} else {
		try = append(try, fmt.Sprint(prefix, ".nil"))
	}
	// default case is just the prefix
	try = append(try, prefix)
	for _, name := range try {
		if tmpl := t.templates.Lookup(name); tmpl != nil {
			return tmpl, nil
		}
	}
	return nil, fmt.Errorf(`Cannot find templates "%s"`, strings.Join(try, `","`))
}

func (t *Templates) execute(name string, w io.Writer, data interface{}) error {
	oldw := t.writer
	if w != nil {
		t.writer = w
	}
	defer func() { t.writer = oldw }()
	tmpl := t.templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("Cannot find template %s", name)
	}
	return tmpl.Execute(w, data)
}
