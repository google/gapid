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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/android/apk"
)

func main() {
	app.ShortHelp = "dump manifest prints the manifest of an apk."
	app.Run(run)
}

func run(ctx context.Context) error {
	path := flag.Arg(0)
	apkData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	files, err := apk.Read(ctx, apkData)
	if err != nil {
		return err
	}
	m, err := apk.GetManifestXML(ctx, files)
	if err != nil {
		return err
	}
	fmt.Println(m)
	return nil
}
