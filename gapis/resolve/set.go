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
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Set creates a copy of the capture referenced by the request's path, but
// with the object, value or memory at p replaced with v. The path returned is
// identical to p, but with the base changed to refer to the new capture.
func Set(ctx context.Context, p *path.Any, v interface{}, r *path.ResolveConfig) (*path.Any, error) {
	obj, err := database.Build(ctx, &SetResolvable{Path: p, Value: service.NewValue(v), Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*path.Any), nil
}

// Resolve implements the database.Resolver interface.
func (r *SetResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, path.FindCapture(r.Path.Node()), r.Config)

	v, err := serviceToInternal(r.Value.Get())
	if err != nil {
		return nil, err
	}

	p, err := change(ctx, r.Path.Node(), v, r.Config)
	if err != nil {
		return nil, err
	}
	return p.Path(), nil
}

func change(ctx context.Context, p path.Node, val interface{}, r *path.ResolveConfig) (path.Node, error) {
	switch p := p.(type) {
	case *path.Report:
		return nil, fmt.Errorf("Reports are immutable")

	case *path.MultiResourceData:
		data, ok := val.(*api.MultiResourceData)
		if !ok {
			return nil, fmt.Errorf("Expected ResourceData, got %T", val)
		}

		c, err := changeResources(ctx, p.After, p.IDs, data.Resources, r)
		if err != nil {
			return nil, err
		}

		return &path.MultiResourceData{
			IDs: p.IDs, // TODO: Shouldn't this change?
			After: &path.Command{
				Capture: c,
				Indices: p.After.Indices,
			},
		}, nil

	case *path.ResourceData:
		data, ok := val.(*api.ResourceData)
		if !ok {
			return nil, fmt.Errorf("Expected ResourceData, got %T", val)
		}

		c, err := changeResources(ctx, p.After, []*path.ID{p.ID}, []*api.ResourceData{data}, r)
		if err != nil {
			return nil, err
		}

		return &path.ResourceData{
			ID: p.ID, // TODO: Shouldn't this change?
			After: &path.Command{
				Capture: c,
				Indices: p.After.Indices,
			},
		}, nil

	case *path.Command:
		cmdIdx := p.Indices[0]
		if len(p.Indices) > 1 {
			return nil, fmt.Errorf("Cannot modify subcommands") // TODO: Subcommands
		}

		// Resolve the command list
		oldCmds, err := NCmds(ctx, p.Capture, cmdIdx+1)
		if err != nil {
			return nil, err
		}

		// Validate the value
		if val == nil {
			return nil, fmt.Errorf("Command cannot be nil")
		}
		cmd, ok := val.(api.Cmd)
		if !ok {
			return nil, fmt.Errorf("Expected Cmd, got %T", val)
		}

		// Clone the command list
		cmds := make([]api.Cmd, len(oldCmds))
		copy(cmds, oldCmds)

		// Propagate extras if the new command omitted them
		oldCmd := oldCmds[cmdIdx]
		if len(cmd.Extras().All()) == 0 {
			cmd.Extras().Add(oldCmd.Extras().All()...)
		}

		// Replace the command
		cmds[cmdIdx] = cmd

		// Store the new command list
		c, err := changeCommands(ctx, p.Capture, cmds)
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
		oldObj, err := ResolveInternal(ctx, p.Parent(), r)
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
			cmd := obj.Interface().(api.Cmd)
			err := api.SetParameter(cmd, p.Name, val)
			switch err {
			case nil:
			case api.ErrParameterNotFound:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrParameterDoesNotExist(cmd.CmdName(), p.Name),
					Path:   p.Path(),
				}
			default:
				return nil, err
			}

			parent, err := change(ctx, p.Parent(), obj.Interface(), r)
			if err != nil {
				return nil, err
			}
			return parent.(*path.Command).Parameter(p.Name), nil

		case *path.Result:
			// TODO: Deal with parameters belonging to sub-commands.
			cmd := obj.Interface().(api.Cmd)
			err := api.SetResult(cmd, val)
			switch err {
			case nil:
			case api.ErrResultNotFound:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrResultDoesNotExist(cmd.CmdName()),
					Path:   p.Path(),
				}
			default:
				return nil, err
			}

			parent, err := change(ctx, p.Parent(), obj.Interface(), r)
			if err != nil {
				return nil, err
			}
			return parent.(*path.Command).Result(), nil

		case *path.Field:
			parent, err := setField(ctx, obj, reflect.ValueOf(val), p.Name, p, r)
			if err != nil {
				return nil, err
			}
			out := &path.Field{Name: p.Name}
			out.SetParent(parent)
			return out, nil

		case *path.ArrayIndex:
			arr, ty := obj, obj.Type()
			switch arr.Kind() {
			case reflect.Array, reflect.Slice:
				ty = ty.Elem()
			case reflect.String:
			default:
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrTypeNotArrayIndexable(typename(arr.Type())),
					Path:   p.Path(),
				}
			}
			val, ok := convert(reflect.ValueOf(val), ty)
			if !ok {
				return nil, fmt.Errorf("Slice or array at %s has element of type %v, got type %v",
					p.Parent(), ty, val.Type())
			}
			if count := uint64(arr.Len()); p.Index >= count {
				return nil, errPathOOB(p.Index, "Index", 0, count-1, p)
			}
			if err := assign(arr.Index(int(p.Index)), val); err != nil {
				return nil, err
			}
			parent, err := change(ctx, p.Parent(), arr.Interface(), r)
			if err != nil {
				return nil, err
			}
			p = &path.ArrayIndex{Index: p.Index}
			p.SetParent(parent)
			return p, nil

		case *path.MapIndex:
			d := dictionary.From(obj.Interface())
			if d == nil {
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrTypeNotMapIndexable(typename(obj.Type())),
					Path:   p.Path(),
				}
			}

			keyTy, valTy := d.KeyTy(), d.ValTy()

			key, ok := convert(reflect.ValueOf(p.KeyValue()), keyTy)
			if !ok {
				return nil, &service.ErrInvalidPath{
					Reason: messages.ErrIncorrectMapKeyType(
						typename(reflect.TypeOf(p.KeyValue())), // got
						typename(keyTy)),                       // expected
					Path: p.Path(),
				}
			}

			val, ok := convert(reflect.ValueOf(val), d.ValTy())
			if !ok {
				return nil, fmt.Errorf("Map at %s has value of type %v, got type %v",
					p.Parent(), valTy, val.Type())
			}

			d.Add(key.Interface(), val.Interface())

			parent, err := change(ctx, p.Parent(), obj.Interface(), r)
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

