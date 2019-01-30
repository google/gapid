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
	"context"
	"io"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/device/bind"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/service"
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
	Capture(ctx context.Context, start task.Signal, stop task.Signal, w io.Writer, written *int64) (size int64, err error)
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
	SetupTrace(ctx context.Context, o *service.TraceOptions) (Process, func(), error)

	// GetDevice returns the device associated with this tracer
	GetDevice() bind.Device
}

// GapiiOptions converts the given TraceOptions to gapii.Options.
func GapiiOptions(o *service.TraceOptions) gapii.Options {
	apis := uint32(0)
	for _, api := range o.Apis {
		if api == "OpenGLES" ||
			api == "GVR" {
			apis |= gapii.GlesAPI
			apis |= gapii.GvrAPI
		}
		if api == "Vulkan" {
			apis |= gapii.VulkanAPI
		}
	}

	flags := gapii.Flags(0)
	if o.DisablePcs {
		flags |= gapii.DisablePrecompiledShaders
	}
	if o.RecordErrorState {
		flags |= gapii.RecordErrorState
	}
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
