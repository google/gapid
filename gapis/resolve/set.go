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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Set creates a copy of the capture referenced by the request's path, but
// with the object, value or memory at p replaced with v. The path returned is
// identical to p, but with the base changed to refer to the new capture.
func Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error) {
	obj, err := database.Build(ctx, &SetResolvable{p, service.NewValue(v)})
	if err != nil {
		return nil, err
	}
	return obj.(*path.Any), nil
}

// Resolve implements the database.Resolver interface.
func (r *SetResolvable) Resolve(ctx context.Context) (interface{}, error) {
	if c := path.FindCapture(r.Path.Node()); c != nil {
		ctx = capture.Put(ctx, c)
	}
	p, err := change(ctx, r.Path.Node(), r.Value.Get())
	if err != nil {
		return nil, err
	}
	return p.Path(), nil
}

func change(ctx context.Context, p path.Node, val interface{}) (path.Node, error) {
	switch p := p.(type) {
	case *path.Report:
		return nil, fmt.Errorf("Reports are immutable")

	case *path.ResourceData:
		meta, err := ResourceMeta(ctx, p.Id, p.After)
		if err != nil {
			return nil, err
		}

		oldList, err := NCommands(ctx, p.After.Commands, p.After.Index+1)
		if err != nil {
			return nil, err
		}

		list := oldList.Clone()
		replaceAtoms := func(where uint64, with gfxapi.ResourceAtom) {
			list.Atoms[where] = with.(atom.Atom)
		}

		if err := meta.Resource.SetResourceData(ctx, p.After, val, meta.IDMap, replaceAtoms); err != nil {
			return nil, err
		}
		commands, err := change(ctx, p.After.Commands, list)
		if err != nil {
			return nil, err
		}
		return &path.ResourceData{
			Id: p.Id, // TODO: Shouldn't this change?
			After: &path.Command{
				Commands: commands.(*path.Commands),
				Index:    p.After.Index,
			},
		}, nil

	case *path.Command:
		// Resolve the command list
		oldList, err := NCommands(ctx, p.Commands, p.Index+1)
		if err != nil {
			return nil, err
		}

		// Validate the value
		if val == nil {
			return nil, fmt.Errorf("Command cannot be nil")
		}
		atom, ok := val.(atom.Atom)
		if !ok {
			return nil, fmt.Errorf("Expected Atom, got %T", val)
		}

		// Clone the atom list
		list := oldList.Clone()

		// Propagate extras if the new atom omitted them
		oldAtom := oldList.Atoms[p.Index]
		if len(atom.Extras().All()) == 0 {
			atom.Extras().Add(oldAtom.Extras().All()...)
		}
		list.Atoms[p.Index] = atom

		// Store the new atom list
		commands, err := change(ctx, p.Commands, list)
		if err != nil {
			return nil, err
		}

		return &path.Command{
			Commands: commands.(*path.Commands),
			Index:    p.Index,
		}, nil

	case *path.Commands:
		old, err := capture.ResolveFromPath(ctx, p.Capture)
		if err != nil {
			return nil, err
		}
		atoms, ok := val.(*atom.List)
		if !ok {
			return nil, fmt.Errorf("Expected *atom.List, got %T", val)
		}
		c, err := capture.ImportAtomList(ctx, old.Name+"*", atoms, old.Header)
		if err != nil {
			return nil, err
		}
		return c.Commands(), nil

	case *path.State:
		return nil, fmt.Errorf("State can not currently be mutated")

	case *path.Field, *path.Parameter, *path.ArrayIndex, *path.MapIndex:
		oldObj, err := Resolve(ctx, p.Parent())
		if err != nil {
			return nil, err
		}

		obj, err := clone(reflect.ValueOf(oldObj))
		if err != nil {
			return nil, err
		}

		switch p := p.(type) {
		case *path.Parameter:
			parent, err := setField(ctx, obj, reflect.ValueOf(val), p.Name, p)
			if err != nil {
				return nil, err
			}
			return &path.Parameter{Name: p.Name, Command: parent.(*path.Command)}, nil

		case *path.Field:
			parent, err := setField(ctx, obj, reflect.ValueOf(val), p.Name, p)
			if err != nil {
				return nil, err
			}
			out := &path.Field{Name: p.Name}
			out.SetParent(parent)
			return out, nil

		case *path.ArrayIndex:
			a, ty := obj, obj.Type()
			switch a.Kind() {
			case reflect.Array, reflect.Slice:
				ty = ty.Elem()
			case reflect.String:
			default:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrTypeNotArrayIndexable(typename(a.Type())),
					Path:   p.Path(),
				}
			}
			val, ok := convert(reflect.ValueOf(val), ty)
			if !ok {
				return nil, fmt.Errorf("Slice or array at %s has element of type %v, got type %v",
					p.Parent().Text(), ty, val.Type())
			}
			if int(p.Index) >= a.Len() {
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrValueOutOfBounds(p.Index, "Index", uint64(0), uint64(a.Len()-1)),
					Path:   p.Path(),
				}
			}
			if err := assign(a.Index(int(p.Index)), val); err != nil {
				return nil, err
			}
			parent, err := change(ctx, p.Parent(), a.Interface())
			if err != nil {
				return nil, err
			}
			p = &path.ArrayIndex{Index: p.Index}
			p.SetParent(parent)
			return p, nil

		case *path.MapIndex:
			m := obj
			if m.Kind() != reflect.Map {
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrTypeNotMapIndexable(typename(m.Type())),
					Path:   p.Path(),
				}
			}
			key, ok := convertMapKey(reflect.ValueOf(p.KeyValue()), m.Type().Key())
			if !ok {
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrIncorrectMapKeyType(
						typename(reflect.TypeOf(p.KeyValue())), // got
						typename(m.Type().Key())),              // expected
					Path: p.Path(),
				}
			}
			val, ok := convert(reflect.ValueOf(val), m.Type().Elem())
			if !ok {
				return nil, fmt.Errorf("Map at %s has value of type %v, got type %v",
					p.Parent().Text(), m.Type().Elem(), val.Type())
			}
			m.SetMapIndex(key, val)
			parent, err := change(ctx, p.Parent(), m.Interface())
			if err != nil {
				return nil, err
			}
			p = &path.MapIndex{Key: p.Key}
			p.SetParent(parent)
			return p, nil
		}
	}
	return nil, fmt.Errorf("Unknown path type %T", p)
}

