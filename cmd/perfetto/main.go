// Copyright (C) 2019 Google Inc.
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
	"io"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/remotessh"
	_ "github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapis/perfetto"
)

var (
	remoteSSHConfig = flag.String("ssh-config", "", "_Path to an ssh config file for remote devices")
)

func main() {
	app.ShortHelp = "perfetto is a Perfetto command line utility"
	app.Name = "perfetto"
	app.Run(app.VerbMain)
}

func setupContext(ctx context.Context) context.Context {
	r := bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, r)

	host := bind.Host(ctx)
	r.AddDevice(ctx, host)

	if devs, err := adb.Devices(ctx); err == nil {
		for _, d := range devs {
			r.AddDevice(ctx, d)
		}
	}

	if *remoteSSHConfig != "" {
		f, err := os.Open(*remoteSSHConfig)
		if err != nil {
			log.E(ctx, "Failed to open remote SSH config: %s", err)
		} else if devs, err := remotessh.Devices(ctx, []io.ReadCloser{f}); err == nil {
			for _, d := range devs {
				r.AddDevice(ctx, d)
			}
		}
	}

	return ctx
}

func connectToPerfetto(ctx context.Context, d bind.Device) (*perfetto.Client, error) {
	log.I(ctx, "Connecting to Perfetto...")
	c, err := d.ConnectPerfetto(ctx)
	if err != nil {
		log.E(ctx, "Failed to connect to perfetto: %s", err)
		return nil, err
	}
	log.I(ctx, "Connected.")
	return c, nil
}
