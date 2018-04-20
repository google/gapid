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

	for _, p := range c.CmdParams() {
		param := &Parameter{
			Name:  p.Name,
			Value: box.NewValue(p.Get()),
		}
		if p.Constants > 0 {
			param.Constants = out.Api.ConstantSet(p.Constants)
		}
		out.Parameters = append(out.Parameters, param)
	}

	if p := c.CmdResult(); p != nil {
		out.Result = &Parameter{
			Name:  p.Name,
			Value: box.NewValue(p.Get()),
		}
		if p.Constants >= 0 {
			out.Result.Constants = out.Api.ConstantSet(p.Constants)
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

	for _, s := range c.Parameters {
		SetParameter(a, s.Name, s.Value.Get())
	}

	if p := a.CmdResult(); p != nil && c.Result != nil {
		a.CmdResult().Set(c.Result.Value.Get())
	}

	return a, nil
}
