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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/stringtable"
)

type reportVerb struct{ ReportFlags }

func init() {
	verb := &reportVerb{}
	app.AddVerb(&app.Verb{
		Name:      "report",
		ShortHelp: "Check a capture replays without issues",
		Action:    verb,
	})
}

func (verb *reportVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capturePath, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	gapisTrace := &bytes.Buffer{}
	stopGapisTrace, err := client.Profile(ctx, nil, gapisTrace, 1)
	if err != nil {
		return err
	}
	defer func() {
		stopGapisTrace()
		ioutil.WriteFile("report.out", gapisTrace.Bytes(), 0644)
	}()

	if err != nil {
		return err
	}
	defer client.Close()

	stringTables, err := client.GetAvailableStringTables(ctx)
	if err != nil {
		return log.Err(ctx, err, "Failed get list of string tables")
	}

	var stringTable *stringtable.StringTable
	if len(stringTables) > 0 {
		// TODO: Let the user pick the string table.
		stringTable, err = client.GetStringTable(ctx, stringTables[0])
		if err != nil {
			return log.Err(ctx, err, "Failed get string table")
		}
	}

	device, err := getDevice(ctx, client, capturePath, verb.Gapir)
	if err != nil {
		return err
	}

	boxedCommands, err := client.Get(ctx, capturePath.Commands().Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Failed to acquire the capture's commands")
	}
	commands := boxedCommands.(*service.Commands).List

	boxedReport, err := client.Get(ctx, capturePath.Report(device, verb.DisplayToSurface).Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Failed to acquire the capture's report")
	}

	var reportWriter io.Writer = os.Stdout
	if verb.Out != "" {
		f, err := os.OpenFile(verb.Out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return log.Err(ctx, err, "Failed to open report output file")
		}
		defer f.Close()
		reportWriter = f
	}

	report := boxedReport.(*service.Report)
	for _, e := range report.Items {
		where := ""
		if e.Command != nil {
			where = fmt.Sprintf("%v %v ", e.Command.Indices, commands[e.Command.Indices[0]]) // TODO: Subcommands
		}
		msg := report.Msg(e.Message).Text(stringTable)
		fmt.Fprintln(reportWriter, fmt.Sprintf("[%s] %s%s", e.Severity.String(), where, msg))
	}

	if len(report.Items) == 0 {
		fmt.Fprintln(reportWriter, "No issues found")
	} else {
		fmt.Fprintf(reportWriter, "%d issues found\n", len(report.Items))
	}

	return nil
}
