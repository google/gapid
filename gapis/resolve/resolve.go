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

package resolve

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/service/types"
	"github.com/google/gapid/gapis/trace"
)

// Capture resolves and returns the capture from the path p.
func Capture(ctx context.Context, p *path.Capture, r *path.ResolveConfig) (*service.Capture, error) {
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.Service(ctx, p), nil
}

// Device resolves and returns the device from the path p.
func Device(ctx context.Context, p *path.Device, r *path.ResolveConfig) (*device.Instance, error) {
	device := bind.GetRegistry(ctx).Device(p.ID.ID())
	if device == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrUnknownDevice()}
	}
	return device.Instance(), nil
}

// DeviceTraceConfiguration resolves and returns the trace config for a device.
func DeviceTraceConfiguration(ctx context.Context, p *path.DeviceTraceConfiguration, r *path.ResolveConfig) (*service.DeviceTraceConfiguration, error) {
	return trace.TraceConfiguration(ctx, p.Device)
}

// ImageInfo resolves and returns the ImageInfo from the path p.
func ImageInfo(ctx context.Context, p *path.ImageInfo, r *path.ResolveConfig) (*image.Info, error) {
	obj, err := database.Resolve(ctx, p.ID.ID())
	if err != nil {
		return nil, err
	}
	ii, ok := obj.(*image.Info)
	if !ok {
		return nil, fmt.Errorf("Path %s gave %T, expected *image.Info", p, obj)
	}
	return ii, err
}

// Blob resolves and returns the byte slice from the path p.
func Blob(ctx context.Context, p *path.Blob, r *path.ResolveConfig) ([]byte, error) {
	obj, err := database.Resolve(ctx, p.ID.ID())
	if err != nil {
		return nil, err
	}
	bytes, ok := obj.([]byte)
	if !ok {
		return nil, fmt.Errorf("Path %s gave %T, expected []byte", p, obj)
	}
	return bytes, nil
}

// Field resolves and returns the field from the path p.
func Field(ctx context.Context, p *path.Field, r *path.ResolveConfig) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}
	v, err := field(ctx, reflect.ValueOf(obj), p.Name, p)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

func Type(ctx context.Context, p *path.Type, r *path.ResolveConfig) (interface{}, error) {
	t, err := types.GetType(p.TypeIndex)
	if _, isEnum := t.GetTy().(*types.Type_Enum); isEnum {
		t.GetEnum().Constants.API = p.API
	}
	return t, err
}

func Messages(ctx context.Context, p *path.Messages) (interface{}, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	m := &service.Messages{List: []*service.Message{}}
	for _, message := range c.Messages {
		m.List = append(m.List, &service.Message{
			Timestamp: message.Timestamp,
			Message:   message.Message,
		})
	}
	return m, nil
}

func field(ctx context.Context, s reflect.Value, name string, p path.Node) (reflect.Value, error) {
	for {
		if isNil(s) {
			return reflect.Value{}, &service.ErrInvalidPath{
				Reason: messages.ErrNilPointerDereference(),
				Path:   p.Path(),
			}
		}

		if pp, ok := s.Interface().(api.PropertyProvider); ok {
			if p := pp.Properties().Find(name); p != nil {
				return reflect.ValueOf(p.Get()), nil
			}
		}

		switch s.Kind() {
		case reflect.Struct:
			f := s.FieldByName(name)
			if !f.IsValid() {
				return reflect.Value{}, &service.ErrInvalidPath{
					Reason: messages.ErrFieldDoesNotExist(typename(s.Type()), name),
					Path:   p.Path(),
				}
			}
			return f, nil
		case reflect.Interface, reflect.Ptr:
			s = s.Elem()
		default:
			return reflect.Value{}, &service.ErrInvalidPath{
				Reason: messages.ErrFieldDoesNotExist(typename(s.Type()), name),
				Path:   p.Path(),
			}
		}
	}
}

