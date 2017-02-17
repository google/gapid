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

package copyright

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const generatedPrefix = "generated"
const externalPrefix = "external"

var (
	current   = Info{Year: strconv.Itoa(time.Now().Year())}
	languages = []*Language{
		{
			Name:       "go",
			License:    "apache_cpp",
			Extensions: []string{".go"},
		},
		{
			Name:       "c",
			License:    "apache_c",
			Extensions: []string{".c", ".cc", ".cpp", ".h", ".hpp", ".inl", ".java", ".mm"},
		},
		{
			Name:       "api",
			License:    "apache_cpp",
			Extensions: []string{".api"},
		},
		{
			Name:       "template",
			License:    "apache_tmpl",
			Extensions: []string{".tmpl"},
		},
		{
			Name:       "javascript",
			License:    "apache_c",
			Extensions: []string{".js"},
		},
		{
			Name:       "cmake",
			License:    "apache_cmake",
			Extensions: []string{".cmake", ".cmake.in", "CMakeLists.txt"},
		},
		{
			Name:       "lingo",
			License:    "apache_lingo",
			Extensions: []string{".lingo"},
		},
		{
			Name:       "proto",
			License:    "apache_proto",
			Extensions: []string{".proto"},
		},
	}
	External  = []*regexp.Regexp{}
	Generated = []*regexp.Regexp{}
	Normal    = []*regexp.Regexp{}
)

type Info struct {
	Year    string
	Tool    string
	Version string
	Source  string
}

type Language struct {
	Name       string
	Extensions []string
	License    string
	Emit       string
	Current    []*regexp.Regexp
	Old        []*regexp.Regexp
}

func init() {
	for _, l := range languages {
		l.Emit = Build(l.License, current)
		l.Current = []*regexp.Regexp{Regexp(l.License, current, false)}
		l.Old = []*regexp.Regexp{Regexp(l.License, Info{}, true)}
	}
	for name := range embedded {
		re := Regexp(name, Info{}, true)
		if strings.HasPrefix(name, externalPrefix) {
			External = append(External, re)
		} else if strings.HasPrefix(name, generatedPrefix) {
			Generated = append(Generated, re)
		} else {
			Normal = append(Normal, re)
		}
	}
}

func get(name string) string {
	header, ok := embedded[name]
	if !ok {
		panic(fmt.Errorf("Invalid header name %s", name))
	}
	return header
}

func build(name string, header string, i Info) string {
	funcs := template.FuncMap{
		"Year":    func() string { return i.Year },
		"Tool":    func() string { return i.Tool },
		"Version": func() string { return i.Version },
		"Source":  func() string { return i.Source },
	}
	t := template.Must(template.New(name).
		Delims("«", "»").
		Funcs(funcs).
		Parse(header))
	b := &bytes.Buffer{}
	if err := t.Execute(b, i); err != nil {
		panic(fmt.Errorf("Error building %s: %s", name, err))
	}
	return b.String()
}

func Build(name string, i Info) string {
	return build(name, get(name), i)
}

func Regexp(name string, i Info, trim bool) *regexp.Regexp {
	if i.Year == "" {
		i.Year = `(.*)`
	}
	if i.Tool == "" {
		i.Tool = `(.*)`
	}
	if i.Version == "" {
		i.Version = `(.*)`
	}
	if i.Source == "" {
		i.Source = `(.*)`
	}
	header := get(name)
	if trim {
		header = `^\s*` + regexp.QuoteMeta(strings.TrimSpace(header)) + `\s*`
	} else {
		header = `^` + regexp.QuoteMeta(header)
	}
	return regexp.MustCompile(build(name, header, i))
}

func FindLanguage(name string) *Language {
	for _, l := range languages {
		if l.Name == name {
			return l
		}
	}
	return nil
}

func FindExtension(ext string) *Language {
	for _, l := range languages {
		for _, e := range l.Extensions {
			if strings.HasSuffix(ext, e) {
				return l
			}
		}
	}
	return nil
}

func match(file []byte, list []*regexp.Regexp) int {
	for _, re := range list {
		match := re.Find(file)
		if len(match) > 0 {
			return len(match)
		}
	}
	return 0
}

func (l *Language) MatchCurrent(file []byte) int {
	return match(file, l.Current)
}

func (l *Language) MatchOld(file []byte) int {
	return match(file, l.Old)
}

func MatchNormal(file []byte) int {
	return match(file, Normal)
}

func MatchExternal(file []byte) int {
	return match(file, External)
}

func MatchGenerated(file []byte) int {
	return match(file, Generated)
}
