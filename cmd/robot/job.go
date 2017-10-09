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
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/monitor"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/search/script"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"github.com/google/gapid/test/robot/trace"
	"google.golang.org/grpc"
)

func init() {
	searchVerb.Add(&app.Verb{
		Name:       "device",
		ShortHelp:  "List the devices",
		ShortUsage: "<query>",
		Action:     &deviceSearchFlags{RobotOptions: defaultRobotOptions},
	})
	searchVerb.Add(&app.Verb{
		Name:       "worker",
		ShortHelp:  "List the workers",
		ShortUsage: "<query>",
		Action:     &workerSearchFlags{RobotOptions: defaultRobotOptions},
	})
	startVerb.Add(&app.Verb{
		Name:      "worker",
		ShortHelp: "Starts a robot worker",
		Action:    &workerStartFlags{RobotOptions: defaultRobotOptions},
	})
}

type deviceSearchFlags struct {
	RobotOptions
}

func (v *deviceSearchFlags) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		w := job.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return w.SearchDevices(ctx, expr.Query(), func(ctx context.Context, entry *job.Device) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

type workerSearchFlags struct {
	RobotOptions
}

func (v *workerSearchFlags) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		w := job.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return w.SearchWorkers(ctx, expr.Query(), func(ctx context.Context, entry *job.Worker) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

type workerStartFlags struct {
	RobotOptions
}

func (v *workerStartFlags) Run(ctx context.Context, flags flag.FlagSet) error {
	tempName, err := ioutil.TempDir("", "robot")
	if err != nil {
		return err
	}
	tempDir := file.Abs(tempName)
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
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

func startAllWorkers(ctx context.Context, managers monitor.Managers, tempDir file.Path) error {
	// TODO: not just ignore all the errors...
	crash.Go(func() { trace.Run(ctx, managers.Stash, managers.Trace, tempDir) })
	crash.Go(func() { report.Run(ctx, managers.Stash, managers.Report, tempDir) })
	crash.Go(func() { replay.Run(ctx, managers.Stash, managers.Replay, tempDir) })
	return nil
}
