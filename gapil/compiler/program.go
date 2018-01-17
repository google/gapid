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

package compiler

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/gapid/core/codegen"
)

type Program struct {
	Settings    Settings
	Commands    map[string]*CommandInfo
	Externs     map[string]*ExternInfo
	Structs     map[string]*StructInfo
	Globals     *StructInfo
	Maps        map[string]*MapInfo
	Locations   []Location
	Module      *codegen.Module
	Initializer codegen.Function // void (ctx* ctx)
}

type CommandInfo struct {
	Function   codegen.Function // void (ctx*, Params*)
	Parameters *codegen.Struct
}

type ExternInfo struct {
	Name       string
	Parameters *codegen.Struct
	Result     codegen.Type
}

type StructInfo struct {
	Type *codegen.Struct
}

type MapInfo struct {
	Type     *codegen.Struct // Maps are held as pointers to these structs
	Elements *codegen.Struct
	Key      codegen.Type
	Val      codegen.Type
	Contains codegen.Function // bool(ctx*, M*, K)
	Index    codegen.Function //   V*(ctx*, M*, K, addIfNotFound)
	Lookup   codegen.Function //   V(ctx*, M*, K)
	Remove   codegen.Function // void(ctx*, M*, K)
	Clear    codegen.Function // void(ctx*, M*)
}

type Location struct {
	File   string
	Line   int
	Column int
}

func (l Location) String() string {
	return fmt.Sprintf("%v:%v:%v", l.File, l.Line, l.Column)
}

func (p *Program) Dump() string {
	//	types := make([]string, 0, len(c.types))
	//	for a, j := range c.types {
	//		ty := strings.Replace(fmt.Sprintf("%v", j), "\n", "\n  ", -1)
	//		types = append(types, fmt.Sprintf("  %v: %v", a.Name(), ty))
	//	}
	//	sort.Strings(types)
	return p.Module.String()
}

var reIRDefineFunc = regexp.MustCompile(`define \w* @(\w*)\([^\)]*\)`)

func (p *Program) IR() map[string]string {
	ir := p.Module.String()
	out := map[string]string{}
	currentFunc, currentIR := "", &bytes.Buffer{}
	flush := func() {
		if currentFunc != "" {
			out[currentFunc] = currentIR.String()
			currentIR.Reset()
			currentFunc = ""
		}
	}
	for _, line := range strings.Split(ir, "\n") {
		matches := reIRDefineFunc.FindStringSubmatch(line)
		if len(matches) == 2 {
			flush()
			currentFunc = matches[1]
			currentIR.WriteString(line)
		} else if currentFunc != "" {
			currentIR.WriteRune('\n')
			currentIR.WriteString(line)
		}
		if line == "}" {
			flush()
			currentFunc = ""
		}
	}
	flush()
	return out
}
