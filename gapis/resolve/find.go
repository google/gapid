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
	"reflect"
	"regexp"
	"strings"

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

const (
	stop = fault.Const("stop")
)

func translateIDForDisplay(idx api.SubCmdIdx, data *sync.Data) (api.CmdID, bool) {
	atomIdx := api.CmdID(0)
	sg, ok := data.SubcommandReferences[api.CmdID(idx[0])]
	if !ok {
		return 0, false
	}
	found := false
	for _, v := range sg {
		if v.Index.Equals(idx) {
			found = true
			atomIdx = v.GeneratingCmd
			break
		}
	}
	return atomIdx, found
}

// Find performs a search using req and calling handler for each result.
func Find(ctx context.Context, req *service.FindRequest, h service.FindHandler) error {
	var pred func(s string) bool
	text := req.Text
	if !req.IsCaseSensitive {
		text = strings.ToLower(text)
	}
	if req.IsRegex {
		re, err := regexp.Compile(text)
		if err != nil {
			return log.Err(ctx, err, "Couldn't compile regular expression")
		}
		if req.IsCaseSensitive {
			pred = re.MatchString
		} else {
			pred = func(s string) bool { return re.MatchString(strings.ToLower(s)) }
		}
	} else if req.IsCaseSensitive {
		pred = func(s string) bool { return strings.Contains(s, text) }
	} else {
		pred = func(s string) bool { return strings.Contains(strings.ToLower(s), text) }
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

		snc, err := SyncData(ctx, cmdTree.path.Capture)
		if err != nil {
			return err
		}

		nodePred := func(item api.SpanItem) bool {
			switch item := item.(type) {
			case api.CmdIDGroup:
				return pred(item.Name)
			case api.SubCmdIdx:
				if len(item) > 1 {
					if idx, found := translateIDForDisplay(item, snc); found {
						return pred(fmt.Sprint(c.Commands[idx]))
					}
					return false
				}
				return pred(fmt.Sprint(c.Commands[item[0]]))
			case api.SubCmdRoot:
				if len(item.Id) > 1 {
					if idx, found := translateIDForDisplay(item.Id, snc); found {
						return pred(fmt.Sprint(c.Commands[idx]))
					}
					return false
				}
				return pred(fmt.Sprint(c.Commands[item.Id[0]]))
			default:
				return false
			}
		}

		emitter := &commandEmitter{ctx, req, from, h, 0, nodePred, false, true}
		err = cmdTree.root.Traverse(req.Backwards, from.Indices, emitter.process)
		if err == nil && req.Wrap && len(from.Indices) > 0 {
			var start []uint64
			if req.Backwards {
				start = []uint64{cmdTree.root.Count() - 1}
			}
			emitter.wrapping = true
			err = cmdTree.root.Traverse(req.Backwards, start, emitter.process)
		}

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

type commandEmitter struct {
	ctx      context.Context
	req      *service.FindRequest
	from     *path.CommandTreeNode
	h        service.FindHandler
	count    uint32
	pred     func(item api.SpanItem) bool
	wrapping bool
	first    bool
}

func (c *commandEmitter) process(indices []uint64, item api.SpanItem) error {
	// Skip the first item if we're not doing the wrapped search.
	if !c.wrapping && c.first {
		if reflect.DeepEqual(c.from.Indices, indices) {
			return task.StopReason(c.ctx)
		}
		c.first = false
	}

	if c.pred(item) {
		if err := c.emit(indices); err != nil {
			return err
		}
	}

	if c.shouldStop(indices) {
		return stop
	}
	return task.StopReason(c.ctx)
}

func (c *commandEmitter) emit(indices []uint64) error {
	err := c.h(&service.FindResponse{
		Result: &service.FindResponse_CommandTreeNode{
			CommandTreeNode: &path.CommandTreeNode{
				Tree:    c.from.Tree,
				Indices: indices,
			},
		},
	})
	if err != nil {
		return err
	}
	c.count++
	if c.req.MaxItems != 0 && c.count >= c.req.MaxItems {
		return stop
	}
	return nil
}

func (c *commandEmitter) shouldStop(indices []uint64) bool {
	// Stop searching if we're wrapping and have arrived back where we started.
	return c.wrapping && reflect.DeepEqual(c.from.Indices, indices)
}
