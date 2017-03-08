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
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/search/script"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/trace"
	"google.golang.org/grpc"
)

func init() {
	traceUpload := &app.Verb{
		Name:       "trace",
		ShortHelp:  "Upload a gfx trace to the server",
		ShortUsage: "<filenames>",
		Run:        doUpload(&traceUploader{}),
	}
	uploadVerb.Add(traceUpload)
	traceSearch := &app.Verb{
		Name:       "trace",
		ShortHelp:  "List build traces in the server",
		ShortUsage: "<query>",
		Run:        doTraceSearch,
	}
	searchVerb.Add(traceSearch)
}

type traceUploader struct {
	traces trace.Manager
}

func (u *traceUploader) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	u.traces = trace.NewRemote(ctx, conn)
	return nil
}

func (u *traceUploader) process(ctx context.Context, id string) error {
	return u.traces.Update(ctx, "", job.Succeeded, &trace.Output{Trace: id})
}

func doTraceSearch(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		traces := trace.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return traces.Search(ctx, expr.Query(), func(ctx context.Context, entry *trace.Action) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}
