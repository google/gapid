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

package build

import (
	"context"
	"reflect"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

var trackClass = reflect.TypeOf(&Track{})

type tracks struct {
	mu       sync.Mutex
	ledger   record.Ledger
	entries  []*Track
	byID     map[string]*Track
	byName   map[string]*Track
	onChange event.Broadcast
}

func (t *tracks) init(ctx context.Context, library record.Library) error {
	ledger, err := library.Open(ctx, "tracks", &Track{})
	if err != nil {
		return err
	}
	t.ledger = ledger
	t.byID = map[string]*Track{}
	t.byName = map[string]*Track{}
	apply := event.AsHandler(ctx, t.apply)
	if err := ledger.Read(ctx, apply); err != nil {
		return err
	}
	ledger.Watch(ctx, apply)
	return nil
}

// apply is called with items coming out of the ledger
// it should be called with the mutation lock already held.
func (t *tracks) apply(ctx context.Context, track *Track) error {
	old := t.byID[track.Id]
	if old == nil {
		t.entries = append(t.entries, track)
		t.byID[track.Id] = track
		if track.Name != "" {
			t.byName[track.Name] = track
		}
		t.onChange.Send(ctx, track)
		return nil
	}
	if track.Name != "" && old.Name != track.Name {
		delete(t.byName, old.Name)
		old.Name = track.Name
		t.byName[old.Name] = old
	}
	if track.Head != "" && old.Head != track.Head {
		old.Head = track.Head
	}
	if track.Description != "" {
		old.Description = track.Description
	}
	t.onChange.Send(ctx, track)
	return nil
}

func (t *tracks) search(ctx context.Context, query *search.Query, handler TrackHandler) error {
	filter := eval.Filter(ctx, query, trackClass, event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, t.entries)
	if query.Monitor {
		return event.Monitor(ctx, &t.mu, t.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

func (t *tracks) createOrUpdate(ctx context.Context, track *Track) (*Track, string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	parent := ""
	if track.Id != "" {
		// We must be updating an exiting track, and the id must be valid
		old, found := t.byID[track.Id]
		if !found {
			return nil, "", log.Err(ctx, nil, "Track not found")
		}
		parent = old.Head
	} else if track.Name != "" {
		if old, found := t.byName[track.Name]; found {
			// Matched by name, update the existing entry
			track.Id = old.Id
			parent = old.Head
		}
	}
	if track.Id == "" {
		// We are making a new track, pick a new unique id
		track.Id = id.Unique().String()
	}
	if err := t.ledger.Add(ctx, track); err != nil {
		return nil, "", err
	}
	return t.byID[track.Id], parent, nil
}

func guessTrackName(ctx context.Context, info *Information) string {
	branch := info.Branch
	if info.Branch == "" {
		// Branch is the primary information for track name, so we always pick one
		branch = "auto"
	}
	if info.Type == BuildBot {
		return branch
	}
	// For non build bot packages, we prepend a user name
	user := info.Uploader
	if user == "" {
		user = "unknown"
	}
	return user + "-" + branch
}

func (t *tracks) addPackage(ctx context.Context, pkg *Package) (string, error) {
	parent := ""
	track := &Track{Name: guessTrackName(ctx, pkg.Information), Head: pkg.Id}
	_, parent, err := t.createOrUpdate(ctx, track)
	return parent, err
}
