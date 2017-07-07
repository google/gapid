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

/*
Package glsl contains routines for manipulation of OpenGL ES Shading Language programs.

It exposes functions for parsing, serializing and evaluating GLES Shading Language programs.
While this package contains a number of sub-packages, the only sub-package which is expected
to be imported directly is the ast package, which contains the definitions of the AST of the
parsed program. The main functionality of the other packages is exposed through the functions
of this package.
*/
package glsl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/evaluator"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
	pp "github.com/google/gapid/gapis/api/gles/glsl/preprocessor"
	"github.com/google/gapid/gapis/api/gles/glsl/sema"
)

// Version describes a GLSL shader version.
type Version struct {
	Major, Minor, Point int
}

// String returns the string form of the GLSL version.
func (v Version) String() string {
	return fmt.Sprintf("%d%d%d", v.Major, v.Minor, v.Point)
}

// GreaterThan returns true if this Version is greater than Version{major, minor}.
func (v Version) GreaterThan(major, minor int) bool {
	switch {
	case v.Major > major:
		return true
	case v.Major < major:
		return false
	case v.Minor > minor:
		return true
	default:
		return false
	}
}

// ParseVersion parses and returns the Version from the string s.
func ParseVersion(s string) Version {
	if i, err := strconv.Atoi(s); err == nil {
		major := (i / 100) % 10
		minor := (i / 10) % 10
		point := i % 10
		return Version{Major: major, Minor: minor, Point: point}
	} else {
		return Version{Major: 1}
	}
}

// Parse preprocesses and parses an OpenGL ES Shading language program present in the first
// argument. The second argument specifies the language, whose syntax to employ during parsing.
// The parsed AST is returned in the first result. If any parsing errors are encountered, they
// are returned in the second result.
func Parse(src string, lang ast.Language) (program interface{}, version Version, extensions []pp.Extension, err []error) {
	prog, v, exts, err := parser.Parse(src, lang, evaluator.EvaluatePreprocessorExpression)
	return prog, ParseVersion(v), exts, err
}

// Formatter is a helper function which turns any AST node into something that can be printed with
// %v. The returned object's default format will print the tree under the ast node in a reindented
// form. The alternate format flag (%#v) will print the node while preserving original whitespace,
// if this is present in the ***Cst nodes of the tree.
func Formatter(node interface{}) fmt.Formatter { return parser.Formatter(node) }

// Analyze performs semantic analysis on the parsed program AST. It computes the types of all program
// expression, array sizes and values of constant variables. Any encountered errors are returned
// as a result.
func Analyze(program interface{}) (err []error) { return sema.Analyze(program, evaluator.Evaluate) }

// Format returns the source for the given shader AST tree and version.
func Format(tree interface{}, version Version, extensions []pp.Extension) string {
	src := fmt.Sprintf("%v", Formatter(tree))
	if version.Point == 0 && version.Minor == 0 && version.Major == 0 {
		return src
	}
	header := fmt.Sprintf("#version %v", version)
	for _, extension := range extensions {
		header += fmt.Sprintf("\n#extension %s : %s", extension.Name, extension.Behaviour)
	}
	if !strings.HasPrefix(src, "\n") {
		header += "\n"
	}
	return header + src
}