// ArrayIndex resolves and returns the array or slice element from the path p.
func ArrayIndex(ctx context.Context, p *path.ArrayIndex, r *path.ResolveConfig) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}

	a := reflect.ValueOf(obj)
	switch {
	case box.IsMemorySlice(a.Type()):
		slice := box.AsMemorySlice(a)
		if count := slice.Count(); p.Index >= count {
			return nil, errPathOOB(p.Index, "Index", 0, count-1, p)
		}
		return slice.ISlice(p.Index, p.Index+1), nil

	default:
		switch a.Kind() {
		case reflect.Array, reflect.Slice, reflect.String:
			if count := uint64(a.Len()); p.Index >= count {
				return nil, errPathOOB(p.Index, "Index", 0, count-1, p)
			}
			return a.Index(int(p.Index)).Interface(), nil

		default:
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrTypeNotArrayIndexable(typename(a.Type())),
				Path:   p.Path(),
			}
		}
	}
}

// Slice resolves and returns the subslice from the path p.
func Slice(ctx context.Context, p *path.Slice, r *path.ResolveConfig) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}
	a := reflect.ValueOf(obj)
	switch {
	case box.IsMemorySlice(a.Type()):
		slice := box.AsMemorySlice(a)
		if count := slice.Count(); p.Start >= count || p.End > count {
			return nil, errPathSliceOOB(p.Start, p.End, count, p)
		}
		return slice.ISlice(p.Start, p.End), nil

	default:
		switch a.Kind() {
		case reflect.Array, reflect.Slice, reflect.String:
			if int(p.Start) >= a.Len() || int(p.End) > a.Len() {
				return nil, errPathSliceOOB(p.Start, p.End, uint64(a.Len()), p)
			}
			return a.Slice(int(p.Start), int(p.End)).Interface(), nil

		default:
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrTypeNotSliceable(typename(a.Type())),
				Path:   p.Path(),
			}
		}
	}
}

// MapIndex resolves and returns the map value from the path p.
func MapIndex(ctx context.Context, p *path.MapIndex, r *path.ResolveConfig) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}

	d := dictionary.From(obj)
	if d == nil {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable(typename(reflect.TypeOf(obj))),
			Path:   p.Path(),
		}
	}

	key, ok := convert(reflect.ValueOf(p.KeyValue()), d.KeyTy())
	if !ok {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType(
				typename(reflect.TypeOf(p.KeyValue())), // got
				typename(d.KeyTy())),                   // expected
			Path: p.Path(),
		}
	}

	val, ok := d.Lookup(key.Interface())
	if !ok {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrMapKeyDoesNotExist(key.Interface()),
			Path:   p.Path(),
		}
	}
	return val, nil
}

// memoryLayout resolves the memory layout for the capture of the given path.
func memoryLayout(ctx context.Context, p path.Node) (*device.MemoryLayout, error) {
	cp := path.FindCapture(p)
	if cp == nil {
		return nil, errPathNoCapture(p)
	}

	c, err := capture.ResolveGraphicsFromPath(ctx, cp)
	if err != nil {
		return nil, err
	}

	return c.Header.ABI.MemoryLayout, nil
}

// ResolveService resolves and returns the object, value or memory at the path p,
// converting the final result to the service representation.
func ResolveService(ctx context.Context, p path.Node, r *path.ResolveConfig) (interface{}, error) {
	v, err := ResolveInternal(ctx, p, r)
	if err != nil {
		return nil, err
	}
	return internalToService(v)
}

