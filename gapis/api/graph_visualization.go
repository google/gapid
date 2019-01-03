// Copyright (C) 2018 Google Inc.
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

type Hierarchy struct {
	LevelsID []int
}

func (h *Hierarchy) GetSize() int {
	return len(h.LevelsID)
}

func (h *Hierarchy) GetID(level int) int {
	return h.LevelsID[level-1]
}

func (h *Hierarchy) PopBack() {
	if len(h.LevelsID) > 0 {
		h.LevelsID = h.LevelsID[:len(h.LevelsID)-1]
	}
}

func (h *Hierarchy) PushBackToResize(newSize int) {
	for len(h.LevelsID) < newSize {
		h.LevelsID = append(h.LevelsID, 0)
	}
}

func (h *Hierarchy) PopBackToResize(newSize int) {
	for len(h.LevelsID) > newSize {
		h.PopBack()
	}
}

func (h *Hierarchy) IncreaseIDByOne(level int) {
	h.LevelsID[level-1]++
}

type GraphVisualizationAPI interface {
	GetCommandLabel(currentHierarchy *Hierarchy, command Cmd) string

	GetSubCommandLabel(index SubCmdIdx) string
}
