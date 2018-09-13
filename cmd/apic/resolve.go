// Copyright (C) 2018 Google Inc.
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

package main

import (
	"context"

	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

func resolve(ctx context.Context, paths []string, search file.PathList, opts resolver.Options) ([]*semantic.API, *semantic.Mappings, error) {
	processor := gapil.NewProcessor()
	processor.Options = opts
	if len(search) > 0 {
		processor.Loader = gapil.NewSearchLoader(search)
	}

	apis := make([]*semantic.API, len(paths))
	for i, path := range paths {
		api, errs := processor.Resolve(path)
		if err := gapil.CheckErrors(path, errs, maxErrors); err != nil {
			return nil, nil, err
		}
		apis[i] = api
	}
	return apis, processor.Mappings, nil
}
