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

package api

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// CmdToService returns the service command representing command c.
func CmdToService(c Cmd) (*Command, error) {
	out := &Command{
		Name:   c.CmdName(),
		Thread: c.Thread(),
	}

	if api := c.API(); api != nil {
		out.Api = &path.API{Id: path.NewID(id.ID(api.ID()))}
	}

	v := reflect.ValueOf(c)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		v, t := v.Field(i), t.Field(i)
		if name, ok := t.Tag.Lookup(paramTag); ok {
			param := &Parameter{
				Name:  name,
				Value: box.NewValue(v.Interface()),
			}

			if cs, ok := t.Tag.Lookup(constsetTag); ok {
				if idx, _ := strconv.Atoi(cs); idx > 0 {
					param.Constants = out.Api.ConstantSet(idx)
				}
			}

			out.Parameters = append(out.Parameters, param)
		}
		if _, ok := t.Tag.Lookup(resultTag); ok {
			out.Result = &Parameter{Value: box.NewValue(v.Interface())}
			if cs, ok := t.Tag.Lookup(constsetTag); ok {
				if idx, _ := strconv.Atoi(cs); idx > 0 {
					out.Result.Constants = out.Api.ConstantSet(idx)
				}
			}
		}
	}

	return out, nil
}

// ServiceToCmd returns the command built from c.
func ServiceToCmd(c *Command) (Cmd, error) {
	api := Find(ID(c.GetApi().GetId().ID()))
	if api == nil {
		return nil, fmt.Errorf("Unknown api '%v'", c.GetApi())
	}
	a := api.CreateCmd(c.Name)
	if a == nil {
		return nil, fmt.Errorf("Unknown command '%v.%v'", api.Name(), c.Name)
	}
	a.SetThread(c.Thread)

	v := reflect.ValueOf(a)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup(paramTag); ok {
			p := c.FindParameter(n)
			if p == nil {
				continue
			}
			if err := p.Value.AssignTo(f.Addr().Interface()); err != nil {
				return nil, err
			}
		}
		if _, ok := t.Tag.Lookup(resultTag); ok {
			p := c.Result
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

// FindParameter returns the parameter with the given name, or nil if no
// parameter is found.
func (c *Command) FindParameter(name string) *Parameter {
	for _, p := range c.Parameters {
		if p.Name == name {
			return p
		}
	}
	return nil
}
