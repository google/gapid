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

package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/search/script"
	"github.com/google/gapid/test/robot/stash"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"google.golang.org/grpc"
)

func init() {
	uploadVerb.Add(&app.Verb{
		Name:       "stash",
		ShortHelp:  "Upload a file to the stash",
		ShortUsage: "<filenames>",
		Action:     &stashUploadVerb{RobotOptions: defaultRobotOptions},
	})
	searchVerb.Add(&app.Verb{
		Name:       "stash",
		ShortHelp:  "List entries in the stash",
		ShortUsage: "<query>",
		Action:     &stashSearchVerb{RobotOptions: defaultRobotOptions},
	})
}

type stashUploadVerb struct {
	RobotOptions
}

func (v *stashUploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return upload(ctx, flags, v.ServerAddress, v)
}
func (stashUploadVerb) prepare(context.Context, *grpc.ClientConn) error { return nil }
func (stashUploadVerb) process(context.Context, string) error           { return nil }

type stashSearchVerb struct {
	RobotOptions
}

func (v *stashSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		store, err := stashgrpc.Connect(ctx, conn)
		if err != nil {
			return err
		}
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return store.Search(ctx, expr.Query(), func(ctx context.Context, entry *stash.Entity) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}
