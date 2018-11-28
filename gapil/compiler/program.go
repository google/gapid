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
	"regexp"
	"strings"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/semantic"
)

// Program is the output of a compilation.
type Program struct {
	// Settings used to compile the program.
	Settings Settings

	// APIs compiled for this program.
	APIs []*semantic.API

	// Commands is a map of command name to CommandInfo.
	Commands map[string]*CommandInfo

	// Structs is a map of struct name to StructInfo.
	Structs map[string]*StructInfo

	// Maps is a map of map name to MapInfo.
	Maps map[string]*MapInfo

	// Globals is the StructInfo of all the globals.
	Globals *StructInfo

	// Functions is a map of function name to plugin implemented functions.
	Functions map[string]*codegen.Function

	// Codegen is the codegen module.
	Codegen *codegen.Module

	// Module is the global that holds the generated gapil_module structure.
	Module codegen.Global
}

// CommandInfo holds the generated execute function for a given command.
type CommandInfo struct {
	// The execute function for the given command.
	// The function has the signature: void (ctx*, Params*)
	Execute *codegen.Function

	// The Params structure that is passed to Execute.
	Parameters *codegen.Struct
}

// StructInfo holds the generated structure for a given structure type.
type StructInfo struct {
	Type *codegen.Struct
}

// MapInfo holds the generated map info for a given semantic map type.
type MapInfo struct {
	Type     *codegen.Struct // Maps are held as pointers to these structs
	Elements *codegen.Struct
	Key      codegen.Type
	Val      codegen.Type
	Element  codegen.Type
	MapMethods
}

// MapMethods are the functions that operate on a map.
type MapMethods struct {
	Contains  *codegen.Function // bool(M*, ctx*, K)
	Index     *codegen.Function //   V*(M*, ctx*, K, addIfNotFound)
	Remove    *codegen.Function // void(M*, ctx*, K)
	Clear     *codegen.Function // void(M*, ctx*)
	ClearKeep *codegen.Function // void(M*, ctx*)
}

// Dump returns the full LLVM IR for the program.
func (p *Program) Dump() string {
	return p.Codegen.String()
}

var reIRDefineFunc = regexp.MustCompile(`define \w* @(\w*)\([^\)]*\)`)

// IR returns a map of function to IR.
func (p *Program) IR() map[string]string {
	ir := p.Codegen.String()
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
