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
		EventFilter:    eventFilter,
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
	tyGlFlush := reflect.TypeOf(&gles.GlFlush{})

	renderer := findContextByCommand(ctxs, tyGvrFrameSubmit)
	reprojection := findContextByCommand(ctxs, tyGlFlush)

	if renderer != nil && reprojection != nil {
		renderer.UserData[rendererCtx] = true
		renderer.Name = "Main context (" + renderer.Name + ")"
		reprojection.UserData[reprojectionCtx] = true
		reprojection.Name = "Reprojection context (" + reprojection.Name + ")"
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

func getRenderContextID(ctx context.Context, p *path.Contexts) *api.ContextID {
	if p == nil {
		return nil
	}
	ctxs, err := resolve.Contexts(ctx, p)
	if err != nil {
		return nil
	}
	for _, c := range ctxs {
		if _, ok := c.UserData[rendererCtx]; ok {
			return &c.ID
		}
	}
	return nil
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
		noSubFrameEventGrouper{cmdgrouper.Sequence("Left eye",
			cmdgrouper.Rule{Pred: eglDestroySyncKHR()},
			cmdgrouper.Rule{Pred: glClientWaitSync()},
			cmdgrouper.Rule{Pred: glDeleteSync()},
			cmdgrouper.Rule{Pred: notGlDrawElements(), Repeats: true},
			cmdgrouper.Rule{Pred: glDrawElements()},
		)},
		noSubFrameEventGrouper{cmdgrouper.Sequence("Right eye",
			cmdgrouper.Rule{Pred: glFenceSync(gles.GLenum_GL_SYNC_GPU_COMMANDS_COMPLETE)},
			cmdgrouper.Rule{Pred: glEndTilingQCOM(1), Optional: true},
			cmdgrouper.Rule{Pred: glClientWaitSync()},
			cmdgrouper.Rule{Pred: glDeleteSync()},
			cmdgrouper.Rule{Pred: notGlDrawElements(), Repeats: true},
			cmdgrouper.Rule{Pred: glDrawElements()},
			cmdgrouper.Rule{Pred: notGlFlush(), Repeats: true},
		)},
	}
}

type noSubFrameEventGrouper struct {
	cmdgrouper.Grouper
}

func (n noSubFrameEventGrouper) Build(end api.CmdID) []cmdgrouper.Group {
	out := n.Grouper.Build(end)
	for i := range out {
		out[i].UserData = &resolve.CmdGroupData{
			Representation:     api.CmdNoID,
			NoFrameEventGroups: true,
		}
	}
	return out
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

func eventFilter(ctx context.Context, p *path.Events) extensions.EventFilter {
	renderCtxID := getRenderContextID(ctx, p.Capture.Contexts())
	if renderCtxID == nil {
		return nil
	}
	return func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
		_, isSwapBuffers := cmd.(*gles.EglSwapBuffers)
		if isSwapBuffers {
			if context := gles.GetContext(s, cmd.Thread()); context != nil {
				ctxID := context.ID()
				if ctxID == *renderCtxID {
					// Strip out eglSwapBuffers from the render context.
					// We don't want them appearing as frames.
					return false
				}
			}
		}
		return true
	}
}
