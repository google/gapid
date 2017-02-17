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
	"flag"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/master"
	"google.golang.org/grpc"
)

var (
	startVerb = &app.Verb{
		Name:      "start",
		ShortHelp: "Start a server",
	}
	stopVerb = &app.Verb{
		Name:      "stop",
		ShortHelp: "Stop a server",
		Run:       doStop,
	}
	restartVerb = &app.Verb{
		Name:      "restart",
		ShortHelp: "Restart a server",
		Run:       doRestart,
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

	stopNow = false
)

func init() {
	app.AddVerb(startVerb)
	app.AddVerb(stopVerb)
	app.AddVerb(restartVerb)
	app.AddVerb(searchVerb)
	app.AddVerb(uploadVerb)
	app.AddVerb(setVerb)

	stopVerb.Flags.Raw.BoolVar(&stopNow, "now", false, "Immediate shutdown")
}

func doStop(ctx log.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		client := master.NewClient(ctx, master.NewRemoteMaster(ctx, conn))
		if stopNow {
			return client.Kill(ctx, flags.Args()...)
		}
		return client.Shutdown(ctx, flags.Args()...)
	}, grpc.WithInsecure())
}

func doRestart(ctx log.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		client := master.NewClient(ctx, master.NewRemoteMaster(ctx, conn))
		return client.Restart(ctx, flags.Args()...)
	}, grpc.WithInsecure())
}
