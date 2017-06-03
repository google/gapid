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
	"net"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/stash"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"google.golang.org/grpc"
)

func init() {
	verb := &app.Verb{
		Name:      "server",
		ShortHelp: "Starts a stash server",
		Action:    &serverVerb{},
	}
	app.AddVerb(verb)
}

type serverVerb struct{}

func (v *serverVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	serveAt := ""
	switch flags.NArg() {
	case 0:
		serveAt = defaultStashServer
	case 1:
		serveAt = flags.Arg(0)
	default:
		app.Usage(ctx, "Expected at most 1 arg (the address to server on)")
		return nil
	}
	log.I(ctx, "Starting server on %s", serveAt)
	return withStore(ctx, true, func(ctx context.Context, client *stash.Client) error {
		return serveStore(ctx, client, serveAt)
	})
}

func serveStore(ctx context.Context, client *stash.Client, address string) error {
	log.I(ctx, "Serving on %s", address)
	return grpcutil.Serve(ctx, address, func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
		if err := stashgrpc.Serve(ctx, server, client); err != nil {
			return err
		}
		return nil
	})
}