func changeResources(ctx context.Context, after *path.Command, ids []*path.ID, data []*api.ResourceData, r *path.ResolveConfig) (*path.Capture, error) {
	meta, err := ResourceMeta(ctx, ids, after, r)
	if err != nil {
		return nil, err
	}
	if len(meta.Resources) != len(ids) {
		return nil, fmt.Errorf("Expected %d resource(s), got %d", len(ids), len(meta.Resources))
	}

	cmdIdx := after.Indices[0]
	// If we change resource data, subcommands do not affect this, so change
	// the main command.

	oldCmds, err := NCmds(ctx, after.Capture, cmdIdx+1)
	if err != nil {
		return nil, err
	}

	cmds := make([]api.Cmd, len(oldCmds))
	copy(cmds, oldCmds)

	replaceCommands := func(where uint64, with interface{}) {
		cmds[where] = with.(api.Cmd)
	}

	oldCapt, err := capture.ResolveGraphicsFromPath(ctx, after.Capture)
	if err != nil {
		return nil, err
	}
	var initialState *capture.InitialState
	mutateInitialState := func(API api.API) api.State {
		if initialState == nil {
			if initialState = oldCapt.CloneInitialState(); initialState == nil {
				return nil
			}
		}
		return initialState.APIs[API]
	}

	for i, resource := range meta.Resources {
		if err := resource.SetResourceData(
			ctx,
			after,
			data[i],
			meta.IDMap,
			replaceCommands,
			mutateInitialState,
			r); err != nil {
			return nil, err
		}
	}

	if initialState == nil {
		initialState = oldCapt.InitialState
	}

	gc, err := capture.NewGraphicsCapture(ctx, oldCapt.Name()+"*", oldCapt.Header, initialState, cmds)
	if err != nil {
		return nil, err
	} else {
		return capture.New(ctx, gc)
	}
}

func changeCommands(ctx context.Context, p *path.Capture, newCmds []api.Cmd) (*path.Capture, error) {
	old, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	c, err := capture.NewGraphicsCapture(ctx, old.Name()+"*", old.Header, old.InitialState, newCmds)
	if err != nil {
		return nil, err
	}
	return capture.New(ctx, c)
}

func setField(
	ctx context.Context,
	str,
	val reflect.Value,
	name string,
	p path.Node,
	r *path.ResolveConfig) (path.Node, error) {

	dst, err := field(ctx, str, name, p)
	if err != nil {
		return nil, err
	}
	if err := assign(dst, val); err != nil {
		return nil, err
	}
	return change(ctx, p.Parent(), str.Interface(), r)
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
