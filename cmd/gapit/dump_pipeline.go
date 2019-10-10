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
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type pipeVerb struct{ PipelineFlags }

func init() {
	verb := &pipeVerb{}

	app.AddVerb(&app.Verb{
		Name:      "dump_pipeline",
		ShortHelp: "Prints the bound pipeline and descriptor sets for a point in a .gfxtrace file",
		Action:    verb,
	})
}

func (verb *pipeVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, c, err := getGapisAndLoadCapture(ctx, verb.Gapis, GapirFlags{}, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, c.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	cmd := c.Command(verb.At[0], verb.At[1:]...)
	pipelineData, err := verb.getBoundPipelineResource(ctx, client, cmd)
	if err != nil {
		return log.Err(ctx, err, "Failed to get bound pipeline resource data")
	}

	return verb.printPipelineData(ctx, client, pipelineData, "")
}

func (verb *pipeVerb) getBoundPipelineResource(ctx context.Context, c client.Client, cmd *path.Command) (*api.Pipeline, error) {
	boxedResources, err := c.Get(ctx, (&path.Resources{Capture: cmd.Capture}).Path(), nil)
	if err != nil {
		return nil, err
	}

	targetType := api.Pipeline_GRAPHICS
	if verb.Compute {
		targetType = api.Pipeline_COMPUTE
	}

	resources := boxedResources.(*service.Resources)
	for _, typ := range resources.Types {
		if typ.Type != api.ResourceType_PipelineResource {
			continue
		}

		for _, resource := range typ.Resources {
			boxedResourceData, err := c.Get(ctx, cmd.ResourceAfter(resource.ID).Path(), nil)
			if err != nil {
				return nil, log.Err(ctx, err, "Failed to load the pipeline resource")
			}
			resourceData := boxedResourceData.(*api.ResourceData)
			pipelineData := protoutil.OneOf(protoutil.OneOf(resourceData)).(*api.Pipeline)
			if pipelineData.Bound && pipelineData.PipelineType == targetType {
				return pipelineData, nil
			}
		}
	}
	return nil, fmt.Errorf("No bound %v pipeline found", targetType)
}

func (verb *pipeVerb) printPipelineData(ctx context.Context, c client.Client, data *api.Pipeline, prefix string) error {
	w := tabwriter.NewWriter(os.Stdout, 4, 4, 0, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s├── %s: \n", prefix, data.PipelineType)

	prefix = prefix + "│   "

	for i, stage := range data.Stages {
		if i == len(data.Stages)-1 {
			fmt.Fprintf(w, "%s└── %s: \n", prefix, stage.StageName)
		} else {
			fmt.Fprintf(w, "%s├── %s: \n", prefix, stage.StageName)
		}

		for j, group := range stage.Groups {
			if j == len(stage.Groups)-1 {
				fmt.Fprintf(w, "%s%s└── %s: \n", prefix, prefix, group.GroupName)
			} else {
				fmt.Fprintf(w, "%s%s├── %s: \n", prefix, prefix, group.GroupName)
			}
		}
	}

	return nil
}

func getConstantSetMap(ctx context.Context, c client.Client, api *path.API, index int32) (map[uint32]string, error) {
	names := map[uint32]string{}
	if index != -1 {
		boxedConstants, err := c.Get(ctx, (&path.ConstantSet{
			API:   api,
			Index: uint32(index),
		}).Path(), nil)
		if err != nil {
			return nil, log.Errf(ctx, err, "Failed to load constant set (%v, %v)", api, index)
		}
		constants := boxedConstants.(*service.ConstantSet)
		for _, c := range constants.Constants {
			names[uint32(c.Value)] = c.Name
		}
	}
	return names, nil
}
