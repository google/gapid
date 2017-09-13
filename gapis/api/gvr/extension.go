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

package gvr

import (
	"context"
	"reflect"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/extensions"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/cmdgrouper"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func init() {
	extensions.Register(extensions.Extension{
		Name:           "GVR",
		AdjustContexts: adjustContexts,
		CmdGroupers:    newReprojectionGroupers,
		Events:         newReprojectionEvents,
	})
}

type contextUsage int

const (
	rendererCtx = contextUsage(iota)
	reprojectionCtx
)

func adjustContexts(ctx context.Context, ctxs []*api.ContextInfo) {
	// Look for the renderer context.
	tyGvrFrameSubmit := reflect.TypeOf(&Gvr_frame_submit{})
	if c := findContextByCommand(ctxs, tyGvrFrameSubmit); c != nil {
		c.UserData[rendererCtx] = true
		c.Name = "Main context (" + c.Name + ")"
	}

	// Look for the reprojection context.
	tyGlFlush := reflect.TypeOf(&gles.GlFlush{})
	if c := findContextByCommand(ctxs, tyGlFlush); c != nil {
		c.UserData[reprojectionCtx] = true
		c.Name = "Reprojection context"
	}
}

func findContextByCommand(ctxs []*api.ContextInfo, ty reflect.Type) *api.ContextInfo {
	highest, best := 0, (*api.ContextInfo)(nil)
	for _, c := range ctxs {
		if count := c.NumCommandsByType[ty]; count > highest {
			highest, best = count, c
		}
	}
	return best
}

func isReprojectionContext(ctx context.Context, p *path.Context) bool {
	// Only group if we're looking at the reprojection thread.
	if p == nil {
		return false
	}
	c, err := resolve.Context(ctx, p)
	if c == nil || err != nil {
		return false
	}
	_, ok := c.UserData[reprojectionCtx]
	return ok
}

func newReprojectionGroupers(ctx context.Context, p *path.CommandTree) []cmdgrouper.Grouper {
	if !isReprojectionContext(ctx, p.Capture.Context(p.GetFilter().GetContext().ID())) {
		return nil
	}
	glFenceSync := func(cond gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlFenceSync)
			return ok && c.Condition == cond
		}
	}
	glEndTilingQCOM := func(cond gles.GLbitfield) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlEndTilingQCOM)
			return ok && c.PreserveMask == cond
		}
	}
	eglDestroySyncKHR := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.EglDestroySyncKHR)
			return ok
		}
	}
	glClientWaitSync := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlClientWaitSync)
			return ok
		}
	}
	glDeleteSync := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlDeleteSync)
			return ok
		}
	}
	glDrawElements := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlDrawElements)
			return ok
		}
	}
	notGlDrawElements := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlDrawElements)
			return !ok
		}
	}
	notGlFlush := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlFlush)
			return !ok
		}
	}
	return []cmdgrouper.Grouper{
		cmdgrouper.Sequence("Left eye",
			cmdgrouper.Rule{Pred: eglDestroySyncKHR()},
			cmdgrouper.Rule{Pred: glClientWaitSync()},
			cmdgrouper.Rule{Pred: glDeleteSync()},
			cmdgrouper.Rule{Pred: notGlDrawElements(), Repeats: true},
			cmdgrouper.Rule{Pred: glDrawElements()},
		),
		cmdgrouper.Sequence("Right eye",
			cmdgrouper.Rule{Pred: glFenceSync(gles.GLenum_GL_SYNC_GPU_COMMANDS_COMPLETE)},
			cmdgrouper.Rule{Pred: glEndTilingQCOM(1), Optional: true},
			cmdgrouper.Rule{Pred: glClientWaitSync()},
			cmdgrouper.Rule{Pred: glDeleteSync()},
			cmdgrouper.Rule{Pred: notGlDrawElements(), Repeats: true},
			cmdgrouper.Rule{Pred: glDrawElements()},
			cmdgrouper.Rule{Pred: notGlFlush(), Repeats: true},
		),
	}
}

func newReprojectionEvents(ctx context.Context, p *path.Events) extensions.EventProvider {
	if !isReprojectionContext(ctx, p.Capture.Context(p.GetFilter().GetContext().ID())) {
		return nil
	}

	var pending []service.EventKind
	return func(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState) []*service.Event {
		events := []*service.Event{}
		for _, kind := range pending {
			events = append(events, &service.Event{
				Kind:    kind,
				Command: p.Capture.Command(uint64(id)),
			})
		}
		pending = nil
		if _, ok := cmd.(*gles.GlFlush); ok {
			if p.LastInFrame {
				events = append(events, &service.Event{
					Kind:    service.EventKind_LastInFrame,
					Command: p.Capture.Command(uint64(id)),
				})
			}
			if p.FirstInFrame {
				pending = append(pending, service.EventKind_FirstInFrame)
			}
		}
		return events
	}
}
