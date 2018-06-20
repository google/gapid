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

package service

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/service/severity"
	"github.com/google/gapid/gapis/stringtable"
)

type Severity = severity.Severity

const (
	Severity_VerboseLevel Severity = 0
	Severity_DebugLevel   Severity = 1
	Severity_InfoLevel    Severity = 2
	Severity_WarningLevel Severity = 3
	Severity_ErrorLevel   Severity = 4
	Severity_FatalLevel   Severity = 5
)

type Service interface {
	// Ping is a no-op function that returns immediately.
	// It can be used to measure connection latency or to keep the
	// process alive if started with the "idle-timeout" command line flag.
	Ping(ctx context.Context) error

	// GetServerInfo returns information about the running server.
	GetServerInfo(ctx context.Context) (*ServerInfo, error)

	// CheckForUpdates checks for a new build of GAPID on the hosting server.
	// Care should be taken to call this infrequently to avoid reaching the
	// server's maximum unauthenticated request limits.
	CheckForUpdates(ctx context.Context, includePrereleases bool) (*Release, error)

	// GetAvailableStringTables returns list of available string table descriptions.
	GetAvailableStringTables(ctx context.Context) ([]*stringtable.Info, error)

	// GetStringTable returns the requested string table.
	GetStringTable(ctx context.Context, info *stringtable.Info) (*stringtable.StringTable, error)

	// ImportCapture imports capture data emitted by the graphics spy, returning
	// the new capture identifier.
	ImportCapture(ctx context.Context, name string, data []uint8) (*path.Capture, error)

	// ExportCapture returns a capture's data that can be consumed by
	// ImportCapture or LoadCapture.
	ExportCapture(ctx context.Context, c *path.Capture) ([]byte, error)

	// LoadCapture imports capture data from a local file, returning the new
	// capture identifier.
	LoadCapture(ctx context.Context, path string) (*path.Capture, error)

	// SaveCapture saves the capture to a local file.
	SaveCapture(ctx context.Context, c *path.Capture, path string) error

	// GetDevices returns the full list of replay devices avaliable to the server.
	// These include local replay devices and any connected Android devices.
	// This list may change over time, as devices are connected and disconnected.
	// If both connected Android and Local replay devices are found,
	// the local Android devices will be returned first.
	GetDevices(ctx context.Context) ([]*path.Device, error)

	// GetDevicesForReplay returns the list of replay devices avaliable to the
	// server that are capable of replaying the given capture.
	// These include local replay devices and any connected Android devices.
	// This list may change over time, as devices are connected and disconnected.
	// If both connected Android and Local replay devices are found,
	// the local Android devices will be returned first.
	GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, error)

	// GetFramebufferAttachment returns the ImageInfo identifier describing the
	// given framebuffer attachment and device, immediately following the
	// command after.
	// The provided RenderSettings structure can be used to adjust maximum desired
	// dimensions of the image, as well as applying debug visualizations.
	GetFramebufferAttachment(
		ctx context.Context,
		replaySettings *ReplaySettings,
		after *path.Command,
		attachment api.FramebufferAttachment,
		settings *RenderSettings,
		hints *UsageHints) (*path.ImageInfo, error)

	// Get resolves and returns the object, value or memory at the path p.
	Get(ctx context.Context, p *path.Any) (interface{}, error)

	// Set creates a copy of the capture referenced by p, but with the object, value
	// or memory at p replaced with v. The path returned is identical to p, but with
	// the base changed to refer to the new capture.
	Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error)

	// Follow returns the path to the object that the value at p links to.
	// If the value at p does not link to anything then nil is returned.
	Follow(ctx context.Context, p *path.Any) (*path.Any, error)

	// BeginCPUProfile starts CPU self-profiling of the server.
	// If the CPU is already being profiled then this function will return an
	// error.
	// This is a debug API, and may be removed in the future.
	BeginCPUProfile(ctx context.Context) error

	// EndCPUProfile ends the CPU profile, returning the pprof samples.
	// This is a debug API, and may be removed in the future.
	EndCPUProfile(ctx context.Context) ([]byte, error)

	// GetPerformanceCounters returns the values of all global counters as
	// a string.
	GetPerformanceCounters(ctx context.Context) (string, error)

	// GetProfile returns the pprof profile with the given name.
	GetProfile(ctx context.Context, name string, debug int32) ([]byte, error)

	// GetLogStream calls the handler with each log record raised until the
	// context is cancelled.
	GetLogStream(context.Context, log.Handler) error

	// Find performs a search using req, streaming the results to h.
	Find(ctx context.Context, req *FindRequest, h FindHandler) error

	// EnableCrashReporting enables or disables crash reporting for this session.
	EnableCrashReporting(ctx context.Context, enable bool) error

	// EnableAnalytics enables or disables analytics reporting for this session.
	EnableAnalytics(ctx context.Context, enable bool, clientID string) error

	// ClientEvent records a client event action, used for analytics.
	// If the user has not opted-in for analytics then this call does nothing.
	ClientEvent(ctx context.Context, req *ClientEventRequest) error
}

