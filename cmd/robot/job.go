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
	"io/ioutil"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/search/script"
	stashgrpc "github.com/google/gapid/core/data/stash/grpc"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/monitor"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/trace"
	"google.golang.org/grpc"
)

func init() {
	deviceSearch := &app.Verb{
		Name:       "device",
		ShortHelp:  "List the devices",
		ShortUsage: "<query>",
		Run:        doDeviceSearch,
	}
	searchVerb.Add(deviceSearch)
	workerSearch := &app.Verb{
		Name:       "worker",
		ShortHelp:  "List the workers",
		ShortUsage: "<query>",
		Run:        doWorkerSearch,
	}
	searchVerb.Add(workerSearch)
	workerStart := &app.Verb{
		Name:      "worker",
		ShortHelp: "Starts a robot worker",
		Run:       doWorkerStart,
	}
	startVerb.Add(workerStart)
}

func doDeviceSearch(ctx log.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		w := job.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := ctx.Raw("").Writer()
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return cause.Explain(ctx, err, "Malformed search query")
		}
		return w.SearchDevices(ctx, expr.Query(), func(ctx log.Context, entry *job.Device) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

func doWorkerSearch(ctx log.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		w := job.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := ctx.Raw("").Writer()
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return cause.Explain(ctx, err, "Malformed search query")
		}
		return w.SearchWorkers(ctx, expr.Query(), func(ctx log.Context, entry *job.Worker) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

func doWorkerStart(ctx log.Context, flags flag.FlagSet) error {
	tempName, err := ioutil.TempDir("", "robot")
	if err != nil {
		return err
	}
	tempDir := file.Abs(tempName)
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		m := master.NewClient(ctx, master.NewRemoteMaster(ctx, conn))
		managers := monitor.Managers{
			Stash:  stashgrpc.MustConnect(ctx, conn),
			Trace:  trace.NewRemote(ctx, conn),
			Report: report.NewRemote(ctx, conn),
			Replay: replay.NewRemote(ctx, conn),
		}
		if err := startAllWorkers(ctx, managers, tempDir); err != nil {
			return err
		}
		shutdown, err := m.Orbit(ctx, master.ServiceList{Worker: true})
		if err != nil {
			return err
		}
		if shutdown.Restart {
			return app.Restart
		}
		return nil
	}, grpc.WithInsecure())
}

func startAllWorkers(ctx log.Context, managers monitor.Managers, tempDir file.Path) error {
	// TODO: not just ignore all the errors...
	go trace.Run(ctx, managers.Stash, managers.Trace, tempDir)
	go report.Run(ctx, managers.Stash, managers.Report, tempDir)
	go replay.Run(ctx, managers.Stash, managers.Replay, tempDir)
	return nil
}
