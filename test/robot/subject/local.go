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

package subject

import (
	"context"
	"reflect"
	"sync"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
	"github.com/google/gapid/test/robot/stash"
)

type local struct {
	mu       sync.Mutex
	store    *stash.Client
	ledger   record.Ledger
	subjects []*Subject
	byID     map[string]*Subject
	onChange event.Broadcast
}

// NewLocal returns a new persistent store of Subjects.
func NewLocal(ctx context.Context, library record.Library, store *stash.Client) (Subjects, error) {
	ledger, err := library.Open(ctx, "subjects", &Subject{})
	if err != nil {
		return nil, err
	}
	s := &local{
		store:    store,
		ledger:   ledger,
		subjects: []*Subject{},
		byID:     map[string]*Subject{},
	}
	apply := event.AsHandler(ctx, s.apply)
	if err := s.ledger.Read(ctx, apply); err != nil {
		return nil, err
	}
	s.ledger.Watch(ctx, apply)
	return s, nil
}

// apply is called with items coming out of the ledger
// it should be called with the mutation lock already held.
func (s *local) apply(ctx context.Context, subject *Subject) error {
	old := s.byID[subject.Id]
	if old == nil {
		s.subjects = append(s.subjects, subject)
		s.byID[subject.Id] = subject
		s.onChange.Send(ctx, subject)
		return nil
	}
	if subject.Hints != nil {
		if old.Hints.TraceTime != subject.Hints.TraceTime {
			old.Hints.TraceTime = subject.Hints.TraceTime
		}
		if old.Hints.API != subject.Hints.API {
			old.Hints.API = subject.Hints.API
		}
	}
	s.onChange.Send(ctx, old)
	return nil
}

// Search implements Subjects.Search
// It searches the set of persisted subjects, and supports monitoring of subjects as they arrive.
func (s *local) Search(ctx context.Context, query *search.Query, handler Handler) error {
	filter := eval.Filter(ctx, query, reflect.TypeOf(&Subject{}), event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, s.subjects)
	if query.Monitor {
		return event.Monitor(ctx, &s.mu, s.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

// Add implements Subjects.Add
func (s *local) Add(ctx context.Context, id string, hints *Hints) (*Subject, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if subject, ok := s.byID[id]; ok {
		return subject, false, nil
	}
	data, err := s.store.Read(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	// TODO: support non apk subjects
	info, err := apk.Analyze(ctx, data)
	if err != nil {
		return nil, false, err
	}
	subject := &Subject{
		Id: id,
		Information: &Subject_APK{
			APK: info,
		},
		Hints: hints,
	}
	if err := s.ledger.Add(ctx, subject); err != nil {
		return nil, false, err
	}
	return subject, true, nil
}

func (s *local) Update(ctx context.Context, subj *Subject) (*Subject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byID[subj.Id]; !found {
		return nil, log.Err(ctx, nil, "Subject not found")
	}
	if err := s.ledger.Add(ctx, subj); err != nil {
		return nil, err
	}
	return s.byID[subj.Id], nil
}
