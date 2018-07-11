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

package resolver

import (
	"strings"

	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
)

const (
	lineDocStart   = "///"
	blockDocStart  = "/**"
	blockDocBlank  = "*"
	blockDocMiddle = "* "
	blockDocEnd    = "*/"
)

func processSeparator(docs *[]string, s cst.Separator) {
	for _, f := range s {
		s := f.Tok().String()
		switch {
		case strings.HasPrefix(s, lineDocStart):
			line := strings.TrimSpace(strings.TrimPrefix(s, lineDocStart))
			if !strings.HasPrefix(line, "/") {
				*docs = append(*docs, line)
			}
		case strings.HasPrefix(s, blockDocStart):
			s = strings.TrimSuffix(strings.TrimPrefix(s, blockDocStart), blockDocEnd)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if i > 0 {
					if strings.HasPrefix(s, blockDocMiddle) {
						line = strings.TrimPrefix(line, blockDocMiddle)
					} else {
						line = strings.TrimPrefix(line, blockDocBlank)
					}
				}
				*docs = append(*docs, line)
			}
		default:
			continue
		}
		for len(*docs) > 0 && (*docs)[len(*docs)-1] == "" {
			*docs = (*docs)[:len(*docs)-1]
		}
	}
	for len(*docs) > 0 && (*docs)[0] == "" {
		*docs = (*docs)[1:]
	}
}

func (rv *resolver) findDocumentation(node ast.Node) []string {
	cst := rv.mappings.AST.CST(node)
	docs := &[]string{}
	processSeparator(docs, cst.Suffix())
	if len(*docs) == 0 {
		processSeparator(docs, cst.Prefix())
	}
	return *docs
}
