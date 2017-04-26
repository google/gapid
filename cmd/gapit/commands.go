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
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type commandsVerb struct{ CommandsFlags }

func init() {
	verb := &commandsVerb{}
	app.AddVerb(&app.Verb{
		Name:      "commands",
		ShortHelp: "Prints the command tree for a .gfxtrace file",
		Auto:      verb,
	})
}

func (verb *commandsVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
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

	boxedTree, err := client.Get(ctx, c.CommandTree(nil, nil).Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.CommandTree)

	return traverseCommandTree(ctx, client, tree.Root, func(p *path.CommandTreeNode, n *service.CommandTreeNode) error {
		if len(p.Index) > 0 {
			fmt.Fprintf(os.Stdout, strings.Repeat("│   ", len(p.Index)-1)+"├── ")
		}
		if n.Group != "" {
			fmt.Fprintln(os.Stdout, n.Group)
			return nil
		}
		return getAndPrintCommand(ctx, client, n.Command)
	})
}

func traverseCommandTree(
	ctx context.Context,
	c client.Client,
	p *path.CommandTreeNode,
	f func(p *path.CommandTreeNode, n *service.CommandTreeNode) error) error {

	if task.Stopped(ctx) {
		return task.StopReason(ctx)
	}

	boxedNode, err := c.Get(ctx, p.Path())
	if err != nil {
		return log.Errf(ctx, err, "Failed to load the node at: %v", p.Text())
	}

	n := boxedNode.(*service.CommandTreeNode)

	if err := f(p, n); err != nil {
		return err
	}

	for i := uint64(0); i < n.NumChildren; i++ {
		if err := traverseCommandTree(ctx, c, p.Child(i), f); err != nil {
			return err
		}
	}

	return nil
}
