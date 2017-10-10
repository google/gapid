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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// FramebufferChanges returns the list of attachment changes over the span of
// the entire capture.
func FramebufferChanges(ctx context.Context, c *path.Capture) (*AttachmentFramebufferChanges, error) {
	obj, err := database.Build(ctx, &FramebufferChangesResolvable{c})
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

const errNoAPI = fault.Const("Command has no API")

// Resolve implements the database.Resolver interface.
func (r *FramebufferChangesResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	out := &AttachmentFramebufferChanges{
		// TODO: Remove hardcoded upper limit
		attachments: make([]framebufferAttachmentChanges, api.FramebufferAttachment_Color3+1),
	}

	postCmdAndSubCmd := func(s *api.GlobalState, subcommandIndex api.SubCmdIdx, cmd api.Cmd) {
		api := cmd.API()
		idx := append([]uint64(nil), subcommandIndex...)
		for _, att := range allFramebufferAttachments {
			info := framebufferAttachmentInfo{after: idx}
			if api != nil {
				if w, h, i, f, err := api.GetFramebufferAttachmentInfo(ctx, idx, s, cmd.Thread(), att); err == nil && f != nil {
					info.width, info.height, info.index, info.format = w, h, i, f
				} else {
					info.err = err
				}
			} else {
				info.err = errNoAPI
			}
			if last := out.attachments[att].last(); !last.equal(info) {
				attachment := out.attachments[att]
				attachment.changes = append(attachment.changes, info)
				out.attachments[att] = attachment
			}
		}
	}

	sync.MutateWithSubcommands(ctx, r.Capture, c.Commands, postCmdAndSubCmd, nil, postCmdAndSubCmd)
	return out, nil
}
