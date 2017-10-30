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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/extensions"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Events resolves and returns the event list from the path p.
func Events(ctx context.Context, p *path.Events) (*service.Events, error) {
	c, err := capture.ResolveFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	sd, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, p.Capture, p.Filter, sd)
	if err != nil {
		return nil, err
	}

	// Add any extension event filters
	filters := CommandFilters{filter}
	for _, e := range extensions.Get() {
		if f := e.EventFilter; f != nil {
			if filter := CommandFilter(f(ctx, p)); filter != nil {
				filters = append(filters, filter)
			}
		}
	}
	filter = filters.All

	// Add any extension events
	eps := []extensions.EventProvider{}
	for _, e := range extensions.Get() {
		if e.Events != nil {
			if ep := e.Events(ctx, p); ep != nil {
				eps = append(eps, ep)
			}
		}
	}

	events := []*service.Event{}

	s := c.NewState()
	lastCmd := api.CmdID(0)
	var pending []service.EventKind
	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, id, s, nil)

		// TODO: Add event generation to the API files.
		if !filter(id, cmd, s) {
			return nil
		}

		for _, kind := range pending {
			events = append(events, &service.Event{
				Kind:    kind,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		pending = nil

		f := cmd.CmdFlags(ctx, id, s)
		if p.Clears && f.IsClear() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_Clear,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.DrawCalls && f.IsDrawCall() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_DrawCall,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.TransformFeedbacks && f.IsTransformFeedback() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_TransformFeedback,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.UserMarkers && f.IsUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_UserMarker,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.LastInFrame && f.IsStartOfFrame() && lastCmd > 0 {
			events = append(events, &service.Event{
				Kind:    service.EventKind_LastInFrame,
				Command: p.Capture.Command(uint64(lastCmd)),
			})
		}
		if p.LastInFrame && f.IsEndOfFrame() && id > 0 {
			events = append(events, &service.Event{
				Kind:    service.EventKind_LastInFrame,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.FirstInFrame && (f.IsStartOfFrame() || id == 0) {
			events = append(events, &service.Event{
				Kind:    service.EventKind_FirstInFrame,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.FirstInFrame && f.IsEndOfFrame() {
			pending = append(pending, service.EventKind_FirstInFrame)
		}
		if p.PushUserMarkers && f.IsPushUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_PushUserMarker,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.PopUserMarkers && f.IsPopUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_PopUserMarker,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.AllCommands {
			events = append(events, &service.Event{
				Kind:    service.EventKind_AllCommands,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		if p.FramebufferObservations {
			// NOTE: gapit SxS video depends on FBO events coming after
			// all other event types.
			for _, e := range cmd.Extras().All() {
				if _, ok := e.(*capture.FramebufferObservation); ok {
					events = append(events, &service.Event{
						Kind:    service.EventKind_FramebufferObservation,
						Command: p.Capture.Command(uint64(id)),
					})
				}
			}
		}

		for _, ep := range eps {
			events = append(events, ep(ctx, id, cmd, s)...)
		}

		lastCmd = id
		return nil
	})

	return &service.Events{List: events}, nil
}
