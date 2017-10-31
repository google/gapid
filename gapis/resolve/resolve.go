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

	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// Capture resolves and returns the capture from the path p.
func Capture(ctx context.Context, p *path.Capture) (*service.Capture, error) {
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.Service(ctx, p), nil
}

// Device resolves and returns the device from the path p.
func Device(ctx context.Context, p *path.Device) (*device.Instance, error) {
	device := bind.GetRegistry(ctx).Device(p.Id.ID())
	if device == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrUnknownDevice()}
	}
	return device.Instance(), nil
}

// ImageInfo resolves and returns the ImageInfo from the path p.
func ImageInfo(ctx context.Context, p *path.ImageInfo) (*image.Info, error) {
	obj, err := database.Resolve(ctx, p.Id.ID())
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
func Blob(ctx context.Context, p *path.Blob) ([]byte, error) {
	obj, err := database.Resolve(ctx, p.Id.ID())
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
func Field(ctx context.Context, p *path.Field) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	v, err := field(ctx, reflect.ValueOf(obj), p.Name, p)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

func field(ctx context.Context, s reflect.Value, name string, p path.Node) (reflect.Value, error) {
	for {
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
			if s.IsNil() {
				return reflect.Value{}, &service.ErrInvalidPath{
					Reason: messages.ErrNilPointerDereference(),
					Path:   p.Path(),
				}
			}
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
func ArrayIndex(ctx context.Context, p *path.ArrayIndex) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}

	a := reflect.ValueOf(obj)
	switch {
	case box.IsMemorySlice(a.Type()):
		ml, err := memoryLayout(ctx, p)
		if err != nil {
			return nil, err
		}

		slice := box.AsMemorySlice(a)
		if count := slice.Count(); p.Index >= count {
			return nil, errPathOOB(p.Index, "Index", 0, count-1, p)
		}
		return slice.IIndex(p.Index, ml), nil

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
func Slice(ctx context.Context, p *path.Slice) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	a := reflect.ValueOf(obj)
	switch {
	case box.IsMemorySlice(a.Type()):
		ml, err := memoryLayout(ctx, p)
		if err != nil {
			return nil, err
		}

		slice := box.AsMemorySlice(a)
		if p.Start >= slice.Count() || p.End > slice.Count() {
			return nil, errPathSliceOOB(p.Start, p.End, slice.Count(), p)
		}
		return slice.ISlice(p.Start, p.End, ml), nil

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
func MapIndex(ctx context.Context, p *path.MapIndex) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
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

	c, err := capture.ResolveFromPath(ctx, cp)
	if err != nil {
		return nil, err
	}

	return c.Header.Abi.MemoryLayout, nil
}

// ResolveService resolves and returns the object, value or memory at the path p,
// converting the final result to the service representation.
func ResolveService(ctx context.Context, p path.Node) (interface{}, error) {
	v, err := ResolveInternal(ctx, p)
	if err != nil {
		return nil, err
	}
	return internalToService(v)
}

// ResolveInternal resolves and returns the object, value or memory at the path
// p without converting the potentially internal result to a service
// representation.
func ResolveInternal(ctx context.Context, p path.Node) (interface{}, error) {
	switch p := p.(type) {
	case *path.ArrayIndex:
		return ArrayIndex(ctx, p)
	case *path.As:
		return As(ctx, p)
	case *path.Blob:
		return Blob(ctx, p)
	case *path.Capture:
		return Capture(ctx, p)
	case *path.Command:
		return Cmd(ctx, p)
	case *path.Commands:
		return Commands(ctx, p)
	case *path.CommandTree:
		return CommandTree(ctx, p)
	case *path.CommandTreeNode:
		return CommandTreeNode(ctx, p)
	case *path.CommandTreeNodeForCommand:
		return CommandTreeNodeForCommand(ctx, p)
	case *path.ConstantSet:
		return ConstantSet(ctx, p)
	case *path.Context:
		return Context(ctx, p)
	case *path.Contexts:
		return Contexts(ctx, p)
	case *path.Device:
		return Device(ctx, p)
	case *path.Events:
		return Events(ctx, p)
	case *path.FramebufferObservation:
		return FramebufferObservation(ctx, p)
	case *path.Field:
		return Field(ctx, p)
	case *path.GlobalState:
		return GlobalState(ctx, p)
	case *path.ImageInfo:
		return ImageInfo(ctx, p)
	case *path.MapIndex:
		return MapIndex(ctx, p)
	case *path.Memory:
		return Memory(ctx, p)
	case *path.Mesh:
		return Mesh(ctx, p)
	case *path.Parameter:
		return Parameter(ctx, p)
	case *path.Report:
		return Report(ctx, p)
	case *path.ResourceData:
		return ResourceData(ctx, p)
	case *path.Resources:
		return Resources(ctx, p.Capture)
	case *path.Result:
		return Result(ctx, p)
	case *path.Slice:
		return Slice(ctx, p)
	case *path.State:
		return State(ctx, p)
	case *path.StateTree:
		return StateTree(ctx, p)
	case *path.StateTreeNode:
		return StateTreeNode(ctx, p)
	case *path.StateTreeNodeForPath:
		return StateTreeNodeForPath(ctx, p)
	case *path.Thumbnail:
		return Thumbnail(ctx, p)
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
