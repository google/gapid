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

	"regexp"
	"strings"

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Find performs a search using req and calling handler for each result.
func Find(ctx context.Context, req *service.FindRequest, h service.FindHandler) error {
	var pred func(s string) bool
	if req.IsRegex {
		re, err := regexp.Compile(req.Text)
		if err != nil {
			return log.Err(ctx, err, "Couldn't compile regular expression")
		}
		pred = re.MatchString
	} else {
		pred = func(s string) bool { return strings.Contains(s, req.Text) }
	}

	switch from := protoutil.OneOf(req.From).(type) {
	case nil:
		return fault.Const("FindRequest.From cannot be nil")

	case *path.CommandTreeNode:
		boxedCmdTree, err := database.Resolve(ctx, from.Tree.ID())
		if err != nil {
			return err
		}

		cmdTree := boxedCmdTree.(*commandTree)

		c, err := capture.ResolveFromPath(ctx, cmdTree.path.Capture)
		if err != nil {
			return err
		}

		const stop = fault.Const("stop")
		count := uint32(0)
		cb := func(indices []uint64, atomIdx uint64, group *atom.Group) error {
			var text string
			if group != nil {
				text = group.Name
			} else {
				text = fmt.Sprint(c.Atoms[atomIdx])
			}
			if !pred(text) {
				return nil
			}
			err := h(&service.FindResponse{
				Result: &service.FindResponse_CommandTreeNode{
					CommandTreeNode: &path.CommandTreeNode{
						Tree:  from.Tree,
						Index: indices,
					},
				},
			})
			if err != nil {
				return err
			}
			count++
			if req.MaxItems != 0 && count > req.MaxItems {
				return stop
			}
			return task.StopReason(ctx)
		}
		err = cmdTree.root.Traverse(req.Backwards, from.Index, cb)
		switch err {
		case nil, stop:
			return nil
		default:
			return err
		}

	case *path.StateTreeNode:
		return fault.Const("TODO: Implement StateTreeNode searching") // TODO
	default:
		return fmt.Errorf("Unsupported FindRequest.From type %T", from)
	}
}
