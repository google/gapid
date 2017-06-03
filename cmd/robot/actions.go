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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/master"
	"google.golang.org/grpc"
)

var (
	startVerb = &app.Verb{
		Name:      "start",
		ShortHelp: "Start a server",
	}
	searchVerb = &app.Verb{
		Name:      "search",
		ShortHelp: "Search for content in the server",
	}
	uploadVerb = &app.Verb{
		Name:      "upload",
		ShortHelp: "Upload a file to a server",
	}
	setVerb = &app.Verb{
		Name:      "set",
		ShortHelp: "Sets a value in a server",
	}
)

func init() {
	app.AddVerb(startVerb)
	app.AddVerb(searchVerb)
	app.AddVerb(uploadVerb)
	app.AddVerb(setVerb)

	app.AddVerb(&app.Verb{
		Name:      "stop",
		ShortHelp: "Stop a server",
		Action:    &stopVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "restart",
		ShortHelp: "Restart a server",
		Action:    &restartVerb{},
	})
}

type stopVerb struct {
	Now bool `help:"Immediate shutdown"`
}

func (v *stopVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		client := master.NewClient(ctx, master.NewRemoteMaster(ctx, conn))
		if v.Now {
			return client.Kill(ctx, flags.Args()...)
		}
		return client.Shutdown(ctx, flags.Args()...)
	}, grpc.WithInsecure())
}

type restartVerb struct {
}

func (v *restartVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		client := master.NewClient(ctx, master.NewRemoteMaster(ctx, conn))
		return client.Restart(ctx, flags.Args()...)
	}, grpc.WithInsecure())
}
