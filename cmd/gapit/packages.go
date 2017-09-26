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
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapidapk"
)

type packagesVerb struct{ PackagesFlags }

func init() {
	app.AddVerb(&app.Verb{
		Name:      "packages",
		ShortHelp: "Prints information about packages installed on a device",
		Action: &packagesVerb{
			PackagesFlags{
				Icons:       false,
				IconDensity: 1.0,
			},
		},
	})
}

func (verb *packagesVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())

	if verb.ADB != "" {
		adb.ADB = file.Abs(verb.ADB)
	}

	d, err := getADBDevice(ctx, verb.Device)
	if err != nil {
		return err
	}

	pkgs, err := gapidapk.PackageList(ctx, d, verb.Icons, float32(verb.IconDensity))
	if err != nil {
		return log.Err(ctx, err, "getting package list")
	}

	w := os.Stdout
	if verb.Out != "" {
		f, err := os.OpenFile(verb.Out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return log.Err(ctx, err, "Failed to open package list output file")
		}
		w = f
		defer w.Close()
	}

	header := []byte{}
	if verb.DataHeader != "" {
		header = []byte(verb.DataHeader)
	}

	switch verb.Format {
	case ProtoString:
		w.Write(header)
		fmt.Fprint(w, pkgs.String())

	case Proto:
		data, err := proto.Marshal(pkgs)
		if err != nil {
			return log.Err(ctx, err, "marshal protobuf")
		}
		w.Write(header)
		w.Write(data)

	case Json:
		w.Write(header)
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		if err := e.Encode(pkgs); err != nil {
			return log.Err(ctx, err, "marshal json")
		}

	case SimpleList:
		w.Write(header)
		for _, a := range pkgs.GetPackages() {
			fmt.Fprintf(w, "%s\n", a.Name)
		}
	}

	return nil
}
