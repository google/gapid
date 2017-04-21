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
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"

	_ "github.com/google/gapid/framework/binary/any"
)

type Service interface {
	// GetServerInfo returns information about the running server.
	GetServerInfo(ctx context.Context) (*ServerInfo, error)

	// The GetSchema returns the type and constant schema descriptions for all
	// objects used in the api.
	// This includes all the types included in or referenced from the atom stream.
	GetSchema(ctx context.Context) (*schema.Message, error)

	// GetAvailableStringTables returns list of available string table descriptions.
	GetAvailableStringTables(ctx context.Context) ([]*stringtable.Info, error)

	// GetStringTable returns the requested string table.
	GetStringTable(ctx context.Context, info *stringtable.Info) (*stringtable.StringTable, error)

	// Import imports capture data emitted by the graphics spy, returning the new
	// capture identifier.
	ImportCapture(ctx context.Context, name string, data []uint8) (*path.Capture, error)

	// LoadCapture imports capture data from a local file, returning the new
	// capture identifier.
	LoadCapture(ctx context.Context, path string) (*path.Capture, error)

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
	// given framebuffer attachment and device, immediately following the atom
	// after.
	// The provided RenderSettings structure can be used to adjust maximum desired
	// dimensions of the image, as well as applying debug visualizations.
	GetFramebufferAttachment(
		ctx context.Context,
		device *path.Device,
		after *path.Command,
		attachment gfxapi.FramebufferAttachment,
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
	// a JSON blob.
	GetPerformanceCounters(ctx context.Context) ([]byte, error)

	// GetProfile returns the pprof profile with the given name.
	GetProfile(ctx context.Context, name string, debug int32) ([]byte, error)

	// GetLogStream calls the handler with each log record raised until the
	// context is cancelled.
	GetLogStream(context.Context, log.Handler) error
}

// NewError attempts to box and return err into an Error.
// If err cannot be boxed into an Error then nil is returned.
func NewError(err error) *Error {
	switch err := err.(type) {
	case nil:
		return nil
	case *ErrDataUnavailable:
		return &Error{&Error_ErrDataUnavailable{err}}
	case *ErrInvalidPath:
		return &Error{&Error_ErrInvalidPath{err}}
	case *ErrInvalidArgument:
		return &Error{&Error_ErrInvalidArgument{err}}
	case *ErrPathNotFollowable:
		return &Error{&Error_ErrPathNotFollowable{err}}
	default:
		return &Error{&Error_ErrInternal{&ErrInternal{err.Error()}}}
	}
}

// Get returns the boxed error.
func (v *Error) Get() error { return protoutil.OneOf(v.Err).(error) }

// NewValue attempts to box and return v into a Value.
// If v cannot be boxed into a Value then nil is returned.
func NewValue(v interface{}) *Value {
	if v := pod.NewValue(v); v != nil {
		return &Value{&Value_Pod{v}}
	}
	switch v := v.(type) {
	case nil:
		return &Value{}
	case binary.Object:
		out := &Object{}
		if err := out.Encode(v); err != nil {
			panic(err)
		}
		return &Value{Val: &Value_Object{out}}

	case *Capture:
		return &Value{&Value_Capture{v}}
	case *Contexts:
		return &Value{&Value_Contexts{v}}
	case []*Context:
		return &Value{&Value_Contexts{&Contexts{v}}}
	case *gfxapi.Mesh:
		return &Value{&Value_Mesh{v}}
	case *gfxapi.Texture2D:
		return &Value{&Value_Texture_2D{v}}
	case *gfxapi.Cubemap:
		return &Value{&Value_Cubemap{v}}
	case *gfxapi.Shader:
		return &Value{&Value_Shader{v}}
	case *gfxapi.Program:
		return &Value{&Value_Program{v}}
	case *Hierarchies:
		return &Value{&Value_Hierarchies{v}}
	case []*Hierarchy:
		return &Value{&Value_Hierarchies{&Hierarchies{v}}}
	case *image.Info2D:
		return &Value{&Value_ImageInfo_2D{v}}
	case *MemoryInfo:
		return &Value{&Value_MemoryInfo{v}}
	case *Report:
		return &Value{&Value_Report{v}}
	case *Resources:
		return &Value{&Value_Resources{v}}
	case *device.Instance:
		return &Value{&Value_Device{v}}

	default:
		panic(fmt.Errorf("Cannot box value type %T", v))
	}
}

// Get returns the boxed value.
func (v *Value) Get() interface{} {
	switch v := v.Val.(type) {
	case nil:
		return nil

	case *Value_Pod:
		return v.Pod.Get()

	case *Value_Object:
		o, err := v.Object.Decode()
		if err != nil {
			panic(err)
		}
		return o

	case *Value_Contexts:
		return v.Contexts.List
	case *Value_Hierarchies:
		return v.Hierarchies.List

	default:
		return protoutil.OneOf(v)
	}
}

// NewCommandRangeList returns a new slice of CommandRange filled with a copy of
// l.
func NewCommandRangeList(l atom.RangeList) []*CommandRange {
	out := make([]*CommandRange, len(l))
	for i, r := range l {
		out[i] = &CommandRange{
			First: r.Start,
			Count: r.End - r.Start,
		}
	}
	return out
}

// NewCommandGroup constructs and returns a new CommandGroup from the
// atom.Group.
func NewCommandGroup(group atom.Group) *CommandGroup {
	subgroups := make([]*CommandGroup, len(group.SubGroups))
	for i, g := range group.SubGroups {
		subgroups[i] = NewCommandGroup(g)
	}
	return &CommandGroup{
		Name:      group.Name,
		Range:     &CommandRange{First: group.Range.Start, Count: group.Range.Length()},
		Subgroups: subgroups,
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

// NewHierarchy constructs and returns a new Hierarchy.
func NewHierarchy(name string, context id.ID, root atom.Group) *Hierarchy {
	return &Hierarchy{
		Name:    name,
		Context: path.NewID(context),
		Root:    NewCommandGroup(root),
	}
}

// Link returns the link to the atom pointed by a report item.
// If nil, nil is returned then the path cannot be followed.
func (i ReportItem) Link(ctx context.Context, p path.Node) (path.Node, error) {
	if capture := path.FindCapture(p); capture != nil {
		return capture.Commands().Index(i.Command), nil
	}
	return nil, nil
}

// Find looks up a resource by type and identifier.
func (r *Resources) Find(ty gfxapi.ResourceType, id id.ID) *Resource {
	for _, t := range r.Types {
		if t.Type == ty {
			for _, r := range t.Resources {
				if r.Id.ID() == id {
					return r
				}
			}
			break
		}
	}
	return nil
}
