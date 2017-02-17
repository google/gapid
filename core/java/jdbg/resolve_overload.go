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

package jdbg

import (
	"bytes"
	"fmt"
	"strings"
)

// resolveMethodErr is the error used when resolveMethod() fails.
type resolveMethodErr struct {
	class      *Class
	name       string
	args       []interface{}
	candidates methods
	problem    string
}

func (e resolveMethodErr) Error() string {
	name := e.name
	if name == constructor {
		name = e.class.name
	}
	argTys := make([]string, len(e.args))
	for i, a := range e.args {
		switch a := a.(type) {
		case Value:
			argTys[i] = fmt.Sprintf("%v", a.Type())
		default:
			argTys[i] = fmt.Sprintf("%T", a)
		}
	}
	sig := fmt.Sprintf("%v(%v)", name, strings.Join(argTys, ", "))
	problem := fmt.Sprintf(e.problem, sig)
	if len(e.candidates) == 0 {
		return problem
	}
	msg := bytes.Buffer{}
	msg.WriteString(problem)
	msg.WriteString("\n")
	for _, candidate := range e.candidates {
		line, marks := bytes.Buffer{}, bytes.Buffer{}
		line.WriteString(candidate.name)
		line.WriteRune('(')
		marks.WriteString(strings.Repeat(" ", line.Len()))
		for i, param := range candidate.sig.Parameters {
			if i > 0 {
				line.WriteString(", ")
				marks.WriteString("  ")
			}
			ty := param.String()
			line.WriteString(ty)
			if i < len(e.args) && e.class.j.assignable(param, e.args[i]) {
				marks.WriteString(strings.Repeat(" ", len(ty)))
			} else {
				marks.WriteString(strings.Repeat("^", len(ty)))
			}
		}
		msg.WriteString(line.String())
		msg.WriteString(")\n")
		msg.WriteString(marks.String())
		msg.WriteString("\n")
	}
	return msg.String()
}

// resolveMethod attempts to unambiguously resolve a single method with the
// specified name and arguments from the class's method list.
// If considerSuper is true then the all base types will also be searched.
func (j *JDbg) resolveMethod(considerSuper bool, class *Class, name string, args []interface{}) method {
	var methods []method
	for search := class; search != nil; search = search.super {
		methods = j.resolveMethods(search, name, args)
		switch len(methods) {
		case 0:
			break
		case 1:
			return methods[0]
		default:
			j.err(resolveMethodErr{class, name, args, methods, "Ambiguous call to %v. Candidates:"})
		}
		if !considerSuper {
			break
		}
	}
	resolved := class.resolve()
	sameName := resolved.allMethods.filter(func(m method) bool { return m.name == name })
	if len(sameName) > 0 {
		j.err(resolveMethodErr{class, name, args, sameName, "No methods match %v. Candidates:"})
	}
	if considerSuper {
		j.err(resolveMethodErr{class, name, args, resolved.allMethods, "No methods match %v. All methods:"})
	}
	j.err(resolveMethodErr{class, name, args, resolved.methods, "No methods match %v. All methods:"})
	return method{}
}

func (j *JDbg) resolveMethods(t *Class, name string, args []interface{}) []method {
	candidates := t.resolve().methods.filter(func(m method) bool {
		return m.name == name && len(m.sig.Parameters) == len(args)
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates.filter(func(m method) bool { return j.accepts(m.sig, args) })
}
