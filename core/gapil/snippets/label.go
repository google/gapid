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

package snippets

import "github.com/google/gapid/framework/binary"

// Label a string label snippet
type Label string

func (Label) isSnippet()      {}
func (Label) finder() findFun { return findLabels }
func (l Label) IsEmpty() bool { return len(l) == 0 }

var _ Snippet = Label("")

// Labels, is a collection of labels which is binary codable.
type Labels struct {
	binary.Generate
	Path   Pathway
	Labels []string
}

var _ KindredSnippets = &Labels{}

// findLabels can be used to build a labels collection which is a
// binary.Object which contains only labels snippets. Snippets of other
// types are return as an additional return value.
func findLabels(path Pathway, snippets Snippets) (KindredSnippets, Snippets) {
	labels := &Labels{Path: path}
	remains := Snippets{}
	for _, s := range snippets {
		if l, ok := s.(Label); ok {
			labels.Labels = append(labels.Labels, string(l))
		} else {
			remains = append(remains, s)
		}
	}
	return labels, remains
}
