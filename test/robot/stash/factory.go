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

package stash

import (
	"context"
	"net/url"
	"runtime"

	"github.com/google/gapid/core/log"
)

// Builder is the function type that opens a stash from a url.
type Builder func(context.Context, *url.URL) (*Client, error)

var schemeMap = map[string]Builder{}

// ReigsterHandler adds a new scheme handler to the factory.
func RegisterHandler(scheme string, builder Builder) {
	//TODO: complain about duplicates?
	schemeMap[scheme] = builder
}

// Dial returns a new client for the given url.
func Dial(ctx context.Context, address string) (*Client, error) {
	ctx = log.V{"Store": address}.Bind(ctx)
	location, err := url.Parse(address)
	if err != nil {
		return nil, log.Err(ctx, err, "Invalid server location")
	}
	if location.Scheme == "" {
		switch {
		case location.Host != "":
			location.Scheme = "grpc"
		case location.Path == "":
			location.Scheme = "memory"
		default:
			location.Scheme = "file"
		}
	}
	if location.Scheme == "file" {
		if runtime.GOOS == "windows" {
			if l := len(location.Path); l > 2 && location.Path[0] == '/' && location.Path[2] == ':' {
				// windows file urls have an extra slash before the volume label that needs to be remove
				// see https://github.com/golang/go/commit/844b625ebcc7101e09fb87828a0e71db942a2416d
				location.Path = location.Path[1:]
				log.I(ctx, "location of new file stash is %v", location.Path)
			}
		}
	}
	builder, found := schemeMap[location.Scheme]
	if !found {
		return nil, log.Errf(ctx, nil, "Invalid stash scheme")
	}
	return builder(ctx, location)
}
