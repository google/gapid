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

package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

type Status struct {
	NotUpdated                []string
	UpdatedInIndex            []string
	AddedToIndex              []string
	DeletedFromIndex          []string
	RenamedInIndex            []string
	CopiedInIndex             []string
	IndexAndWorkTreeMatches   []string
	WorkTreeChangedSinceIndex []string
	DeletedInWorkTree         []string
	UnmergedBothDeleted       []string
	UnmergedAddedByUs         []string
	UnmergedDeletedByThem     []string
	UnmergedAddedByThem       []string
	UnmergedDeletedByUs       []string
	UnmergedBothAdded         []string
	UnmergedBothModified      []string
	Untracked                 []string
	Ignored                   []string
}

// Clean returns true if the status is empty (no file changes).
func (s Status) Clean() bool {
	return len(s.NotUpdated) == 0 &&
		len(s.UpdatedInIndex) == 0 &&
		len(s.AddedToIndex) == 0 &&
		len(s.DeletedFromIndex) == 0 &&
		len(s.RenamedInIndex) == 0 &&
		len(s.CopiedInIndex) == 0 &&
		len(s.IndexAndWorkTreeMatches) == 0 &&
		len(s.WorkTreeChangedSinceIndex) == 0 &&
		len(s.DeletedInWorkTree) == 0 &&
		len(s.UnmergedBothDeleted) == 0 &&
		len(s.UnmergedAddedByUs) == 0 &&
		len(s.UnmergedDeletedByThem) == 0 &&
		len(s.UnmergedAddedByThem) == 0 &&
		len(s.UnmergedDeletedByUs) == 0 &&
		len(s.UnmergedBothAdded) == 0 &&
		len(s.UnmergedBothModified) == 0 &&
		len(s.Untracked) == 0 &&
		len(s.Ignored) == 0
}

func (s Status) String() string {
	b := bytes.Buffer{}
	append := func(name string, files []string) {
		if len(files) > 0 {
			b.WriteString(name)
			b.WriteString(": \n")
			for _, file := range files {
				b.WriteString("    ")
				b.WriteString(file)
				b.WriteString("\n")
			}
		}
	}
	append("Not updated", s.NotUpdated)
	append("Updated in index", s.UpdatedInIndex)
	append("Added to index", s.AddedToIndex)
	append("Deleted from index", s.DeletedFromIndex)
	append("Renamed in index", s.RenamedInIndex)
	append("Copied in index", s.CopiedInIndex)
	append("Index and work tree matches", s.IndexAndWorkTreeMatches)
	append("Work tree changed since index", s.WorkTreeChangedSinceIndex)
	append("Deleted in work tree", s.DeletedInWorkTree)
	append("Unmerged, both deleted", s.UnmergedBothDeleted)
	append("Unmerged, added by us", s.UnmergedAddedByUs)
	append("Unmerged, deleted by them", s.UnmergedDeletedByThem)
	append("Unmerged, added by them", s.UnmergedAddedByThem)
	append("Unmerged, deleted by us", s.UnmergedDeletedByUs)
	append("Unmerged, both added", s.UnmergedBothAdded)
	append("Unmerged, both modified", s.UnmergedBothModified)
	append("Untracked", s.Untracked)
	append("Ignored", s.Ignored)
	return b.String()
}

// Status performs a `git status` call.
func (g Git) Status(ctx context.Context) (Status, error) {
	str, _, err := g.run(ctx, "status", "-z")
	if err != nil {
		return Status{}, err
	}
	any := func(r byte, options string) bool {
		for _, x := range []byte(options) {
			if r == x {
				return true
			}
		}
		return false
	}
	status := Status{}
	for _, line := range strings.Split(str, "\x00") {
		// each line takes the form: 'XY <path>'
		if len(line) < 3 {
			continue
		}
		x := line[0]
		y := line[1]
		file := line[3:]
		switch {
		case x == ' ' && any(y, "MD"):
			status.NotUpdated = append(status.NotUpdated, file)
		case x == 'M' && any(y, " MD"):
			status.UpdatedInIndex = append(status.UpdatedInIndex, file)
		case x == 'A' && any(y, " MD"):
			status.AddedToIndex = append(status.AddedToIndex, file)
		case x == 'D' && any(y, " M"):
			status.DeletedFromIndex = append(status.DeletedFromIndex, file)
		case x == 'R' && any(y, " MD"):
			status.RenamedInIndex = append(status.RenamedInIndex, file)
		case x == 'C' && any(y, " MD"):
			status.CopiedInIndex = append(status.CopiedInIndex, file)
		case any(x, "MARC") && y == ' ':
			status.IndexAndWorkTreeMatches = append(status.IndexAndWorkTreeMatches, file)
		case any(x, " MARC") && y == 'M':
			status.WorkTreeChangedSinceIndex = append(status.WorkTreeChangedSinceIndex, file)
		case any(x, " MARC") && y == 'D':
			status.DeletedInWorkTree = append(status.DeletedInWorkTree, file)
		case x == 'D' && y == 'D':
			status.UnmergedBothDeleted = append(status.UnmergedBothDeleted, file)
		case x == 'A' && y == 'U':
			status.UnmergedAddedByUs = append(status.UnmergedAddedByUs, file)
		case x == 'U' && y == 'D':
			status.UnmergedDeletedByThem = append(status.UnmergedDeletedByThem, file)
		case x == 'U' && y == 'A':
			status.UnmergedAddedByThem = append(status.UnmergedAddedByThem, file)
		case x == 'D' && y == 'U':
			status.UnmergedDeletedByUs = append(status.UnmergedDeletedByUs, file)
		case x == 'A' && y == 'A':
			status.UnmergedBothAdded = append(status.UnmergedBothAdded, file)
		case x == 'U' && y == 'U':
			status.UnmergedBothModified = append(status.UnmergedBothModified, file)
		case x == '?' && y == '?':
			status.Untracked = append(status.Untracked, file)
		case x == '!' && y == '!':
			status.Ignored = append(status.Ignored, file)
		default:
			return Status{}, fmt.Errorf("Unknown file status '%c%c'", x, y)
		}
	}
	return status, nil
}
