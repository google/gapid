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

package tracer

import (
	"bytes"
	"context"
	"io"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/device/bind"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// TraceTargetTreeNode represents a node in the traceable application
// Tree
type TraceTargetTreeNode struct {
	Name            string   // What is the name of this tree node
	Icon            []byte   // What is the icon for this node
	URI             string   // What is the URI of this node
	TraceURI        string   // Can this node be traced
	Children        []string // Child URIs of this node
	Parent          string   // What is the URI of this node's parent
	ApplicationName string   // The friendly application name for the trace node if it exists
	ExecutableName  string   // The friendly executable name for the trace node if it exists
}

// Process is a handle to an initialized trace that can be started.
type Process interface {
	// Capture connects to this trace and waits for a capture to be delivered.
	// It copies the capture into the supplied writer.
	// If the process was started with the DeferStart flag, then tracing will wait
	// until start is fired.
	// Capturing will stop when the stop signal is fired (clean stop) or the
	// context is cancelled (abort).
	Capture(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64) (size int64, err error)
}

// Tracer is an option interface that a bind.Device can implement.
// If it exists, it is used to set up and connect to a tracing application.
type Tracer interface {
	// TraceConfiguration returns the device's supported trace configuration.
	TraceConfiguration(ctx context.Context) (*service.DeviceTraceConfiguration, error)
	// GetTraceTargetNode returns a TraceTargetTreeNode for the given URI
	// on the device
	GetTraceTargetNode(ctx context.Context, uri string, iconDensity float32) (*TraceTargetTreeNode, error)
	// FindTraceTargets finds TraceTargetTreeNodes for a given search string on
	// the device
	FindTraceTargets(ctx context.Context, uri string) ([]*TraceTargetTreeNode, error)

	// SetupTrace starts the application on the device, and causes it to wait
	// for the trace to be started. It returns the process that was created, as
	// well as a function that can be used to clean up the device
	SetupTrace(ctx context.Context, o *service.TraceOptions) (Process, app.Cleanup, error)

	// GetDevice returns the device associated with this tracer
	GetDevice() bind.Device
	// ProcessProfilingData takes a buffer for a Perfetto trace and translates it into
	// a ProfilingData
	ProcessProfilingData(ctx context.Context, buffer *bytes.Buffer, capture *path.Capture, handleMapping *map[uint64][]service.VulkanHandleMappingItem, syncData *sync.Data) (*service.ProfilingData, error)
	// Validate validates the GPU profiling capabilities of the given device and returns
	// an error if validation failed or the GPU profiling data is invalid.
	Validate(ctx context.Context) error
}

// LayersFromOptions Parses the perfetto options, and returns the required layers
func LayersFromOptions(ctx context.Context, o *service.TraceOptions) []string {
	ret := []string{}
	if o.PerfettoConfig == nil {
		return ret
	}

	added := map[string]struct{}{}
	for _, x := range o.PerfettoConfig.GetDataSources() {
		if layer, err := layout.LayerFromDataSource(x.GetConfig().GetName()); err == nil {
			if _, err := layout.LibraryFromLayerName(layer); err == nil {
				if _, ok := added[layer]; !ok {
					added[layer] = struct{}{}
					ret = append(ret, layer)
				}
			}
		}
	}
	return ret
}

// GapiiOptions converts the given TraceOptions to gapii.Options.
func GapiiOptions(o *service.TraceOptions) gapii.Options {
	apis := uint32(0)
	for _, api := range o.Apis {
		if api == "Vulkan" {
			apis |= gapii.VulkanAPI
		}
	}

	flags := gapii.Flags(0)
	if o.DeferStart {
		flags |= gapii.DeferStart
	}
	if o.NoBuffer {
		flags |= gapii.NoBuffer
	}
	if o.HideUnknownExtensions {
		flags |= gapii.HideUnknownExtensions
	}
	if o.RecordTraceTimes {
		flags |= gapii.StoreTimestamps
	}
	if o.DisableCoherentMemoryTracker {
		flags |= gapii.DisableCoherentMemoryTracker
	}

	return gapii.Options{
		o.ObserveFrameFrequency,
		o.ObserveDrawFrequency,
		o.StartFrame,
		o.FramesToCapture,
		apis,
		flags,
		o.AdditionalCommandLineArgs,
		o.PipeName,
	}
}
