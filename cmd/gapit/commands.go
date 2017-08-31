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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type commandsVerb struct{ CommandsFlags }

func init() {
	verb := &commandsVerb{
		CommandsFlags: CommandsFlags{
			CommandFilterFlags: CommandFilterFlags{
				Context: -1,
			},
		},
	}
	verb.Context = -1
	app.AddVerb(&app.Verb{
		Name:      "commands",
		ShortHelp: "Prints the command tree for a .gfxtrace file",
		Action:    verb,
	})
}

func (verb *commandsVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	c, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture file")
	}

	filter, err := verb.commandFilter(ctx, client, c)
	if err != nil {
		return log.Err(ctx, err, "Failed to build the CommandFilter")
	}

	treePath := c.CommandTree(filter)
	treePath.GroupByApi = verb.GroupByAPI
	treePath.GroupByContext = verb.GroupByContext
	treePath.GroupByThread = verb.GroupByThread
	treePath.GroupByDrawCall = verb.GroupByDrawCall
	treePath.GroupByFrame = verb.GroupByFrame
	treePath.GroupByUserMarkers = verb.GroupByUserMarkers
	treePath.IncludeNoContextGroups = verb.IncludeNoContextGroups
	treePath.AllowIncompleteFrame = verb.AllowIncompleteFrame

	treePath.MaxChildren = int32(verb.MaxChildren)

	boxedTree, err := client.Get(ctx, treePath.Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.CommandTree)

	if verb.Name != "" {
		req := &service.FindRequest{
			From:    &service.FindRequest_CommandTreeNode{CommandTreeNode: tree.Root},
			Text:    verb.Name,
			IsRegex: false, // TODO: Flag for this?
		}
		client.Find(ctx, req, func(r *service.FindResponse) error {
			p := r.GetCommandTreeNode()
			boxedNode, err := client.Get(ctx, p.Path())
			if err != nil {
				return err
			}
			n := boxedNode.(*service.CommandTreeNode)

			if n.Group != "" {
				fmt.Fprintln(os.Stdout, n.Group)
				return nil
			}
			return getAndPrintCommand(ctx, client, n.Commands.First(), verb.Observations)
		})
		return nil
	}

	return traverseCommandTree(ctx, client, tree.Root, func(n *service.CommandTreeNode, prefix string) error {
		fmt.Fprintf(os.Stdout, prefix)
		if n.Group != "" {
			fmt.Fprintln(os.Stdout, n.Group)
			return nil
		}
		return getAndPrintCommand(ctx, client, n.Commands.First(), verb.Observations)
	}, "", true)
}

func traverseCommandTree(
	ctx context.Context,
	c client.Client,
	p *path.CommandTreeNode,
	f func(n *service.CommandTreeNode, prefix string) error,
	prefix string,
	last bool) error {

	if task.Stopped(ctx) {
		return task.StopReason(ctx)
	}

	boxedNode, err := c.Get(ctx, p.Path())
	if err != nil {
		return log.Errf(ctx, err, "Failed to load the node at: %v", p)
	}

	n := boxedNode.(*service.CommandTreeNode)

	curPrefix := prefix
	if len(p.Indices) > 0 {
		if last {
			curPrefix += "└──"
		} else {
			curPrefix += "├──"
		}
	}

	if err := f(n, curPrefix); err != nil {
		return err
	}

	if last {
		prefix += "    "
	} else {
		prefix += "│   "
	}
	for i := uint64(0); i < n.NumChildren; i++ {
		err := traverseCommandTree(ctx, c, p.Child(i), f, prefix, i == n.NumChildren-1)
		if err != nil {
			return err
		}
	}

	return nil
}
