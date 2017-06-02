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
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/monitor"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/subject"
	"github.com/google/gapid/test/robot/trace"
	"github.com/google/gapid/test/robot/web"
	"google.golang.org/grpc"
)

var (
	port = 8080
	root = file.Path{}
)

func init() {
	webStart := &app.Verb{
		Name:      "web",
		ShortHelp: "Starts a robot web server",
		Run:       doWebStart,
	}
	webStart.Flags.Raw.IntVar(&port, "port", port, "The port to serve the website on")
	webStart.Flags.Raw.Var(&root, "root", "The directory to use as the root of static content")
	startVerb.Add(webStart)
}

func doWebStart(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		config := web.Config{
			Port:       port,
			StaticRoot: root,
			Managers: monitor.Managers{
				Master:  master.NewRemoteMaster(ctx, conn),
				Stash:   stashgrpc.MustConnect(ctx, conn),
				Build:   build.NewRemote(ctx, conn),
				Subject: subject.NewRemote(ctx, conn),
				Job:     job.NewRemote(ctx, conn),
				Trace:   trace.NewRemote(ctx, conn),
				Replay:  replay.NewRemote(ctx, conn),
				Report:  report.NewRemote(ctx, conn),
			},
		}
		w, err := web.Create(ctx, config)
		m := master.NewClient(ctx, config.Master)
		restart := false
		go func() {
			shutdown, err := m.Orbit(ctx, master.ServiceList{Worker: true})
			if err != nil {
				return
			}
			restart = shutdown.Restart
			w.Close()
		}()
		log.I(ctx, "Starting web server")
		err = w.Serve(ctx)
		if restart {
			return app.Restart
		}
		return err
	}, grpc.WithInsecure())
}
