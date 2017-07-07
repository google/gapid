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

package api

// CmdIDSet is a set of CmdIDs.
type CmdIDSet map[CmdID]struct{}

// Remove removes id from the set. If the id was not in the set then the call
// does nothing.
func (s *CmdIDSet) Remove(id CmdID) {
	delete(*s, id)
}

// Add adds id to the set. If the id was already in the set then the call does
// nothing.
func (s *CmdIDSet) Add(id CmdID) {
	(*s)[id] = struct{}{}
}

// Contains returns true if id is in the set, otherwise false.
func (s CmdIDSet) Contains(id CmdID) bool {
	_, ok := s[id]
	return ok
}
