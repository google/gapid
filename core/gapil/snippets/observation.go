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

import (
	"fmt"

	"github.com/google/gapid/framework/binary"
)

// observation is a snippet used to annotate a memory observation
type Observation ObservationType

func (Observation) isSnippet()      {}
func (Observation) finder() findFun { return findObservations }
func (Observation) IsEmpty() bool   { return false }

func (o Observation) String() string {
	return fmt.Sprintf("Observation(%s)", ObservationType(o))
}

// observations, is a collection of observations which is binary codable.
type Observations struct {
	binary.Generate
	path         Pathway
	observations []ObservationType
}

var _ KindredSnippets = &Observations{}

// findObservations can be used to build a observations collection which is a
// binary.Object which contains only observation snippets. Snippets of other
// types are return as an addition return value.
func findObservations(path Pathway, snippets Snippets) (KindredSnippets, Snippets) {
	observations := &Observations{path: path}
	remains := Snippets{}
	for _, s := range snippets {
		if l, ok := s.(Observation); ok {
			observations.observations = append(observations.observations, ObservationType(l))
		} else {
			remains = append(remains, s)
		}
	}
	return observations, remains
}
