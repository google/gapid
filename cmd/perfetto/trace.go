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
	"errors"
	"flag"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"

	config "protos/perfetto/config"
)

type traceVerb struct {
	Config string `help:"File containing the trace configuration proto."`
	Out    string `help:"The file to store the trace data in."`
}

func init() {
	verb := &traceVerb{
		Out: "trace.perfetto",
	}
	app.AddVerb(&app.Verb{
		Name:      "trace",
		ShortHelp: "Captures a Perfetto trace",
		Action:    verb,
	})
}

func (verb *traceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if verb.Config == "" {
		app.Usage(ctx, "The Perfetto trace config is required.")
		return nil
	}
	cfg, err := verb.readConfig(ctx)
	if err != nil {
		return err
	}

	out, err := os.Create(verb.Out)
	if err != nil {
		return log.Errf(ctx, err, "Failed to create output file")
	}
	defer out.Close()

	ctx = setupContext(ctx)

	d, err := findDevice(ctx)
	if err != nil {
		return err
	}

	c, err := connectToPerfetto(ctx, d)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	sess, err := c.Trace(ctx, cfg, out)
	if err != nil {
		return log.Errf(ctx, err, "Failed to start Perfetto trace")
	}

	if err := sess.Wait(ctx); err != nil {
		return log.Errf(ctx, err, "Perfetto trace failed")
	}

	return nil
}

func (verb *traceVerb) readConfig(ctx context.Context) (*config.TraceConfig, error) {
	data, err := ioutil.ReadFile(verb.Config)
	if err != nil {
		return nil, log.Errf(ctx, err, "Failed to read the trace config")
	}
	cfg := &config.TraceConfig{}
	if err := proto.UnmarshalText(string(data), cfg); err != nil {
		return nil, log.Errf(ctx, err, "Failed to parse Perfetto config")
	}
	return cfg, nil
}

func findDevice(ctx context.Context) (bind.Device, error) {
	for _, d := range bind.GetRegistry(ctx).Devices() {
		if !d.SupportsPerfetto(ctx) {
			log.I(ctx, "Device %s doesn't support Perfetto", d)
			continue
		}
		return d, nil
	}
	return nil, errors.New("No Perfetto supporting device found")
}
