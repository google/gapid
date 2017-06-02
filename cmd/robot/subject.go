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
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/search/script"
	"github.com/google/gapid/test/robot/subject"
	"google.golang.org/grpc"
)

func init() {
	uploadVerb.Add(&app.Verb{
		Name:       "subject",
		ShortHelp:  "Upload a traceable application to the server",
		ShortUsage: "<filenames>",
		Auto:       &subjectUploadVerb{},
	})
	searchVerb.Add(&app.Verb{
		Name:       "subject",
		ShortHelp:  "List traceable applications in the server",
		ShortUsage: "<query>",
		Auto:       &subjectSearchVerb{},
	})
}

type subjectUploadVerb struct {
	TraceTime time.Duration `help:"trace time override (if non-zero)"`
	subjects  subject.Subjects
}

func (v *subjectUploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return upload(ctx, flags, v)
}
func (v *subjectUploadVerb) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	v.subjects = subject.NewRemote(ctx, conn)
	return nil
}
func (v *subjectUploadVerb) process(ctx context.Context, id string) error {
	var hints *subject.Hints
	if v.TraceTime != 0 {
		hints = &subject.Hints{TraceTime: ptypes.DurationProto(v.TraceTime)}
	}
	subject, created, err := v.subjects.Add(ctx, id, hints)
	if err != nil {
		return log.Err(ctx, err, "Failed processing subject")
	}
	if created {
		log.I(ctx, "Added new subject")
	} else {
		log.I(ctx, "Already existing subject")
	}

	log.I(ctx, "Subject info %s", subject)
	return nil
}

type subjectSearchVerb struct{}

func (v *subjectSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		subjects := subject.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return subjects.Search(ctx, expr.Query(), func(ctx context.Context, entry *subject.Subject) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}
