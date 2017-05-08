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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Events resolves and returns the event list from the path p.
func Events(ctx context.Context, p *path.Events) (*service.Events, error) {
	c, err := capture.ResolveFromPath(ctx, p.Commands.Capture)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, p.Commands.Capture, p.Filter)
	if err != nil {
		return nil, err
	}

	events := []*service.Event{}

	s := c.NewState()
	for i, a := range c.Atoms {
		a.Mutate(ctx, s, nil)
		// TODO: Add event generation to the API files.
		if !filter(a, s) {
			continue
		}
		f := a.AtomFlags()
		if p.DrawCalls && f.IsDrawCall() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_DrawCall,
				Command: p.Commands.Capture.Command(uint64(i)),
			})
		}
		if p.DrawCalls && f.IsUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_UserMarker,
				Command: p.Commands.Capture.Command(uint64(i)),
			})
		}
		if p.LastInFrame && f.IsEndOfFrame() && i > 0 {
			events = append(events, &service.Event{
				Kind:    service.EventKind_LastInFrame,
				Command: p.Commands.Capture.Command(uint64(i) - 1),
			})
		}
		if p.FirstInFrame && (f.IsEndOfFrame() || i == 0) {
			events = append(events, &service.Event{
				Kind:    service.EventKind_FirstInFrame,
				Command: p.Commands.Capture.Command(uint64(i)),
			})
		}
		if p.PushUserMarkers && f.IsPushUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_PushUserMarker,
				Command: p.Commands.Capture.Command(uint64(i)),
			})
		}
		if p.PopUserMarkers && f.IsPopUserMarker() {
			events = append(events, &service.Event{
				Kind:    service.EventKind_PopUserMarker,
				Command: p.Commands.Capture.Command(uint64(i)),
			})
		}

		if p.FramebufferObservations {
			if _, ok := a.(*atom.FramebufferObservation); ok {
				events = append(events, &service.Event{
					Kind:    service.EventKind_FramebufferObservation,
					Command: p.Commands.Capture.Command(uint64(i)),
				})
			}
		}
	}

	return &service.Events{List: events}, nil
}
