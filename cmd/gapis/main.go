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
	"path/filepath"
	"time"

	"google.golang.org/grpc/grpclog"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/server"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/stringtable"

	// Extensions
	_ "github.com/google/gapid/gapis/extensions/unity"
)

var (
	rpc             = flag.String("rpc", "localhost:0", "TCP host:port of the server's RPC listener")
	stringsPath     = flag.String("strings", "strings", "Directory containing string table packages")
	persist         = flag.Bool("persist", false, "Server will keep running even when no connections remain")
	gapisAuthToken  = flag.String("gapis-auth-token", "", "The connection authorization token for gapis")
	gapirAuthToken  = flag.String("gapir-auth-token", "", "The connection authorization token for gapir")
	gapirArgStr     = flag.String("gapir-args", "", `"The arguments to be passed to the host-run gapir"`)
	scanAndroidDevs = flag.Bool("monitor-android-devices", true, "Server will scan for locally connected Android devices")
	addLocalDevice  = flag.Bool("add-local-device", true, "Server will create a new local replay device")
	idleTimeout     = flag.Duration("idle-timeout", 0, "Closes GAPIS if the server is not repeatedly pinged within this duration")
	adbPath         = flag.String("adb", "", "Path to the adb executable; leave empty to search the environment")
)

func main() {
	app.ShortHelp = "GAPIS is the graphics API server"
	app.Name = "GAPIS" // Has to be this for version parsing compatability
	app.Run(run)
}

// features is the reported list of features supported by the server.
// This feature list can be used by the client to determine what new RPCs can be
// called.
var features = []string{}

// addFallbackLogHandler adds a handler to b that calls to fallback when nothing
// else is listening.
func addFallbackLogHandler(b *log.Broadcaster, fallback log.Handler) {
	b.Listen(log.NewHandler(func(m *log.Message) {
		if b.Count() == 1 {
			fallback.Handle(m)
		}
	}, fallback.Close))
}

func run(ctx context.Context) error {
	logBroadcaster := log.Broadcast()
	oldHandler := app.LogHandler.SetTarget(logBroadcaster)
	addFallbackLogHandler(logBroadcaster, oldHandler)

	if *adbPath != "" {
		adb.ADB = file.Abs(*adbPath)
	}

	r := bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, r)
	m := replay.New(ctx)
	ctx = replay.PutManager(ctx, m)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	grpclog.SetLogger(log.From(ctx))

	if *addLocalDevice {
		host := bind.Host(ctx)
		r.AddDevice(ctx, host)
		r.SetDeviceProperty(ctx, host, client.LaunchArgsKey, text.SplitArgs(*gapirArgStr))
	}

	deviceScanDone, onDeviceScanDone := task.NewSignal()
	if *scanAndroidDevs {
		crash.Go(func() { monitorAndroidDevices(ctx, r, onDeviceScanDone) })
	} else {
		onDeviceScanDone(ctx)
	}

	return server.Listen(ctx, *rpc, server.Config{
		Info: &service.ServerInfo{
			Name:         host.Instance(ctx).Name,
			VersionMajor: uint32(app.Version.Major),
			VersionMinor: uint32(app.Version.Minor),
			VersionPoint: uint32(app.Version.Point),
			Features:     features,
		},
		StringTables:   loadStrings(ctx),
		AuthToken:      auth.Token(*gapisAuthToken),
		DeviceScanDone: deviceScanDone,
		LogBroadcaster: logBroadcaster,
		IdleTimeout:    *idleTimeout,
	})
}

func monitorAndroidDevices(ctx context.Context, r *bind.Registry, onDeviceScanDone task.Task) {
	// Populate the registry with all the existing devices.
	func() {
		defer onDeviceScanDone(ctx) // Signal that we have a primed registry.

		if devs, err := adb.Devices(ctx); err == nil {
			for _, d := range devs {
				r.AddDevice(ctx, d)
			}
		}
	}()

	if err := adb.Monitor(ctx, r, time.Second*3); err != nil {
		log.W(ctx, "Could not scan for local Android devices. Error: %v", err)
	}
}

func loadStrings(ctx context.Context) []*stringtable.StringTable {
	files, err := filepath.Glob(filepath.Join(*stringsPath, "*.stb"))
	if err != nil {
		log.E(ctx, "Couldn't scan for stringtables. Error: %v", err)
		return nil
	}

	out := make([]*stringtable.StringTable, 0, len(files))

	for _, path := range files {
		ctx := log.V{"path": path}.Bind(ctx)
		st, err := stringtable.Load(path)
		if err != nil {
			log.E(ctx, "Couldn't load stringtable file. Error: %v", err)
			continue
		}
		out = append(out, st)
	}

	return out
}