func setField(ctx context.Context, str, val reflect.Value, name string, p path.Node) (path.Node, error) {
	dst, err := field(ctx, str, name, p)
	if err != nil {
		return nil, err
	}
	if err := assign(dst, val); err != nil {
		return nil, err
	}
	return change(ctx, p.Parent(), str.Interface())
}

func clone(v reflect.Value) (reflect.Value, error) {
	var o reflect.Value
	switch v.Kind() {
	case reflect.Slice:
		o = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	case reflect.Map:
		o = reflect.MakeMap(v.Type())
	default:
		o = reflect.New(v.Type()).Elem()
	}
	return o, shallowCopy(o, v)
}

func shallowCopy(dst, src reflect.Value) error {
	switch dst.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !src.IsNil() {
			o := reflect.New(src.Elem().Type())
			shallowCopy(o.Elem(), src.Elem())
			dst.Set(o)
		}

	case reflect.Slice, reflect.Array:
		reflect.Copy(dst, src)

	case reflect.Map:
		for _, k := range src.MapKeys() {
			val := src.MapIndex(k)
			dst.SetMapIndex(k, val)
		}

	default:
		dst.Set(src)
	}
	return nil
}

func assign(dst, src reflect.Value) error {
	if !dst.CanSet() {
		return fmt.Errorf("Value is unassignable")
	}

	dstTy := dst.Type()

	var srcTy reflect.Type
	if !src.IsValid() {
		src = reflect.Zero(dstTy)
		srcTy = dstTy
	} else {
		srcTy = src.Type()
	}

	switch {
	case srcTy.AssignableTo(dstTy):
		dst.Set(src)
		return nil

	case srcTy.ConvertibleTo(dstTy):
		dst.Set(src.Convert(dstTy))
		return nil

	default:
		return fmt.Errorf("Cannot assign type %v to type %v", srcTy.Name(), dstTy.Name())
	}
}
