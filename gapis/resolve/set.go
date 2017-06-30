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

	"github.com/google/gapid/core/data/deep"
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

	v, err := serviceToInternal(r.Value.Get())
	if err != nil {
		return nil, err
	}

	p, err := change(ctx, r.Path.Node(), v)
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

		atomIdx := p.After.Indices[0]
		if len(p.After.Indices) > 1 {
			return nil, fmt.Errorf("Subcommands currently not supported for changing") // TODO: Subcommands
		}

		oldList, err := NAtoms(ctx, p.After.Capture, atomIdx+1)
		if err != nil {
			return nil, err
		}

		list := oldList.Clone()
		replaceAtoms := func(where uint64, with interface{}) {
			list.Atoms[where] = with.(atom.Atom)
		}

		data, ok := val.(*gfxapi.ResourceData)
		if !ok {
			return nil, fmt.Errorf("Expected ResourceData, got %T", val)
		}

		if err := meta.Resource.SetResourceData(ctx, p.After, data, meta.IDMap, replaceAtoms); err != nil {
			return nil, err
		}

		// Store the new atom list
		c, err := changeAtoms(ctx, p.After.Capture, list.Atoms)
		if err != nil {
			return nil, err
		}

		return &path.ResourceData{
			Id: p.Id, // TODO: Shouldn't this change?
			After: &path.Command{
				Capture: c,
				Indices: p.After.Indices,
			},
		}, nil

	case *path.Command:
		atomIdx := p.Indices[0]
		if len(p.Indices) > 1 {
			return nil, fmt.Errorf("Subcommands currently not supported, for changing") // TODO: Subcommands
		}

		// Resolve the command list
		oldList, err := NAtoms(ctx, p.Capture, atomIdx+1)
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
		oldAtom := oldList.Atoms[atomIdx]
		if len(atom.Extras().All()) == 0 {
			atom.Extras().Add(oldAtom.Extras().All()...)
		}
		list.Atoms[atomIdx] = atom

		// Store the new atom list
		c, err := changeAtoms(ctx, p.Capture, list.Atoms)
		if err != nil {
			return nil, err
		}

		return &path.Command{
			Capture: c,
			Indices: p.Indices,
		}, nil

	case *path.Commands:
		return nil, fmt.Errorf("Commands can not be changed directly")

	case *path.State:
		return nil, fmt.Errorf("State can not currently be mutated")

	case *path.Field, *path.Parameter, *path.ArrayIndex, *path.MapIndex:
		oldObj, err := ResolveInternal(ctx, p.Parent())
		if err != nil {
			return nil, err
		}

		obj, err := clone(reflect.ValueOf(oldObj))
		if err != nil {
			return nil, err
		}

		switch p := p.(type) {
		case *path.Parameter:
			// TODO: Deal with parameters belonging to sub-commands.
			a := obj.Interface().(atom.Atom)
			err := atom.SetParameter(ctx, a, p.Name, val)
			switch err {
			case nil:
			case atom.ErrParameterNotFound:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrParameterDoesNotExist(a.AtomName(), p.Name),
					Path:   p.Path(),
				}
			default:
				return nil, err
			}

			parent, err := change(ctx, p.Parent(), obj.Interface())
			if err != nil {
				return nil, err
			}
			return parent.(*path.Command).Parameter(p.Name), nil

		case *path.Result:
			// TODO: Deal with parameters belonging to sub-commands.
			a := obj.Interface().(atom.Atom)
			err := atom.SetResult(ctx, a, val)
			switch err {
			case nil:
			case atom.ErrResultNotFound:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrResultDoesNotExist(a.AtomName()),
					Path:   p.Path(),
				}
			default:
				return nil, err
			}

			parent, err := change(ctx, p.Parent(), obj.Interface())
			if err != nil {
				return nil, err
			}
			return parent.(*path.Command).Result(), nil

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
			if count := uint64(a.Len()); p.Index >= count {
				return nil, errPathOOB(p.Index, "Index", 0, count-1, p)
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

func changeAtoms(ctx context.Context, p *path.Capture, newAtoms []atom.Atom) (*path.Capture, error) {
	old, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	c, err := capture.New(ctx, old.Name+"*", old.Header, newAtoms)
	if err != nil {
		return nil, err
	}
	return c, nil
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

	return deep.Copy(dst.Addr().Interface(), src.Interface())
}