// ResolveInternal resolves and returns the object, value or memory at the path
// p without converting the potentially internal result to a service
// representation.
func ResolveInternal(ctx context.Context, p path.Node, r *path.ResolveConfig) (interface{}, error) {
	ctx = status.Start(ctx, "Resolve<%v>", p)
	defer status.Finish(ctx)

	switch p := p.(type) {
	case *path.ArrayIndex:
		return ArrayIndex(ctx, p, r)
	case *path.As:
		return As(ctx, p, r)
	case *path.Blob:
		return Blob(ctx, p, r)
	case *path.Capture:
		return Capture(ctx, p, r)
	case *path.Command:
		return Cmd(ctx, p, r)
	case *path.Commands:
		return Commands(ctx, p, r)
	case *path.CommandTree:
		return CommandTree(ctx, p, r)
	case *path.CommandTreeNode:
		return CommandTreeNode(ctx, p, r)
	case *path.CommandTreeNodeForCommand:
		return CommandTreeNodeForCommand(ctx, p, r)
	case *path.ConstantSet:
		return ConstantSet(ctx, p, r)
	case *path.Device:
		return Device(ctx, p, r)
	case *path.DeviceTraceConfiguration:
		return DeviceTraceConfiguration(ctx, p, r)
	case *path.Events:
		return Events(ctx, p, r)
	case *path.FramebufferObservation:
		return FramebufferObservation(ctx, p, r)
	case *path.FramebufferAttachments:
		return FramebufferAttachments(ctx, p, r)
	case *path.FramebufferAttachment:
		return FramebufferAttachment(ctx, p, r)
	case *path.Field:
		return Field(ctx, p, r)
	case *path.Framegraph:
		return Framegraph(ctx, p, r)
	case *path.GlobalState:
		return GlobalState(ctx, p, r)
	case *path.ImageInfo:
		return ImageInfo(ctx, p, r)
	case *path.MapIndex:
		return MapIndex(ctx, p, r)
	case *path.Memory:
		return Memory(ctx, p, r)
	case *path.MemoryAsType:
		return MemoryAsType(ctx, p, r)
	case *path.Metrics:
		return Metrics(ctx, p, r)
	case *path.Mesh:
		return Mesh(ctx, p, r)
	case *path.Messages:
		return Messages(ctx, p)
	case *path.Parameter:
		return Parameter(ctx, p, r)
	case *path.Pipelines:
		return Pipelines(ctx, p, r)
	case *path.Report:
		return Report(ctx, p, r)
	case *path.ResourceData:
		return ResourceData(ctx, p, r)
	case *path.Resources:
		return Resources(ctx, p.Capture, r)
	case *path.Result:
		return Result(ctx, p, r)
	case *path.Slice:
		return Slice(ctx, p, r)
	case *path.State:
		return State(ctx, p, r)
	case *path.StateTree:
		return StateTree(ctx, p, r)
	case *path.StateTreeNode:
		return StateTreeNode(ctx, p, r)
	case *path.StateTreeNodeForPath:
		return StateTreeNodeForPath(ctx, p, r)
	case *path.Thumbnail:
		return Thumbnail(ctx, p, r)
	case *path.Stats:
		return Stats(ctx, p, r)
	case *path.Type:
		return Type(ctx, p, r)
	default:
		return nil, fmt.Errorf("Unknown path type %T", p)
	}
}

func typename(t reflect.Type) string {
	if s := t.Name(); len(s) > 0 {
		return s
	}
	switch t.Kind() {
	case reflect.Ptr:
		return "ptr<" + typename(t.Elem()) + ">"
		// TODO: Format other composite types?
	default:
		return t.String()
	}
}

func convert(val reflect.Value, ty reflect.Type) (reflect.Value, bool) {
	if !val.IsValid() {
		return reflect.Zero(ty), true
	}
	valTy := val.Type()
	if valTy == ty {
		return val, true
	}
	if valTy.ConvertibleTo(ty) {
		return val.Convert(ty), true
	}
	// slice -> array
	if valTy.Kind() == reflect.Slice && ty.Kind() == reflect.Array {
		if valTy.Elem().ConvertibleTo(ty.Elem()) {
			c := sint.Min(val.Len(), ty.Len())
			out := reflect.New(ty).Elem()
			for i := 0; i < c; i++ {
				v, ok := convert(val.Index(i), ty.Elem())
				if !ok {
					return val, false
				}
				out.Index(i).Set(v)
			}
			return out, true
		}
	}
	return val, false
}

// SetupContext binds the capture and a replay device to the returned context.
func SetupContext(ctx context.Context, c *path.Capture, r *path.ResolveConfig) context.Context {
	if c != nil {
		ctx = capture.Put(ctx, c)
	}

	if d := r.GetReplayDevice(); d != nil {
		ctx = replay.PutDevice(ctx, d)
	} else {
		registry := bind.GetRegistry(ctx)
		if d := registry.DefaultDevice(); d != nil {
			ctx = replay.PutDevice(ctx, path.NewDevice(d.Instance().ID.ID()))
		}
	}

	return ctx
}
