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
	"net"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/data/record"
	"github.com/google/gapid/core/data/search/script"
	"github.com/google/gapid/core/data/stash"
	stashgrpc "github.com/google/gapid/core/data/stash/grpc"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/monitor"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/scheduler"
	"github.com/google/gapid/test/robot/subject"
	"github.com/google/gapid/test/robot/trace"
	"github.com/google/gapid/test/robot/web"
	"google.golang.org/grpc"
)

var (
	baseAddr     = file.Abs(".")
	stashAddr    = ""
	shelfAddr    = ""
	startWorkers = true
	startWeb     = true
)

func init() {
	verb := &app.Verb{
		Name:      "master",
		ShortHelp: "Starts a robot master server",
		Run:       doMasterStart,
	}
	verb.Flags.Raw.Var(&baseAddr, "base", "The base path for all robot files")
	verb.Flags.Raw.StringVar(&stashAddr, "stash", stashAddr, "The address of the stash, defaults to a directory below base")
	verb.Flags.Raw.StringVar(&shelfAddr, "shelf", shelfAddr, "The path to the persisted data, defaults to a directory below base")
	verb.Flags.Raw.BoolVar(&startWorkers, "worker", startWorkers, "Enables local workers")
	verb.Flags.Raw.BoolVar(&startWeb, "web", startWeb, "Enables serving the web client")
	verb.Flags.Raw.IntVar(&port, "port", port, "The port to serve the website on")
	verb.Flags.Raw.Var(&root, "root", "The directory to use as the root of static content")
	startVerb.Add(verb)
	masterSearch := &app.Verb{
		Name:       "master",
		ShortHelp:  "List satellites registered with the master",
		ShortUsage: "<query>",
		Run:        doMasterSearch,
	}
	searchVerb.Add(masterSearch)
}

func doMasterStart(ctx log.Context, flags flag.FlagSet) error {
	tempName, err := ioutil.TempDir("", "robot")
	if err != nil {
		return err
	}
	tempDir := file.Abs(tempName)
	restart := false
	err = grpcutil.Serve(ctx, serverAddress, func(ctx log.Context, listener net.Listener, server *grpc.Server) error {
		managers := monitor.Managers{}
		err := error(nil)
		if stashAddr == "" {
			stashAddr = baseAddr.Join("stash").System()
		}
		if shelfAddr == "" {
			shelfAddr = baseAddr.Join("shelf").System()
		}
		library := record.NewLibrary(ctx)
		shelf, err := record.NewShelf(ctx, shelfAddr)
		if err != nil {
			return cause.Explain(ctx, err, "Could not open shelf").With("shelf", shelfAddr)
		}
		library.Add(ctx, shelf)
		if managers.Stash, err = stash.Dial(ctx, stashAddr); err != nil {
			return cause.Explain(ctx, err, "Could not open stash").With("stash", stashAddr)
		}
		managers.Master = master.NewLocal(ctx)
		if managers.Subject, err = subject.NewLocal(ctx, library, managers.Stash); err != nil {
			return err
		}
		if managers.Build, err = build.NewLocal(ctx, managers.Stash, library); err != nil {
			return err
		}
		if managers.Job, err = job.NewLocal(ctx, library); err != nil {
			return err
		}
		if managers.Trace, err = trace.NewLocal(ctx, library, managers.Job); err != nil {
			return err
		}
		if managers.Report, err = report.NewLocal(ctx, library, managers.Job); err != nil {
			return err
		}
		if managers.Replay, err = replay.NewLocal(ctx, library, managers.Job); err != nil {
			return err
		}
		if err := serveAll(ctx, server, managers); err != nil {
			return err
		}
		if startWorkers {
			if err := startAllWorkers(ctx, managers, tempDir); err != nil {
				return err
			}
		}
		go func() {
			if err := monitor.Run(ctx, managers, monitor.NewDataOwner(), scheduler.Tick); err != nil {
				jot.Fatal(ctx, err, "Scheduler died")
			}
		}()

		if startWeb {
			config := web.Config{
				Port:       port,
				StaticRoot: root,
				Managers:   managers,
			}
			w, err := web.Create(ctx, config)
			if err != nil {
				return err
			}
			go w.Serve(ctx)
		}

		c := master.NewClient(ctx, managers.Master)
		services := master.ServiceList{
			Master: true,
			Worker: startWorkers,
			Web:    startWeb,
		}
		go func() {
			shutdown, err := c.Orbit(ctx, services)
			if err != nil {
				ctx.Notice().Log("Orbit failed")
				server.Stop()
				return
			}
			restart = shutdown.Restart
			if shutdown.Now {
				ctx.Notice().Log("Kill now")
				server.Stop()
			} else {
				ctx.Notice().Log("Graceful stop")
				server.GracefulStop()
			}
		}()
		return nil
	})
	if restart {
		return app.Restart
	}
	return err
}

func serveAll(ctx log.Context, server *grpc.Server, managers monitor.Managers) error {
	if err := master.Serve(ctx, server, managers.Master); err != nil {
		return err
	}
	if err := stashgrpc.Serve(ctx, server, managers.Stash); err != nil {
		return err
	}
	if err := subject.Serve(ctx, server, managers.Subject); err != nil {
		return err
	}
	if err := build.Serve(ctx, server, managers.Build); err != nil {
		return err
	}
	if err := job.Serve(ctx, server, managers.Job); err != nil {
		return err
	}
	if err := trace.Serve(ctx, server, managers.Trace); err != nil {
		return err
	}
	if err := report.Serve(ctx, server, managers.Report); err != nil {
		return err
	}
	if err := replay.Serve(ctx, server, managers.Replay); err != nil {
		return err
	}
	return nil
}

func doMasterSearch(ctx log.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
		m := master.NewRemoteMaster(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := ctx.Raw("")
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return cause.Explain(ctx, err, "Malformed search query")
		}
		return m.Search(ctx, expr.Query(), func(ctx log.Context, entry *master.Satellite) error {
			out.Log(entry.String())
			return nil
		})
	}, grpc.WithInsecure())
}
