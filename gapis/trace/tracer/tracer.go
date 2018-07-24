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

	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/service"
)

// APITraceOptions represents API-sepcific trace options for a given
// device.
type APITraceOptions struct {
	APIName                    string                // APIName is the name of the API in question
	CanDisablePCS              bool                  // Does it make sense for this API to disable PCS
	MidExecutionCaptureSupport service.FeatureStatus // Does this API support MEC
}

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

// TraceOptions are the device specific parameters to start
// a trace on the device.
type TraceOptions struct {
	URI               string   // What application should be traced
	UploadApplication []byte   // This application should be uploaded and traced
	ClearCache        bool     // Should the cache be cleared on the device
	PortFile          string   // What port file should be written to
	APIs              []string // What apis should be traced
	Environment       []string // Environment variables to set for the trace
	WriteFile         string   // Where should we write the output
	AdditionalFlags   string   // Additional flags to pass to the application
	CWD               string   // What directory the application use as a CWD

	Duration              float32 // How many seconds should we trace
	ObserveFrameFrequency uint32  // How frequently should we do frame observations
	ObserveDrawFrequency  uint32  // How frequently should we do draw observations
	StartFrame            uint32  // What frame should we start capturing
	FramesToCapture       uint32  // How many frames should we capture
	DisablePCS            bool    // Should we disable PCS
	RecordErrorState      bool    // Should we record the driver error state after each command
	DeferStart            bool    // Should we record extra error state
	NoBuffer              bool    // Disable buffering.
	HideUnknownExtensions bool    // Hide unknown extensions from the application.
}

// Tracer is an option interface that a bind.Device can implement.
// If it exists, it is used to set up and connect to a tracing application.
type Tracer interface {
	// IsServerLocal returns true if all paths on this device can be server-local
	IsServerLocal() bool
	// CanSpecifyCWD returns true if this device has the concept of a CWD
	CanSpecifyCWD() bool
	// CanUploadApplication returns true if an application can be uploaded to this device
	CanUploadApplication() bool
	// HasCache returns true if the device has an appliction cache that can be cleared
	HasCache() bool
	// CanSpecifyEnv() returns true if you can specify environment variables for the tracer
	CanSpecifyEnv() bool

	// PreferredRootUri returns the preferred path to search URIs
	PreferredRootUri(ctx context.Context) (string, error)

	// TraceOptions returns API-specific trace options for this device
	APITraceOptions(ctx context.Context) []APITraceOptions
	// GetTraceTargetNode returns a TraceTargetTreeNode for the given URI
	// on the device
	GetTraceTargetNode(ctx context.Context, uri string, iconDensity float32) (*TraceTargetTreeNode, error)
	// FindTraceTarget finds and returns a unique TraceTargetTreenode for a given
	// search string on the device
	FindTraceTarget(ctx context.Context, uri string) (*TraceTargetTreeNode, error)

	// SetupTrace starts the application on the device, and causes it to wait
	// for the trace to be started. It returns the process that was created, as
	// well as a function that can be used to clean up the device
	SetupTrace(ctx context.Context, o *TraceOptions) (*gapii.Process, func(), error)
}

// GapiiOptions converts the given TraceOptions to gapii.Options.
func (o TraceOptions) GapiiOptions() gapii.Options {
	apis := uint32(0)
	for _, api := range o.APIs {
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
	if o.DisablePCS {
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

	return gapii.Options{
		o.ObserveFrameFrequency,
		o.ObserveDrawFrequency,
		o.StartFrame,
		o.FramesToCapture,
		apis,
		flags,
		o.AdditionalFlags,
	}
}
