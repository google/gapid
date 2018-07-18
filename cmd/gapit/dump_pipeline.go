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
	"path/filepath"
	"strconv"
	"strings"
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

	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	c, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture file")
	}

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, c.Path())
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	cmd := c.Command(verb.At[0], verb.At[1:]...)
	pipelineData, err := getBoundPipelineResource(ctx, client, cmd)
	if err != nil {
		return log.Err(ctx, err, "Failed to get bound pipeline resource data")
	}

	return verb.printPipelineData(ctx, client, pipelineData)
}

func getBoundPipelineResource(ctx context.Context, c client.Client, cmd *path.Command) (*api.Pipeline, error) {
	boxedResources, err := c.Get(ctx, (&path.Resources{Capture: cmd.Capture}).Path())
	if err != nil {
		return nil, err
	}

	resources := boxedResources.(*service.Resources)
	for _, typ := range resources.Types {
		if typ.Type != api.ResourceType_PipelineResource {
			continue
		}

		for _, resource := range typ.Resources {
			boxedResourceData, err := c.Get(ctx, cmd.ResourceAfter(resource.ID).Path())
			if err != nil {
				return nil, log.Err(ctx, err, "Failed to load the pipeline resource")
			}
			resourceData := boxedResourceData.(*api.ResourceData)
			pipelineData := protoutil.OneOf(protoutil.OneOf(resourceData)).(*api.Pipeline)
			if pipelineData.Bound {
				return pipelineData, nil
			}
		}
	}
	return nil, fmt.Errorf("No bound pipeline found")
}

func (verb *pipeVerb) printPipelineData(ctx context.Context, c client.Client, data *api.Pipeline) error {
	// Get the names for descriptor types and image layouts
	typeNames, err := getConstantSetMap(ctx, c, data.API, data.BindingTypeConstantsIndex)
	if err != nil {
		return err
	}
	layoutNames, err := getConstantSetMap(ctx, c, data.API, data.ImageLayoutConstantsIndex)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 4, 4, 0, ' ', 0)
	defer w.Flush()

	if verb.Print.Shaders {
		fmt.Fprintf(w, "%v shader stages:\n", len(data.Stages))
		for _, stage := range data.Stages {
			fmt.Fprintf(w, "\t%v stage:\n", stage.Type)
			fmt.Fprintf(w, "%v\n", stage.Shader.Source)
		}
	}
	fmt.Fprintf(w, "%v bindings:\n", len(data.Bindings))
	for _, binding := range data.Bindings {
		fmt.Fprintf(w, "Binding #%v.%v:\n", binding.Set, binding.Binding)
		typeName, ok := typeNames[binding.Type]
		if !ok {
			typeName = strconv.FormatUint(uint64(binding.Type), 10)
		}
		fmt.Fprintf(w, "\tType: \t%v\n", typeName)
		stageTypes := make([]api.StageType, len(binding.StageIdxs))
		for i, idx := range binding.StageIdxs {
			stageTypes[i] = data.Stages[idx].Type
		}
		if len(stageTypes) > 0 {
			fmt.Fprintf(w, "\tUsed by stages: \t%v\n", strings.Trim(fmt.Sprint(stageTypes), "[]"))
		} else {
			fmt.Fprintf(w, "\tUnused by pipeline\n")
		}
		if len(binding.Values) > 1 {
			fmt.Fprintf(w, "\tBound values:\n")
		}
		for i, val := range binding.Values {
			var valueType string
			switch protoutil.OneOf(val).(type) {
			case *api.BindingValue_Unbound:
				valueType = "Unbound"
			case *api.BindingValue_ImageInfo:
				valueType = "Image"
			case *api.BindingValue_BufferInfo:
				valueType = "Buffer"
			case *api.BindingValue_TexelBufferView:
				valueType = "Texel Buffer View"
			}
			if len(binding.Values) > 1 {
				fmt.Fprintf(w, "\t%v: %v\n", i, valueType)
			} else {
				fmt.Fprintf(w, "\tBound value: %v\n", valueType)
			}
			switch v := protoutil.OneOf(val).(type) {
			case *api.BindingValue_ImageInfo:
				fmt.Fprintf(w, "\t\tSampler: \t%v\n", v.ImageInfo.Sampler)
				fmt.Fprintf(w, "\t\tImage View: \t%v\n", v.ImageInfo.ImageView)
				layoutName, ok := layoutNames[v.ImageInfo.ImageLayout]
				if !ok {
					layoutName = strconv.FormatUint(uint64(v.ImageInfo.ImageLayout), 10)
				}
				fmt.Fprintf(w, "\t\tImage Layout: \t%v\n", layoutName)
			case *api.BindingValue_BufferInfo:
				fmt.Fprintf(w, "\t\tHandle: \t%v\n", v.BufferInfo.Buffer)
				fmt.Fprintf(w, "\t\tOffset: \t%v\n", v.BufferInfo.Offset)
				fmt.Fprintf(w, "\t\tRange: \t%v\n", v.BufferInfo.Range)
			case *api.BindingValue_TexelBufferView:
				fmt.Fprintf(w, "\t\tBuffer View: \t%v\n", v.TexelBufferView)
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
		}).Path())
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
