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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// FramebufferObservation returns the framebuffer observation for the given
// command.
func FramebufferObservation(ctx context.Context, p *path.FramebufferObservation) (*image.Info, error) {
	obj, err := database.Build(ctx, &FramebufferObservationResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(*image.Info), nil
}

// Resolve implements the database.Resolver interface.
func (r *FramebufferObservationResolvable) Resolve(ctx context.Context) (interface{}, error) {
	cmd, err := Cmd(ctx, r.Path.Command)
	if err != nil {
		return nil, err
	}
	for _, e := range cmd.Extras().All() {
		if o, ok := e.(*capture.FramebufferObservation); ok {
			data, err := database.Store(ctx, o.Data)
			if err != nil {
				return nil, err
			}
			return &image.Info{
				Format: image.RGBA_U8_NORM,
				Width:  o.DataWidth,
				Height: o.DataHeight,
				Depth:  1,
				Bytes:  image.NewID(data),
			}, nil
		}
	}
	return nil, fmt.Errorf("%v does not contain a framebuffer observation", r.Path.Command)
}
