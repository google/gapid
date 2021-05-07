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
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/vulkan/loader"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/api/sync"
	perfetto "github.com/google/gapid/gapis/perfetto/desktop"
	"github.com/google/gapid/gapis/service"
	gapis_path "github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace/tracer"
)

const (
	// captureProcessNameEnvVar is the environment variable holding the name of
	// the process to capture. Mirrored in gapii/cc/spy.cpp.
	captureProcessNameEnvVar = "GAPID_CAPTURE_PROCESS_NAME"
)

type DesktopTracer struct {
	b bind.Device
}

func NewTracer(dev bind.Device) *DesktopTracer {
	return &DesktopTracer{dev.(bind.Device)}
}

func (t *DesktopTracer) GetDevice() bind.Device {
	return t.b
}

func (t *DesktopTracer) ProcessProfilingData(ctx context.Context, buffer *bytes.Buffer, capture *gapis_path.Capture, handleMapping *map[uint64][]service.VulkanHandleMappingItem, syncData *sync.Data) (*service.ProfilingData, error) {
	return nil, log.Err(ctx, nil, "Desktop replay profiling is unsupported.")
}

func (t *DesktopTracer) Validate(ctx context.Context) error {
	return nil
}

// TraceConfiguration returns the device's supported trace configuration.
func (t *DesktopTracer) TraceConfiguration(ctx context.Context) (*service.DeviceTraceConfiguration, error) {
	apis := make([]*service.TraceTypeCapabilities, 0, 1)
	if len(t.b.Instance().GetConfiguration().GetDrivers().GetVulkan().GetPhysicalDevices()) > 0 && *flags.EnableVulkanTracing {
		apis = append(apis, tracer.VulkanTraceOptions())
	}

	preferredRoot, err := t.b.GetWorkingDirectory(ctx)
	if err != nil {
		return nil, err
	}

	isLocal, err := t.b.IsLocal(ctx)
	if err != nil {
		return nil, err
	}

	if t.b.SupportsPerfetto(ctx) {
		apis = append(apis, tracer.PerfettoTraceOptions())
	}

	return &service.DeviceTraceConfiguration{
		Apis:                 apis,
		ServerLocalPath:      isLocal,
		CanSpecifyCwd:        true,
		CanUploadApplication: false,
		CanSpecifyEnv:        true,
		PreferredRootUri:     preferredRoot,
		HasCache:             false,
	}, nil
}

// JoinPath provides a path.Join() for this specific target
func (t *DesktopTracer) JoinPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	if t.b.Instance().GetConfiguration().GetOS().GetKind() == device.Windows {
		if paths[0] == "" {
			return path.Clean(strings.Join(paths[1:], "\\"))
		}
		return path.Clean(strings.Join(paths, "\\"))
	} else {
		return path.Join(paths...)
	}
}

// SplitPath performs a path.Split() operation for this specific target
func (t *DesktopTracer) SplitPath(p string) (string, string) {
	if t.b.Instance().GetConfiguration().GetOS().GetKind() == device.Windows {
		if runtime.GOOS == "windows" {
			return filepath.Split(p)
		} else {
			lastSlash := strings.LastIndex(p, "\\")
			if lastSlash == -1 {
				return "", p
			}
			return p[0 : lastSlash+1], p[lastSlash+1:]
		}
	} else {
		return path.Split(p)
	}
}

func (t *DesktopTracer) FindTraceTargets(ctx context.Context, str string) ([]*tracer.TraceTargetTreeNode, error) {
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
		if t.b.Instance().GetConfiguration().GetOS().GetKind() == device.Windows {
			str = ".\\" + file
		} else {
			str = "./" + file
		}
	}

	node := &tracer.TraceTargetTreeNode{
		Name:            file,
		Icon:            nil,
		URI:             str,
		TraceURI:        str,
		Children:        nil,
		Parent:          dir,
		ApplicationName: "",
		ExecutableName:  file,
	}

	return []*tracer.TraceTargetTreeNode{node}, nil
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

	dir, file := t.SplitPath(uri)
	name := file
	if name == "" {
		name = dir
	}

	children := append(dirs, files...)
	for i := range children {
		children[i] = t.JoinPath([]string{uri, children[i]})
		if uri == "." {
			if t.b.Instance().GetConfiguration().GetOS().GetKind() == device.Windows {
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

func (t *DesktopTracer) SetupTrace(ctx context.Context, o *service.TraceOptions) (tracer.Process, app.Cleanup, error) {
	env, err := t.b.GetEnv(ctx)
	if err != nil {
		return nil, nil, err
	}
	var portFile string
	var cleanup app.Cleanup

	ignorePort := true
	if o.Type == service.TraceType_Graphics {
		if o.LoadValidationLayer {
			// We don't ship desktop builds of the VVL, and we have no guarantee
			// that they are present on the desktop.
			log.W(ctx, "Loading Vulkan validation layer at capture time is not supported on desktop")
		}
		cleanup, portFile, err = loader.SetupTrace(ctx, t.b, t.b.Instance().Configuration.ABIs[0], env)
		if err != nil {
			cleanup.Invoke(ctx)
			return nil, nil, err
		}
		ignorePort = false

	}
	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	args := r.FindAllString(o.AdditionalCommandLineArgs, -1)

	for _, x := range o.Environment {
		env.Add(x)
	}
	if o.ProcessName != "" {
		env.Add(captureProcessNameEnvVar + "=" + o.ProcessName)
	}
	var p tracer.Process
	var boundPort int

	if o.Type == service.TraceType_Perfetto {
		layers := tracer.LayersFromOptions(ctx, o)
		c, err := loader.SetupLayers(ctx, layers, false, t.b, t.b.Instance().Configuration.ABIs[0], env)
		if err != nil {
			cleanup.Invoke(ctx)
		}
		cleanup = cleanup.Then(c)
		p, err = perfetto.Start(ctx, t.b, t.b.Instance().Configuration.ABIs[0], o)
	}

	if o.GetUri() != "" {
		log.D(ctx, "URI provided for trace: %+v", o.GetUri())
		boundPort, err = process.StartOnDevice(ctx, o.GetUri(), process.StartOptions{
			Env:        env,
			Args:       args,
			PortFile:   portFile,
			WorkingDir: o.Cwd,
			Device:     t.b,
			IgnorePort: ignorePort,
		})
	}

	if err != nil {
		cleanup.Invoke(ctx)
		return nil, nil, err
	}
	if p != nil {
		return p, cleanup, nil
	}
	process := &gapii.Process{Port: boundPort, Device: t.b, Options: tracer.GapiiOptions(o)}
	return process, cleanup, nil
}
