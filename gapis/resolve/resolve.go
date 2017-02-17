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

// Package resolve exposes functions for performing complex data queries.
package resolve

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Capture resolves and returns the capture from the path p.
func Capture(ctx log.Context, p *path.Capture) (*service.Capture, error) {
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.Service(ctx, p), nil
}

// Commands resolves and returns the atom list from the path p.
func Commands(ctx log.Context, p *path.Commands) (*atom.List, error) {
	c, err := capture.ResolveFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	return c.Atoms(ctx)
}

// NCommands resolves and returns the atom list from the path p, ensuring
// that the number of commands is at least N.
func NCommands(ctx log.Context, p *path.Commands, n uint64) (*atom.List, error) {
	list, err := Commands(ctx, p)
	if err != nil {
		return nil, err
	}
	if count := uint64(len(list.Atoms)); count < n {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(n-1, "Index", uint64(0), count-1),
			Path:   p.Index(n - 1).Path(),
		}
	}
	return list, nil
}

// Command resolves and returns the atom from the path p.
func Command(ctx log.Context, p *path.Command) (atom.Atom, error) {
	list, err := NCommands(ctx, p.Commands, p.Index+1)
	if err != nil {
		return nil, err
	}
	return list.Atoms[p.Index], nil
}

// Device resolves and returns the device from the path p.
func Device(ctx log.Context, p *path.Device) (*device.Instance, error) {
	device := bind.GetRegistry(ctx).Device(p.Id.ID())
	if device == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrUnknownDevice()}
	}
	return device.Instance(), nil
}

// ImageInfo resolves and returns the ImageInfo from the path p.
func ImageInfo(ctx log.Context, p *path.ImageInfo) (*image.Info2D, error) {
	obj, err := database.Resolve(ctx, p.Id.ID())
	if err != nil {
		return nil, err
	}
	ii, ok := obj.(*image.Info2D)
	if !ok {
		return nil, fmt.Errorf("Path %s gave %T, expected *image.Info2D", p, obj)
	}
	return ii, err
}

// Blob resolves and returns the byte slice from the path p.
func Blob(ctx log.Context, p *path.Blob) ([]byte, error) {
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

// Mesh resolves and returns the Mesh from the path p.
func Mesh(ctx log.Context, p *path.Mesh) (*gfxapi.Mesh, error) {
	obj, err := Resolve(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	if ao, ok := obj.(gfxapi.APIObject); ok {
		if api := ao.API(); api != nil {
			if m, ok := api.(gfxapi.MeshProvider); ok {
				return m.Mesh(ctx, obj, p)
			}
		}
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshNotAvailable()}
}

// Parameter resolves and returns the parameter from the path p.
func Parameter(ctx log.Context, p *path.Parameter) (interface{}, error) {
	cmd, err := Resolve(ctx, p.Command)
	if err != nil {
		return nil, err
	}
	v, err := field(ctx, reflect.ValueOf(cmd), p.Name, p)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

// Field resolves and returns the field from the path p.
func Field(ctx log.Context, p *path.Field) (interface{}, error) {
	obj, err := Resolve(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	v, err := field(ctx, reflect.ValueOf(obj), p.Name, p)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

func field(ctx log.Context, s reflect.Value, name string, p path.Node) (reflect.Value, error) {
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
func ArrayIndex(ctx log.Context, p *path.ArrayIndex) (interface{}, error) {
	obj, err := Resolve(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	a := reflect.ValueOf(obj)
	switch a.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		if int(p.Index) >= a.Len() {
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrValueOutOfBounds(p.Index, "Index", uint64(0), uint64(a.Len()-1)),
				Path:   p.Path(),
			}
		}
		return a.Index(int(p.Index)).Interface(), nil

	default:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable(typename(a.Type())),
			Path:   p.Path(),
		}
	}
}

// Slice resolves and returns the subslice from the path p.
func Slice(ctx log.Context, p *path.Slice) (interface{}, error) {
	obj, err := Resolve(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	a := reflect.ValueOf(obj)
	switch a.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		if int(p.Start) >= a.Len() || int(p.End) > a.Len() {
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrSliceOutOfBounds(p.Start, p.End, "Start", "End", uint64(0), uint64(a.Len()-1)),
				Path:   p.Path(),
			}
		}
		return a.Slice(int(p.Start), int(p.End)).Interface(), nil

	default:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotSliceable(typename(a.Type())),
			Path:   p.Path(),
		}
	}
}

// MapIndex resolves and returns the map value from the path p.
func MapIndex(ctx log.Context, p *path.MapIndex) (interface{}, error) {
	obj, err := Resolve(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	m := reflect.ValueOf(obj)
	switch m.Kind() {
	case reflect.Map:
		key, ok := convertMapKey(reflect.ValueOf(p.KeyValue()), m.Type().Key())
		if !ok {
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrIncorrectMapKeyType(
					typename(reflect.TypeOf(p.KeyValue())), // got
					typename(m.Type().Key())),              // expected
				Path: p.Path(),
			}
		}
		val := m.MapIndex(key)
		if !val.IsValid() {
			return nil, &service.ErrInvalidPath{
				Reason: messages.ErrMapKeyDoesNotExist(key.Interface()),
				Path:   p.Path(),
			}
		}
		return val.Interface(), nil

	default:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable(typename(m.Type())),
			Path:   p.Path(),
		}
	}
}

// Resolve resolves and returns the object, value or memory at the path p.
func Resolve(ctx log.Context, p path.Node) (interface{}, error) {
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
		return Command(ctx, p)
	case *path.Commands:
		return Commands(ctx, p)
	case *path.Context:
		return Context(ctx, p)
	case *path.Contexts:
		return Contexts(ctx, p)
	case *path.Device:
		return Device(ctx, p)
	case *path.Field:
		return Field(ctx, p)
	case *path.Hierarchies:
		return Hierarchies(ctx, p)
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
		return Report(ctx, p.Capture, p.Device)
	case *path.ResourceData:
		return ResourceData(ctx, p)
	case *path.Resources:
		return Resources(ctx, p.Capture)
	case *path.Slice:
		return Slice(ctx, p)
	case *path.State:
		return APIState(ctx, p)
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

func convertMapKey(val reflect.Value, ty reflect.Type) (reflect.Value, bool) {
	valTy := val.Type()
	if valTy == ty {
		return val, true
	}
	if valTy.ConvertibleTo(ty) {
		return val.Convert(ty), true
	}
	return val, false
}

func convert(val reflect.Value, ty reflect.Type) (reflect.Value, bool) {
	if !val.IsValid() {
		return reflect.Zero(ty), true
	}
	if valTy := val.Type(); valTy != ty {
		if valTy.ConvertibleTo(ty) {
			val = val.Convert(ty)
		} else {
			return val, false
		}
	}
	return val, true
}
