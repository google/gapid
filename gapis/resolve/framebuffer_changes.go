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

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// FramebufferChanges returns the list of attachment changes over the span of
// the entire capture.
func FramebufferChanges(ctx context.Context, c *path.Capture, r *path.ResolveConfig) (*AttachmentFramebufferChanges, error) {
	obj, err := database.Build(ctx, &FramebufferChangesResolvable{Capture: c, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*AttachmentFramebufferChanges), nil
}

// AttachmentFramebufferChanges describes the list of attachment changes over
// the span of the entire capture.
type AttachmentFramebufferChanges struct {
	attachments []framebufferAttachmentChanges
}

// Get returns the framebuffer dimensions and format after a given command in
// the given capture, command and attachment.
func (c AttachmentFramebufferChanges) Get(ctx context.Context, after *path.Command, att uint32) (FramebufferAttachmentInfo, error) {
	info, err := c.attachments[att].after(ctx, api.SubCmdIdx(after.Indices))
	if err != nil {
		return FramebufferAttachmentInfo{}, err
	}
	if info.Err != nil {
		log.W(ctx, "Framebuffer error after %v: %v", after, info.Err)
		return FramebufferAttachmentInfo{}, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}
	return info, nil
}

const errNoAPI = fault.Const("Command has no API")
const errDetached = fault.Const("Attachment detached from Framebuffer")

// Resolve implements the database.Resolver interface.
func (r *FramebufferChangesResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, r.Capture, r.Config)

	c, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	out := &AttachmentFramebufferChanges{
		// TODO: Remove hardcoded upper limit
		attachments: make([]framebufferAttachmentChanges, 0),
	}

	postCmd := func(s *api.GlobalState, subcommandIndex api.SubCmdIdx, cmd api.Cmd) {
		api := cmd.API()
		fbaInfos, _ := api.GetFramebufferAttachmentInfos(ctx, s)
		idx := append([]uint64(nil), subcommandIndex...)
		for i, inf := range fbaInfos {
			info := FramebufferAttachmentInfo{After: idx}
			if api != nil {
				if inf.Err == nil {
					info.Width, info.Height, info.Index, info.Format, info.CanResize, info.Type = inf.Width, inf.Height, inf.Index, inf.Format, inf.CanResize, inf.Type
				} else {
					info.Err = inf.Err
				}
			} else {
				info.Err = errNoAPI
			}
			if len(out.attachments) == i {
				out.attachments = append(out.attachments, framebufferAttachmentChanges{changes: make([]FramebufferAttachmentInfo, 0)})
			}
			if last := out.attachments[i].last(); !last.equal(info) {
				attachment := out.attachments[i]
				attachment.changes = append(attachment.changes, info)
				out.attachments[i] = attachment
			}
		}

		// Current command may be in a renderpass with fewer attachments.
		// Go ahead and fill up the spaces with info.Err for future filtering
		for i := len(fbaInfos); i < len(out.attachments); i++ {
			info := FramebufferAttachmentInfo{After: idx}
			info.Err = errDetached
			if last := out.attachments[i].last(); !last.equal(info) {
				attachment := out.attachments[i]
				attachment.changes = append(attachment.changes, info)
				out.attachments[i] = attachment
			}
		}
	}

	postSubCmd := func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{}) {
		postCmd(s, idx, cmd)
	}

	sync.MutateWithSubcommands(ctx, r.Capture, c.Commands, postCmd, nil, postSubCmd)
	return out, nil
}
