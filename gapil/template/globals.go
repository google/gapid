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
	"path/filepath"
	"strings"
)

type globalMap map[string]interface{}

func initGlobals(f *Functions, globalList []string) {
	apiBase := filepath.Base(f.apiFile)
	f.globals["API"] = strings.TrimSuffix(apiBase, filepath.Ext(apiBase))
	f.globals["OutputDir"] = filepath.Base(f.basePath)
	if i := strings.LastIndex(f.basePath, "genfiles"+string(filepath.Separator)); i > 0 {
		// GeneratedPath is the bazel genfiles relative path of the current
		// template. Assumes the build system is bazel.
		f.globals["GeneratedPath"] = f.basePath[i+9:]
	}
	for _, g := range globalList {
		v := strings.SplitN(g, "=", 2)
		f.globals[v[0]] = v[1]
	}
}

// Gets or sets a template global variable
// Example:
//
//	{{Global "CatSays" "Meow"}}
//	The cat says: {{Global "CatSays"}}
func (f *Functions) Global(name string, values ...interface{}) (interface{}, error) {
	switch len(values) {
	case 0:
		if value, ok := f.globals[name]; ok {
			return value, nil
		} else {
			return "", nil
		}
	case 1:
		f.globals[name] = values[0]
		return "", nil
	default:
		f.globals[name] = values
		return "", nil
	}
}

// Increments and returns a global variable
// Example:
//
//	{{Global "ProtoID" 0}}
//	bool field = {{Inc "ProtoID"}}
func (f *Functions) Inc(name string) (interface{}, error) {
	g, found := f.globals[name]
	if !found {
		return nil, fmt.Errorf("Inc called for invalid global %s", name)
	}
	v, ok := g.(int)
	if !ok {
		return nil, fmt.Errorf("Inc called for non numeric value %s (was %T)", name, g)
	}
	v++
	f.globals[name] = v
	return v, nil
}
