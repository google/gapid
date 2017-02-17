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

package langsvr

import "github.com/google/gapid/core/langsvr/protocol"

// SourceCode contains some source code and a language specifier.
type SourceCode struct {
	// The language the source is in.
	Language string

	// The source code.
	Source string
}

// SourceCodeList is a list of source code snippits.
type SourceCodeList []SourceCode

// Add appends the source code snippet with the specified language and source.
func (l *SourceCodeList) Add(lang, source string) {
	*l = append(*l, SourceCode{Language: lang, Source: source})
}

func (s SourceCode) toProtocol() protocol.MarkedString {
	return protocol.MarkedString{
		Language: s.Language,
		Value:    s.Source,
	}
}

func (l SourceCodeList) toProtocol() []protocol.MarkedString {
	out := make([]protocol.MarkedString, len(l))
	for i, s := range l {
		out[i] = s.toProtocol()
	}
	return out
}
