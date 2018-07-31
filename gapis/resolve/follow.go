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

	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Follow resolves the path to the object that the value at Path links to.
// If the value at Path does not link to anything then nil is returned.
func Follow(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	obj, err := database.Build(ctx, &FollowResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*path.Any), nil
}

// Resolve implements the database.Resolver interface.
func (r *FollowResolvable) Resolve(ctx context.Context) (interface{}, error) {
	obj, err := ResolveInternal(ctx, r.Path.Node(), r.Config)
	if err != nil {
		return nil, err
	}

	linker, ok := obj.(path.Linker)
	if !ok {
		return nil, &service.ErrPathNotFollowable{Path: r.Path}
	}

	link, err := linker.Link(ctx, r.Path.Node(), r.Config)
	if err != nil {
		return link, err
	}
	if link == nil {
		return nil, &service.ErrPathNotFollowable{Path: r.Path}
	}
	if err := link.Validate(); err != nil {
		return nil, fmt.Errorf("Following path %v gave an invalid link %v: %v",
			r.Path, link, err)
	}
	return link.Path(), nil
}
