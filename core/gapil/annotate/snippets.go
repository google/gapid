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

package annotate

import "github.com/google/gapid/core/gapil/snippets"

// newObservation returns a new leader location associated with an observation
func newObservation(ot snippets.ObservationType) *Location {
	return newLocation(snippets.Observation(ot))
}

// newLabel returns the result of an expression with the label name
// associated with it.
func newLabel(name string) *Location {
	return newLocation(snippets.Label(name))
}
