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

package monitor

import (
	"context"

	"github.com/google/gapid/test/robot/subject"
)

// Subject is the in memory representation/wrapper for a subject.Subject
type Subject struct {
	subject.Subject
}

// Subjects is the type that manages a set of Subject objects.
type Subjects struct {
	entries []*Subject
}

// All returns the complete set of Subject objects we have seen so far.
func (s *Subjects) All() []*Subject {
	return s.entries
}

func (o *DataOwner) updateSubject(ctx context.Context, subj *subject.Subject) error {
	o.Write(func(data *Data) {
		for i, e := range data.Subjects.entries {
			if subj.Id == e.Id {
				data.Subjects.entries[i].Subject = *subj
			}
		}
		data.Subjects.entries = append(data.Subjects.entries, &Subject{Subject: *subj})
	})
	return nil
}
