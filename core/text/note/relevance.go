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

package note

import "strconv"
import "encoding/json"

type (
	// Relevance is the relative importance of a piece of information.
	// It is used by styles to decide whether information should be printed or not.
	// Relevance can be used directly as a section marker.
	Relevance int
)

const (
	// Unknown is the zero value of importance, and means accept the parent importance.
	Unknown = Relevance(iota)
	// Critical should always be printed, even in raw modes.
	Critical
	// Important information is printed by all normal loggers.
	Important
	// Relevant information is printed in standard logging modes.
	Relevant
	// Irrelevant is a marker for items that should ony appear in verbose modes.
	Irrelevant
	// Never is a relevance end marker.
	Never
)

// OmitKey returns true if the key can be omitted when printing.
func OmitKey(key interface{}) bool {
	v, matched := key.(OmitKeyMarker)
	return matched && v.OmitKey()
}

// String returns a readable representation of the relevance.
func (r Relevance) String() string {
	switch r {
	case Unknown:
		return "Unknown"
	case Critical:
		return "Critical"
	case Important:
		return "Important"
	case Relevant:
		return "Relevant"
	case Irrelevant:
		return "Irrelevant"
	default:
		return strconv.Itoa(int(r))
	}
}

// MarshalJSON writes the relevance as a JSON string
func (r Relevance) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// OrderBy returns string sort key for the relevance, such that items come in decreasing order of importance
func (r Relevance) OrderBy() string {
	return strconv.Itoa(int(r))
}
