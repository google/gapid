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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type replaceResourceVerb struct{ ReplaceResourceFlags }

func init() {
	verb := &replaceResourceVerb{
		ReplaceResourceFlags{
			At:              -1,
			OutputTraceFile: "newcapture.gfxtrace",
		},
	}
	app.AddVerb(&app.Verb{
		Name:      "replace_resource",
		ShortHelp: "Produce a new trace with the given resource replaced at the given command",
		Action:    verb,
	})
}

func (verb *replaceResourceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	if (verb.Handle == "") == (verb.UpdateResourceBinary == "") {
		app.Usage(ctx, "only one of -handle or -updateresourcebinary arguments is required")
		return nil
	}

	if verb.Handle != "" && verb.ResourcePath == "" {
		app.Usage(ctx, "-resourcepath argument is required if -handle is specified")
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	boxedResources, err := client.Get(ctx, capture.Resources().Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Could not find the capture's resources")
	}
	resources := boxedResources.(*service.Resources)

	if verb.At == -1 {
		boxedCapture, err := client.Get(ctx, capture.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = int(boxedCapture.(*service.Capture).NumCommands) - 1
	}

	var resourcePath *path.Any
	var resourceData interface{}

	switch {
	case verb.Handle != "":
		matchedResource, err := resources.FindSingle(func(t path.ResourceType, r service.Resource) bool {
			return t == path.ResourceType_Shader &&
				(strings.Contains(r.GetHandle(), verb.Handle) || strings.Contains(r.GetID().ID().String(), verb.Handle))
		})
		if err != nil {
			return err
		}
		resourcePath = capture.Command(uint64(verb.At)).ResourceAfter(matchedResource.ID).Path()
		oldResourceData, err := client.Get(ctx, resourcePath, nil)
		if err != nil {
			log.Errf(ctx, err, "Could not get data for shader: %v", matchedResource)
			return err
		}
		shaderResourceData := oldResourceData.(*api.ResourceData).GetShader()
		newResourceBytes, err := ioutil.ReadFile(verb.ResourcePath)
		if err != nil {
			return log.Errf(ctx, err, "Could not read resource file %s", verb.ResourcePath)
		}
		resourceData = api.NewResourceData(&api.Shader{
			Type:   shaderResourceData.GetType(),
			Source: string(newResourceBytes),
		})
	case verb.UpdateResourceBinary != "":
		shaderResources := resources.FindAll(func(t path.ResourceType, r service.Resource) bool {
			return t == path.ResourceType_Shader
		})
		ids := make([]*path.ID, len(shaderResources))
		resourcesSource := make([]*api.ResourceData, len(shaderResources))
		for i, v := range shaderResources {
			ids[i] = v.ID
			resourcePath := capture.Command(uint64(verb.At)).ResourceAfter(v.ID)
			rd, err := client.Get(ctx, resourcePath.Path(), nil)
			if err != nil {
				log.Errf(ctx, err, "Could not get data for shader: %v", v)
				return err
			}
			newData, err := verb.getNewResourceData(ctx, rd.(*api.ResourceData).GetShader().GetSource())
			if err != nil {
				log.Errf(ctx, err, "Could not update the shader: %v", v)
				return err
			}
			resourcesSource[i] = api.NewResourceData(&api.Shader{
				Type:   rd.(*api.ResourceData).GetShader().GetType(),
				Source: string(newData),
			})
		}
		resourceData = api.NewMultiResourceData(resourcesSource)
		resourcePath = capture.Command(uint64(verb.At)).ResourcesAfter(ids).Path()
	}

	newResourcePath, err := client.Set(ctx, resourcePath, resourceData, nil)
	if err != nil {
		return log.Errf(ctx, err, "Could not update resource data: %v", resourcePath)
	}
	newCapture := path.FindCapture(newResourcePath.Node())
	log.I(ctx, "New capture id: %s", newCapture.ID)

	if verb.SkipOutput {
		log.I(ctx, "Skipped writing new capture to file.")
	} else {
		newCaptureFilepath, err := filepath.Abs(verb.OutputTraceFile)
		if err != nil {
			return log.Errf(ctx, err, "Could not handle capture file path '%s'", newCaptureFilepath)
		}
		err = client.SaveCapture(ctx, newCapture, newCaptureFilepath)
		if err != nil {
			return log.Errf(ctx, err, "Failed to write capture to: '%s'", newCaptureFilepath)
		}
		log.I(ctx, "Capture written to: '%s'", newCaptureFilepath)
	}

	return nil
}

// getNewResourceData runs the update resource binary on the old resource data
// and returns the newly generated resource data
func (verb *replaceResourceVerb) getNewResourceData(ctx context.Context, resourceData string) (string, error) {
	stdout, stderr := bytes.Buffer{}, bytes.Buffer{}
	cmd := shell.Cmd{Name: verb.UpdateResourceBinary}.
		Read(bytes.NewBufferString(resourceData)).
		Capture(&stdout, &stderr)

	if err := cmd.Run(ctx); err != nil {
		msg := fmt.Sprintf("Command '%v' returned error", verb.UpdateResourceBinary)
		if stderr.Len() > 0 {
			msg += fmt.Sprintf(": %v", stderr.String())
		}
		return "", log.Errf(ctx, err, msg)
	}
	return stdout.String(), nil
}
