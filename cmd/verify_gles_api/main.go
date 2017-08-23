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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

var (
	apiPath   = flag.String("api", "", "Filename of the api file to verify (required)")
	cacheDir  = flag.String("cache", "", "Directory for caching downloaded files (required)")
	apiRoot   *semantic.API
	mappings  *resolver.Mappings
	numErrors = 0
)

func main() {
	app.ShortHelp = "stringgen compiles string table files to string packages and a Go definition file."
	app.Run(run)
}

func run(ctx context.Context) error {
	if *apiPath == "" {
		app.Usage(ctx, "Mustsupply api path")
	}
	if *cacheDir == "" {
		app.Usage(ctx, "Must supply cache dir")
	}
	processor := gapil.NewProcessor()
	mappings = processor.Mappings
	api, errs := processor.Resolve(*apiPath)
	if len(errs) > 0 {
		for _, err := range errs {
			PrintError("%v\n", err.Message)
		}
		os.Exit(2)
	}
	apiRoot = api
	reg := DownloadRegistry()
	VerifyApi(reg)
	return nil
}

func PrintError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	numErrors = numErrors + 1
}

func VerifyApi(reg *Registry) {
	VerifyEnum(reg, false)
	VerifyEnum(reg, true)
	for _, cmd := range reg.Command {
		VerifyCommand(reg, cmd)
	}
}

func VerifyEnum(r *Registry, bitfields bool) {
	name := "GLenum"
	if bitfields {
		name = "GLbitfield"
	}
	expected := make(map[string]struct{})
	for _, enums := range r.Enums {
		if (enums.Type == "bitmask") == bitfields && enums.Namespace == "GL" {
			for _, enum := range enums.Enum {
				// Boolean values are handles specially (as labels)
				if enum.Name == "GL_TRUE" || enum.Name == "GL_FALSE" {
					continue
				}
				// The following 64bit values are not proper GLenum values.
				if enum.Name == "GL_TIMEOUT_IGNORED" || enum.Name == "GL_TIMEOUT_IGNORED_APPLE" {
					continue
				}
				if enum.API == "" || enum.API == GLES1API || enum.API == GLES2API {
					var value uint32
					if v, err := strconv.ParseUint(enum.Value, 0, 32); err == nil {
						value = uint32(v)
					} else if v, err := strconv.ParseInt(enum.Value, 0, 32); err == nil {
						value = uint32(v)
					} else {
						PrintError("Failed to parse enum value %v", enum.Value)
						continue
					}
					expected[fmt.Sprintf("%s = 0x%08X", enum.Name, value)] = struct{}{}
				}
			}
		}
	}
	seen := make(map[string]struct{})
	for _, e := range apiRoot.Enums {
		if e.Name() == name {
			for _, m := range e.Entries {
				seen[fmt.Sprintf("%s = 0x%08X", m.Name(), m.Value)] = struct{}{}
			}
		}
	}
	CompareSets(expected, seen, name+": ")
}

func CompareSets(expected, seen map[string]struct{}, msg_prefix string) {
	for k := range expected {
		if _, found := seen[k]; !found {
			PrintError("%sMissing %s\n", msg_prefix, k)
		}
	}
	for k := range seen {
		if _, found := expected[k]; !found {
			PrintError("%sUnexpected %s\n", msg_prefix, k)
		}
	}
	return
}

var re_const_ptr_pre = regexp.MustCompile(`^const (\w+) \*$`)
var re_const_ptr_post = regexp.MustCompile(`^(.+)\bconst\*$`)

func VerifyType(cmd string, paramIndex int, expected string, seen semantic.Type) bool {
	expected = strings.TrimSpace(expected)
	name := seen.(semantic.NamedNode).Name()
	switch s := seen.(type) {
	case *semantic.Pointer:
		if s.Const {
			if match := re_const_ptr_pre.FindStringSubmatch(expected); match != nil {
				return VerifyType(cmd, paramIndex, match[1], s.To)
			}
			if match := re_const_ptr_post.FindStringSubmatch(expected); match != nil {
				return VerifyType(cmd, paramIndex, match[1], s.To)
			}
		} else {
			if strings.HasSuffix(expected, "*") {
				return VerifyType(cmd, paramIndex, strings.TrimSuffix(expected, "*"), s.To)
			}
		}
	case *semantic.Pseudonym:
		if s.Name() == expected {
			return true
		} else if expected == "GLDEBUGPROCKHR" && s.Name() == "GLDEBUGPROC" {
			return true
		} else {
			return VerifyType(cmd, paramIndex, expected, s.To)
		}
	case *semantic.Enum:
		if s.Name() == expected {
			return true
		}
	case *semantic.Builtin:
		if s.Name() == expected {
			return true
		} else if expected == "const GLchar *" && s.Name() == "string" {
			return true
		}
	}
	PrintError("%s: Param %v: Expected type %s but seen %s (%T)\n", cmd, paramIndex, expected, name, seen)
	return false
}

