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
	"flag"
	"fmt"

	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

func resolve(ctx context.Context, search file.PathList, flags flag.FlagSet, opts resolver.Options) (*semantic.API, *semantic.Mappings, error) {
	args := flags.Args()
	if len(args) < 1 {
		return nil, nil, fmt.Errorf("Missing api file")
	}
	path := args[0]
	processor := gapil.NewProcessor()
	if len(search) > 0 {
		processor.Loader = gapil.NewSearchLoader(search)
	}
	processor.Options = opts
	compiled, errs := processor.Resolve(path)
	if err := gapil.CheckErrors(path, errs, maxErrors); err != nil {
		return nil, nil, err
	}
	return compiled, processor.Mappings, nil
}
