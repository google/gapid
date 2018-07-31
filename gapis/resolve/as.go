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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// As resolves and returns the object at p transformed to the requested type.
func As(ctx context.Context, p *path.As, r *path.ResolveConfig) (interface{}, error) {
	o, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}
	switch to := p.To.(type) {
	case *path.As_ImageFormat:
		switch o := o.(type) {
		case image.Convertable:
			return o.ConvertTo(ctx, to.ImageFormat)
		}
	case *path.As_VertexBufferFormat:
		f := to.VertexBufferFormat
		switch o := o.(type) {
		case *api.Mesh:
			return o.ConvertTo(ctx, f)
		}
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrUnsupportedConversion()}
}
