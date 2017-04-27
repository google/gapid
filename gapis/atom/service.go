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

package atom

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// ErrParameterNotFound is the error returned by SetParameter when the atom
// does not have the named parameter.
const ErrParameterNotFound = fault.Const("Parameter not found")

// ToService returns the service command representing atom a.
func ToService(a Atom) (*service.Command, error) {
	out := &service.Command{Name: a.AtomName()}

	if api := a.API(); api != nil {
		out.Api = &path.API{Id: path.NewID(id.ID(api.ID()))}
	}

	v := reflect.ValueOf(a)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		v, t := v.Field(i), t.Field(i)
		if name, ok := t.Tag.Lookup("param"); ok {
			param := &service.Parameter{
				Name:  name,
				Value: box.NewValue(v.Interface()),
			}

			if cs, ok := t.Tag.Lookup("constset"); ok {
				if idx, _ := strconv.Atoi(cs); idx > 0 {
					param.Constants = out.Api.ConstantSet(idx)
				}
			}

			out.Parameters = append(out.Parameters, param)
		}
	}

	return out, nil
}

// ToAtom returns the service command representing atom a.
func ToAtom(c *service.Command) (Atom, error) {
	api := gfxapi.Find(gfxapi.ID(c.GetApi().GetId().ID()))
	if api == nil {
		return nil, fmt.Errorf("Unknown api '%v'", c.GetApi())
	}
	a := Create(api, c.Name)
	if a == nil {
		return nil, fmt.Errorf("Unknown command '%v.%v'", api.Name(), c.Name)
	}

	v := reflect.ValueOf(a)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup("param"); ok {
			p := c.FindParameter(n)
			if p == nil {
				continue
			}
			if err := p.Value.AssignTo(f.Addr().Interface()); err != nil {
				return nil, err
			}
		}
	}

	return a, nil
}

// Parameter returns the parameter value with the specified name.
func Parameter(ctx context.Context, a Atom, name string) (interface{}, error) {
	v := reflect.ValueOf(a)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup("param"); ok {
			if name == n {
				return f.Interface(), nil
			}
		}
	}
	return nil, ErrParameterNotFound
}

// SetParameter sets the parameter with the specified name with val.
func SetParameter(ctx context.Context, a Atom, name string, val interface{}) error {
	v := reflect.ValueOf(a)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup("param"); ok {
			if name == n {
				return deep.Copy(f.Addr().Interface(), val)
			}
		}
	}
	return ErrParameterNotFound
}
