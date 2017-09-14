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
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

func internalToService(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case api.Cmd:
		return api.CmdToService(v)
	case []*api.ContextInfo:
		out := &service.Contexts{List: make([]*path.Context, len(v))}
		for i, c := range v {
			out.List[i] = c.Path
		}
		return out, nil
	case *api.ContextInfo:
		return &service.Context{
			Name:     v.Name,
			Api:      path.NewAPI(id.ID(v.API)),
			Priority: uint32(v.Priority),
		}, nil
	default:
		return v, nil
	}
}

func serviceToInternal(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case *api.Command:
		return api.ServiceToCmd(v)
	case *box.Value:
		return v.Get(), nil
	default:
		return v, nil
	}
}
