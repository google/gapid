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
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/service/types"
)

func internalToService(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case api.Cmd:
		return cmdToService(v)
	default:
		return v, nil
	}
}

func serviceToInternal(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case *api.Command:
		return serviceToCmd(v)
	case *box.Value:
		return v.Get(), nil
	default:
		return v, nil
	}
}

// cmdToService returns the service command representing command c.
func cmdToService(c api.Cmd) (*api.Command, error) {
	out := &api.Command{
		Name:       c.CmdName(),
		Thread:     c.Thread(),
		Terminated: c.Terminated(),
	}

	if a := c.API(); a != nil {
		out.API = &path.API{ID: path.NewID(id.ID(a.ID()))}
	}

	for _, p := range c.CmdParams() {
		t, err := types.GetTypeIndex(p.Get())
		if err != nil {
			return nil, err
		}
		param := &api.Parameter{
			Name:  p.Name,
			Value: box.NewValue(p.Get()),
			Type: &path.Type{
				TypeIndex: t,
				API:       out.API,
			},
		}
		if p.Constants > 0 {
			param.Constants = out.API.ConstantSet(p.Constants)
		}
		out.Parameters = append(out.Parameters, param)
	}

	if p := c.CmdResult(); p != nil {
		out.Result = &api.Parameter{
			Name:  p.Name,
			Value: box.NewValue(p.Get()),
		}
		if p.Constants >= 0 {
			out.Result.Constants = out.API.ConstantSet(p.Constants)
		}
	}

	return out, nil
}

// serviceToCmd returns the command built from c.
func serviceToCmd(c *api.Command) (api.Cmd, error) {
	a := api.Find(api.ID(c.GetAPI().GetID().ID()))
	if a == nil {
		return nil, fmt.Errorf("Unknown api '%v'", c.GetAPI())
	}
	cmd := a.CreateCmd(c.Name)
	if cmd == nil {
		return nil, fmt.Errorf("Unknown command '%v.%v'", a.Name(), c.Name)
	}

	cmd.SetThread(c.Thread)

	for _, s := range c.Parameters {
		api.SetParameter(cmd, s.Name, s.Value.Get())
	}

	if p := cmd.CmdResult(); p != nil && c.Result != nil {
		cmd.CmdResult().Set(c.Result.Value.Get())
	}

	return cmd, nil
}
