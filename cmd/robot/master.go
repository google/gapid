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
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/monitor"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/scheduler"
	"github.com/google/gapid/test/robot/search/script"
	"github.com/google/gapid/test/robot/stash"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"github.com/google/gapid/test/robot/subject"
	"github.com/google/gapid/test/robot/trace"
	"github.com/google/gapid/test/robot/web"
	"google.golang.org/grpc"
)

const defaultMasterPort = 8081

var defaultMasterAddress = fmt.Sprintf("localhost:%v", defaultMasterPort)

func init() {
	startVerb.Add(&app.Verb{
		Name:      "master",
		ShortHelp: "Starts a robot master server",
		Action: &masterVerb{
			BaseAddr:     file.Abs("."),
			StashAddr:    "",
			ShelfAddr:    "",
			Port:         defaultMasterPort,
			StartWorkers: true,
			StartWeb:     true,
			WebPort:      8080,
		},
	})
	searchVerb.Add(&app.Verb{
		Name:       "master",
		ShortHelp:  "List satellites registered with the master",
		ShortUsage: "<query>",
		Action:     &masterSearchVerb{RobotOptions: defaultRobotOptions},
	})
}

type masterVerb struct {
	BaseAddr     file.Path `help:"The base path for all robot files"`
	StashAddr    string    `help:"The address of the stash, defaults to a directory below base"`
	ShelfAddr    string    `help:"The path to the persisted data, defaults to a directory below base"`
	Port         int       `help:"The port to serve the master on"`
	StartWorkers bool      `help:"Enables local workers"`
	StartWeb     bool      `help:"Enables serving the web client"`
	WebPort      int       `help:"The port to serve the website on"`
	Root         file.Path `help:"The directory to use as the root of static content"`
}

func (v *masterVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	tempName, err := ioutil.TempDir("", "robot")
	if err != nil {
		return err
	}
	tempDir := file.Abs(tempName)
	restart := false
	serverAddress := fmt.Sprintf(":%v", v.Port)
	err = grpcutil.Serve(ctx, serverAddress, func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
		managers := monitor.Managers{}
		err := error(nil)
		var stashURL *url.URL
		var shelfURL *url.URL
		if v.StashAddr == "" {
			stashURL = v.BaseAddr.Join("stash").URL()
		} else if shelfURL, err = url.Parse(v.StashAddr); err != nil {
			return log.Errf(ctx, err, "Invalid server location", v.StashAddr)
		}
		if v.ShelfAddr == "" {
			shelfURL = v.BaseAddr.Join("shelf").URL()
		} else if stashURL, err = url.Parse(v.ShelfAddr); err != nil {
			return log.Errf(ctx, err, "Invalid record shelf location", v.ShelfAddr)
		}
		library := record.NewLibrary(ctx)
		shelf, err := record.NewShelf(ctx, shelfURL)
		if err != nil {
			return log.Errf(ctx, err, "Could not open shelf: %v", shelfURL)
		}
		library.Add(ctx, shelf)
		if managers.Stash, err = stash.Dial(ctx, stashURL); err != nil {
			return log.Errf(ctx, err, "Could not open stash: %v", stashURL)
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
		if v.StartWorkers {
			if err := startAllWorkers(ctx, managers, tempDir); err != nil {
				return err
			}
		}
		crash.Go(func() {
			if err := monitor.Run(ctx, managers, monitor.NewDataOwner(), scheduler.Tick); err != nil {
				log.E(ctx, "Scheduler died. Error: %v", err)
			}
		})

		if v.StartWeb {
			config := web.Config{
				Port:       v.WebPort,
				StaticRoot: v.Root,
				Managers:   managers,
			}
			w, err := web.Create(ctx, config)
			if err != nil {
				return err
			}
			crash.Go(func() { w.Serve(ctx) })
		}

		c := master.NewClient(ctx, managers.Master)
		services := master.ServiceList{
			Master: true,
			Worker: v.StartWorkers,
			Web:    v.StartWeb,
		}
		crash.Go(func() {
			shutdown, err := c.Orbit(ctx, services)
			if err != nil {
				log.I(ctx, "Orbit failed")
				server.Stop()
				return
			}
			restart = shutdown.Restart
			if shutdown.Now {
				log.I(ctx, "Kill now")
				server.Stop()
			} else {
				log.I(ctx, "Graceful stop")
				server.GracefulStop()
			}
		})
		return nil
	})
	if restart {
		return app.Restart
	}
	return err
}

func serveAll(ctx context.Context, server *grpc.Server, managers monitor.Managers) error {
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

type masterSearchVerb struct {
	RobotOptions
}

func (v *masterSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		m := master.NewRemoteMaster(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return m.Search(ctx, expr.Query(), func(ctx context.Context, entry *master.Satellite) error {
			log.I(ctx, "%s", entry.String())
			return nil
		})
	}, grpc.WithInsecure())
}