// FindHandler is the handler of found items using Service.Find.
type FindHandler func(*FindResponse) error

// NewError attempts to box and return err into an Error.
// If err cannot be boxed into an Error then nil is returned.
func NewError(err error) *Error {
	if err == nil {
		return nil
	}
	type causer interface {
		Cause() error
	}
	cause := err
	for i := 0; i < 64 && cause != nil; i++ {
		switch err := cause.(type) {
		case *ErrDataUnavailable:
			return &Error{Err: &Error_ErrDataUnavailable{err}}
		case *ErrInvalidPath:
			return &Error{Err: &Error_ErrInvalidPath{err}}
		case *ErrInvalidArgument:
			return &Error{Err: &Error_ErrInvalidArgument{err}}
		case *ErrPathNotFollowable:
			return &Error{Err: &Error_ErrPathNotFollowable{err}}
		case *ErrUnsupportedVersion:
			return &Error{Err: &Error_ErrUnsupportedVersion{err}}
		}

		causer, ok := cause.(causer)
		if !ok {
			break
		}
		cause = causer.Cause()
	}
	return &Error{Err: &Error_ErrInternal{&ErrInternal{Message: err.Error()}}}
}

// Get returns the boxed error.
func (v *Error) Get() error { return protoutil.OneOf(v.Err).(error) }

// NewValue attempts to box and return v into a Value.
// If v cannot be boxed into a Value then nil is returned.
func NewValue(v interface{}) *Value {
	switch v := v.(type) {
	case nil:
		return &Value{}
	case *Capture:
		return &Value{Val: &Value_Capture{v}}
	case *Context:
		return &Value{Val: &Value_Context{v}}
	case *Contexts:
		return &Value{Val: &Value_Contexts{v}}
	case *Commands:
		return &Value{Val: &Value_Commands{v}}
	case *CommandTree:
		return &Value{Val: &Value_CommandTree{v}}
	case *CommandTreeNode:
		return &Value{Val: &Value_CommandTreeNode{v}}
	case *ConstantSet:
		return &Value{Val: &Value_ConstantSet{v}}
	case *Event:
		return &Value{Val: &Value_Event{v}}
	case *Events:
		return &Value{Val: &Value_Events{v}}
	case *Memory:
		return &Value{Val: &Value_Memory{v}}
	case *path.Any:
		return &Value{Val: &Value_Path{v}}
	case path.Node:
		return &Value{Val: &Value_Path{v.Path()}}
	case *Report:
		return &Value{Val: &Value_Report{v}}
	case *Resources:
		return &Value{Val: &Value_Resources{v}}
	case *StateTree:
		return &Value{Val: &Value_StateTree{v}}
	case *StateTreeNode:
		return &Value{Val: &Value_StateTreeNode{v}}
	case *api.Command:
		return &Value{Val: &Value_Command{v}}
	case *api.Mesh:
		return &Value{Val: &Value_Mesh{v}}
	case *api.Metrics:
		return &Value{Val: &Value_Metrics{v}}
	case *api.ResourceData:
		return &Value{Val: &Value_ResourceData{v}}
	case *image.Info:
		return &Value{Val: &Value_ImageInfo{v}}
	case *device.Instance:
		return &Value{Val: &Value_Device{v}}

	default:
		if v := box.NewValue(v); v != nil {
			return &Value{Val: &Value_Box{v}}
		}
	}
	panic(fmt.Errorf("Cannot box value type %T", v))
}

// Get returns the boxed value.
func (v *Value) Get() interface{} {
	switch v := v.Val.(type) {
	case nil:
		return nil
	case *Value_Box:
		return v.Box.Get()
	default:
		return protoutil.OneOf(v)
	}
}

// NewMemoryRange constructs and returns a new MemoryRange from the
// memory.Range.
func NewMemoryRange(rng memory.Range) *MemoryRange {
	return &MemoryRange{Base: rng.Base, Size: rng.Size}
}

// NewMemoryRanges constructs and returns a new slice of MemoryRanges from the
// memory.RangeList.
func NewMemoryRanges(l memory.RangeList) []*MemoryRange {
	out := make([]*MemoryRange, len(l))
	for i, r := range l {
		out[i] = NewMemoryRange(r)
	}
	return out
}

// Find looks up a resource by type and identifier.
func (r *Resources) Find(ty api.ResourceType, id id.ID) *Resource {
	for _, t := range r.Types {
		if t.Type == ty {
			for _, r := range t.Resources {
				if r.ID.ID() == id {
					return r
				}
			}
			break
		}
	}
	return nil
}
