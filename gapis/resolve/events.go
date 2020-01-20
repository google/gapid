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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/extensions"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Events resolves and returns the event list from the path p.
func Events(ctx context.Context, p *path.Events, r *path.ResolveConfig) (*service.Events, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	sd, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, p.Capture, p.Filter, sd, r)
	if err != nil {
		return nil, err
	}

	// Add any extension event filters
	filters := CommandFilters{filter}
	for _, e := range extensions.Get() {
		if f := e.EventFilter; f != nil {
			if filter := CommandFilter(f(ctx, p, r)); filter != nil {
				filters = append(filters, filter)
			}
		}
	}
	filter = filters.All

	// Add any extension events
	eps := []extensions.EventProvider{}
	for _, e := range extensions.Get() {
		if e.Events != nil {
			if ep := e.Events(ctx, p, r); ep != nil {
				eps = append(eps, ep)
			}
		}
	}

	events := []*service.Event{}

	s := c.NewState(ctx)
	lastCmd := api.CmdID(0)
	var pending []service.EventKind

	getTime := func(cmd api.Cmd) uint64 {
		if !p.IncludeTiming {
			return 0
		}
		for _, e := range cmd.Extras().All() {
			if t, ok := e.(*api.TimeStamp); ok {
				return t.Nanoseconds
			}
		}
		return 0
	}
	err = api.ForeachCmd(ctx, c.Commands, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}

		// TODO: Add event generation to the API files.
		if !filter(id, cmd, s) {
			return nil
		}

		// For a given command, if the command has a FirstInFrame
		// event, we want that to come before all other events for that
		// command, and likewise for a LastInFrame event, it should be
		// after all other events for that command (except
		// FramebufferObservation as described below).
		// Since extension.EventProvider's can provide any type of
		// event, we split them up into FirstInFrame, LastInFrame, and
		// rest, and then insert them into the event list in the right
		// spot to ensure the ordering is maintained.
		var epFirstInFrame, epNormal, epLastInFrame []*service.Event
		for _, ep := range eps {
			for _, event := range ep(ctx, id, cmd, s) {
				switch event.Kind {
				case service.EventKind_FirstInFrame:
					epFirstInFrame = append(epFirstInFrame, event)
				case service.EventKind_LastInFrame:
					epLastInFrame = append(epFirstInFrame, event)
				default:
					epNormal = append(epNormal, event)
				}
			}
		}

		f := cmd.CmdFlags(ctx, id, s)

		// Add LastInFrame event of a previous command first.
		if p.LastInFrame && f.IsStartOfFrame() && lastCmd > 0 {
			events = append(events, &service.Event{
				Kind:      service.EventKind_LastInFrame,
				Command:   p.Capture.Command(uint64(lastCmd)),
				Timestamp: getTime(c.Commands[lastCmd]),
			})
		}

		// Add any pending events (currently only FirstInFrame events).
		for _, kind := range pending {
			events = append(events, &service.Event{
				Kind:      kind,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		pending = nil

		// Add all first in frame events
		if p.FirstInFrame && (f.IsStartOfFrame() || id == 0) {
			events = append(events, &service.Event{
				Kind:      service.EventKind_FirstInFrame,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.FirstInFrame {
			events = append(events, epFirstInFrame...)
		}
		if p.FirstInFrame && f.IsEndOfFrame() {
			pending = append(pending, service.EventKind_FirstInFrame)
		}
		if p.Clears && f.IsClear() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_Clear,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		// Add all non-special event types
		events = append(events, epNormal...)
		if p.DrawCalls && f.IsDrawCall() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_DrawCall,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.Submissions && f.IsSubmission() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_Submission,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.TransformFeedbacks && f.IsTransformFeedback() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_TransformFeedback,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.UserMarkers && f.IsUserMarker() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_UserMarker,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.PushUserMarkers && f.IsPushUserMarker() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_PushUserMarker,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.PopUserMarkers && f.IsPopUserMarker() {
			events = append(events, &service.Event{
				Kind:      service.EventKind_PopUserMarker,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.AllCommands {
			events = append(events, &service.Event{
				Kind:      service.EventKind_AllCommands,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		// Add LastInFrame events after other events for a given command
		if p.LastInFrame && f.IsEndOfFrame() && id > 0 {
			events = append(events, &service.Event{
				Kind:      service.EventKind_LastInFrame,
				Command:   p.Capture.Command(uint64(id)),
				Timestamp: getTime(cmd),
			})
		}
		if p.LastInFrame {
			events = append(events, epLastInFrame...)
		}
		if p.FramebufferObservations {
			// NOTE: gapit SxS video depends on FBO events coming after
			// all other event types.
			for _, e := range cmd.Extras().All() {
				if _, ok := e.(*capture.FramebufferObservation); ok {
					events = append(events, &service.Event{
						Kind:      service.EventKind_FramebufferObservation,
						Command:   p.Capture.Command(uint64(id)),
						Timestamp: getTime(cmd),
					})
				}
			}
		}

		lastCmd = id
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &service.Events{List: events}, nil
}
