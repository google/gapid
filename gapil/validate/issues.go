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

package validate

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/text/parse/cst"
)

// Issue holds details of a problem found when validating an API file.
type Issue struct {
	At      cst.Fragment
	Problem interface{}
}

func (i Issue) String() string {
	if at := i.At; at != nil {
		return fmt.Sprintf("%v %v", at.Tok().At(), i.Problem)
	}
	return fmt.Sprintf("%v", i.Problem)
}

// Issues is a list of issues.
type Issues []Issue

func (l Issues) String() string {
	lines := make([]string, len(l))
	for i, s := range l {
		lines[i] = s.String()
	}
	return strings.Join(lines, "\n")
}

func (l *Issues) add(at cst.Fragment, problem interface{}) {
	(*l) = append(*l, Issue{At: at, Problem: problem})
}

func (l *Issues) addf(at cst.Fragment, msg string, args ...interface{}) {
	l.add(at, fmt.Errorf(msg, args...))
}

func (l Issues) Len() int      { return len(l) }
func (l Issues) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l Issues) Less(i, j int) bool {
	a, b := l[i].At, l[j].At
	if a != nil && b != nil {
		return a.Tok().Less(b.Tok())
	}
	return fmt.Sprint(l[i].Problem) < fmt.Sprint(l[j].Problem)
}
