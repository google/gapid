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
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/search/script"
	"google.golang.org/grpc"
)

func init() {
	searchVerb.Add(&app.Verb{
		Name:       "replay",
		ShortHelp:  "List build replays in the server",
		ShortUsage: "<query>",
		Action:     &replaySearchFlags{RobotOptions: defaultRobotOptions},
	})
}

type replaySearchFlags struct {
	RobotOptions
}

func (v *replaySearchFlags) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		replays := replay.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return replays.Search(ctx, expr.Query(), func(ctx context.Context, entry *replay.Action) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}
