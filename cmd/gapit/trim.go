// Copyright (C) 2018 Google Inc.
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
	"io/ioutil"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type trimVerb struct{ TrimFlags }

func init() {
	verb := &trimVerb{}
	verb.Frames.Count = allTheWay

	app.AddVerb(&app.Verb{
		Name:      "trim",
		ShortHelp: "(WIP) Trims a gfx trace to the dependencies of the requested frames",
		Action:    verb,
	})
}

func (verb *trimVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	eofCommands, err := verb.eofCommands(ctx, capture, client)
	if err != nil {
		return err
	}

	dceRequest := verb.getDCERequest(eofCommands, capture)
	if len(dceRequest) > 0 {
		capture, err = client.DCECapture(ctx, capture, dceRequest)
		if err != nil {
			return log.Errf(ctx, err, "DCECapture(%v, %v)", capture, dceRequest)
		}
	}

	data, err := client.ExportCapture(ctx, capture)
	if err != nil {
		return log.Errf(ctx, err, "ExportCapture(%v)", capture)
	}

	output := verb.Out
	if output == "" {
		output = "trimmed.gfxtrace"
	}
	if err := ioutil.WriteFile(output, data, 0666); err != nil {
		return log.Errf(ctx, err, "Writing file: %v", output)
	}
	return nil
}

func (verb *trimVerb) eofCommands(ctx context.Context, capture *path.Capture, client client.Client) ([]*path.Command, error) {
	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}
	filter.OnlyEndOfFrames = true

	treePath := capture.CommandTree(filter)

	boxedTree, err := client.Get(ctx, treePath.Path(), nil)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.CommandTree)

	var eofCommands []*path.Command
	traverseCommandTree(ctx, client, tree.Root, func(n *service.CommandTreeNode, prefix string) error {
		if n.Group != "" {
			return nil
		}
		eofCommands = append(eofCommands, n.Commands.First())
		return nil
	}, "", true)

	lastFrame := len(eofCommands) - 1
	if verb.Frames.Count > 0 {
		lastFrame += verb.Frames.Count - 1
	}
	if lastFrame >= len(eofCommands) {
		return nil, log.Errf(ctx, nil, "Requested frame %d, but capture only contains %d frames", lastFrame, len(eofCommands))
	}

	return eofCommands, nil
}

func (verb *trimVerb) getDCERequest(eofCommands []*path.Command, p *path.Capture) []*path.Command {
	frameCount := verb.Frames.Count
	if frameCount < 0 {
		frameCount = len(eofCommands) - verb.Frames.Start
	}
	dceRequest := make([]*path.Command, 0, frameCount+len(verb.ExtraCommands))
	for i := 0; i < frameCount; i++ {
		indices := eofCommands[verb.Frames.Start+i].Indices
		newIndices := make([]uint64, len(indices))
		copy(newIndices, indices)
		cmd := &path.Command{
			Capture: p,
			Indices: newIndices,
		}
		dceRequest = append(dceRequest, cmd)
	}
	for _, id := range verb.ExtraCommands {
		cmd := &path.Command{
			Capture: p,
			Indices: []uint64{id},
		}
		dceRequest = append(dceRequest, cmd)
	}
	return dceRequest
}
