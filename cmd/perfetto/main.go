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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	_ "github.com/google/gapid/gapidapk"
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

	return ctx
}