func UniqueStrings(strs []string) (res []string) {
	seen := map[string]struct{}{}
	for _, str := range strs {
		if _, ok := seen[str]; !ok {
			res = append(res, str)
			seen[str] = struct{}{}
		}
	}
	return
}

func VerifyCommand(reg *Registry, cmd *Command) {
	cmdName := cmd.Name()
	versions := append(reg.GetVersions(GLES1API, cmdName), reg.GetVersions(GLES2API, cmdName)...)
	extensions := append(reg.GetExtensions(GLES1API, cmdName), reg.GetExtensions(GLES2API, cmdName)...)
	extensions = UniqueStrings(extensions)
	if len(versions) == 0 && len(extensions) == 0 {
		return // It is not a GLES command.
	}

	// Expected annotations.
	annots := []string{}
	if strings.HasPrefix(cmdName, "glDraw") && !strings.HasPrefix(cmdName, "glDrawBuffers") {
		annots = append(annots, "@draw_call")
	}
	for _, version := range versions {
		if version == Version("1.0") && len(versions) > 1 {
			continue // TODO: Add those in the api file.
		}
		url, _ := GetCoreManpage(version, cmdName)
		annots = append(annots, fmt.Sprintf(`@doc("%s", Version.GLES%v)`, url, strings.Replace(string(version), ".", "", -1)))
	}
	for _, extension := range extensions {
		url, _ := GetExtensionManpage(extension)
		annots = append(annots, fmt.Sprintf(`@doc("%s", Extension.%v)`, url, extension))
	}
	if len(versions) != 0 {
		cond := fmt.Sprintf(`Version.GLES%v`, strings.Replace(string(versions[0]), ".", "", -1))
		if len(versions) == 1 && versions[0] == Version("1.0") {
			cond = "Version.GLES10 && !Version.GLES20" // Deprecated in GLES20
		}
		annots = append(annots, fmt.Sprintf(`@if(%v)`, cond))
	}
	if len(extensions) != 0 {
		conds := []string{}
		for _, extension := range extensions {
			conds = append(conds, fmt.Sprintf(`Extension.%v`, extension))
		}
		annots = append(annots, fmt.Sprintf("@if(%s)", strings.Join(conds, " || ")))
	}

	// Find existing API function.
	var apiCmd *semantic.Function
	for _, apiFunction := range apiRoot.Functions {
		if apiFunction.Name() == cmdName {
			apiCmd = apiFunction
		}
	}

	// Print command to stdout if it is missing.
	if apiCmd == nil {
		params := []string{}
		for _, param := range cmd.Param {
			params = append(params, param.Type()+" "+param.Name)
		}
		fmt.Printf("%s\ncmd %s %s(%s) { }\n", strings.Join(annots, "\n"),
			cmd.Proto.Type(), cmdName, strings.Join(params, ", "))
		return
	}

	// Check documentation strings.
	expected := make(map[string]struct{})
	for _, a := range annots {
		expected[a] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, a := range apiCmd.Annotations {
		if a.Name() == "if" || a.Name() == "doc" || a.Name() == "draw_call" {
			seen[getSource(mappings.CST(a.AST))] = struct{}{}
		}
	}
	CompareSets(expected, seen, fmt.Sprintf("%s: ", cmdName))

	// Check parameter types.
	if len(cmd.Param) != len(apiCmd.CallParameters()) {
		PrintError("%s: Expected %v parameters but seen %v\n", cmdName, len(cmd.Param), len(apiCmd.CallParameters()))
	} else {
		for i, p := range cmd.Param {
			VerifyType(cmdName, i, p.Type(), apiCmd.FullParameters[i].Type)
		}
	}
}

func getSource(n parse.Node) string {
	return string(n.Token().Source.Runes[n.Token().Start:n.Token().End])
}
