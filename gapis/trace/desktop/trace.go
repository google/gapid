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

package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/vulkan/loader"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/trace/tracer"
)

// Only update the package list every 30 seconds at most
var packageUpdateTime = 30.0

type DesktopTracer struct {
	b bind.Device
}

func NewTracer(dev bind.Device) *DesktopTracer {
	return &DesktopTracer{dev.(bind.Device)}
}

// IsServerLocal returns true if all paths on this device can be server-local
func (t *DesktopTracer) IsServerLocal() bool {
	return true
}

func (t *DesktopTracer) CanSpecifyCWD() bool {
	return true
}

func (t *DesktopTracer) CanSpecifyEnv() bool {
	return true
}

func (t *DesktopTracer) CanUploadApplication() bool {
	return false
}

func (t *DesktopTracer) HasCache() bool {
	return false
}

func (t *DesktopTracer) CanUsePortFile() bool {
	return true
}

func (t *DesktopTracer) APITraceOptions(ctx context.Context) []tracer.APITraceOptions {
	options := make([]tracer.APITraceOptions, 0, 1)
	if len(t.b.Instance().Configuration.Drivers.Vulkan.PhysicalDevices) > 0 {
		options = append(options, tracer.VulkanTraceOptions())
	}
	return options
}

func (t *DesktopTracer) FindTraceTarget(ctx context.Context, str string) (*tracer.TraceTargetTreeNode, error) {
	isFile, err := t.b.IsFile(ctx, str)
	if err != nil {
		return nil, err
	}
	if !isFile {
		return nil, fmt.Errorf("Trace target is not an executable file %+v", str)
	}
	dir, file := filepath.Split(str)

	if dir == "" {
		dir = "."
		if runtime.GOOS == "windows" {
			str = ".\\" + file
		} else {
			str = "./" + file
		}
	}

	tttn := &tracer.TraceTargetTreeNode{
		Name:            file,
		Icon:            nil,
		URI:             str,
		TraceURI:        str,
		Children:        nil,
		Parent:          dir,
		ApplicationName: "",
		ExecutableName:  file,
	}

	return tttn, nil
}

func (t *DesktopTracer) GetTraceTargetNode(ctx context.Context, uri string, iconDensity float32) (*tracer.TraceTargetTreeNode, error) {
	dirs := []string{}
	files := []string{}
	var err error

	traceUri := ""
	if uri == "" {
		uri = t.b.GetURIRoot()
	}

	isFile, err := t.b.IsFile(ctx, uri)
	if err != nil {
		return nil, err
	}
	if !isFile {
		dirs, err = t.b.ListDirectories(ctx, uri)
		if err != nil {
			return nil, err
		}

		files, err = t.b.ListExecutables(ctx, uri)
		if err != nil {
			return nil, err
		}
	} else {
		traceUri = uri
	}

	dir, file := filepath.Split(uri)
	name := file
	if name == "" {
		name = dir
	}

	children := append(dirs, files...)
	for i := range children {
		children[i] = filepath.Join(uri, children[i])
		// filepath.Join will clean off preceding .
		if uri == "." {
			if runtime.GOOS == "windows" {
				children[i] = ".\\" + children[i]
			} else {
				children[i] = "./" + children[i]
			}
		}
	}

	tttn := &tracer.TraceTargetTreeNode{
		Name:            name,
		Icon:            nil,
		URI:             uri,
		TraceURI:        traceUri,
		Children:        children,
		Parent:          dir,
		ApplicationName: "",
		ExecutableName:  file,
	}

	return tttn, nil
}

func (t *DesktopTracer) SetupTrace(ctx context.Context, o *tracer.TraceOptions) (*gapii.Process, func(), error) {
	env, err := t.b.GetEnv(ctx)
	if err != nil {
		return nil, nil, err
	}
	cleanup, portFile, err := loader.SetupTrace(ctx, t.b, t.b.Instance().Configuration.ABIs[0], env)
	if err != nil {
		cleanup(ctx)
		return nil, nil, err
	}
	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	args := r.FindAllString(o.AdditionalFlags, -1)

	for _, x := range o.Environment {
		env.Add(x)
	}

	boundPort, err := process.StartOnDevice(ctx, o.URI, process.StartOptions{
		Env:        env,
		Args:       args,
		PortFile:   portFile,
		WorkingDir: o.CWD,
		Device:     t.b,
	})

	if err != nil {
		cleanup(ctx)
		return nil, nil, err
	}
	process := &gapii.Process{Port: boundPort, Device: t.b, Options: o.GapiiOptions()}
	return process, func() { cleanup(ctx) }, nil
}

func (t *DesktopTracer) PreferredRootUri(ctx context.Context) (string, error) {
	return t.b.GetWorkingDirectory(ctx)
}
