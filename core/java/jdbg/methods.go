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
	"fmt"
	"strings"

	"github.com/google/gapid/core/java/jdwp"
)

const (
	constructor = "<init>"
)

// method describes a Java function.
type method struct {
	id    jdwp.MethodID
	mod   jdwp.ModBits
	name  string
	sig   methodSignature
	class *Class
}

func (m method) String() string {
	args := make([]string, len(m.sig.Parameters))
	for i, a := range m.sig.Parameters {
		args[i] = fmt.Sprintf("%v", a)
	}
	if m.name == constructor {
		return fmt.Sprintf("%v %v(%v)", m.mod, m.class.String(), strings.Join(args, ", "))
	}
	return fmt.Sprintf("%v %v %v.%v(%v)", m.mod, m.sig.Return, m.class.String(), m.name, strings.Join(args, ", "))
}

// methods is a list of methods.
type methods []method

func (m methods) String() string {
	lines := make([]string, len(m))
	for i, m := range m {
		lines[i] = m.String()
	}
	return strings.Join(lines, "\n")
}

// filter returns a copy of the method list with the methods that fail the
// predicate test removed.
func (m methods) filter(predicate func(m method) bool) methods {
	out := make(methods, 0, len(m))
	for _, m := range m {
		if predicate(m) {
			out = append(out, m)
		}
	}
	return out
}
