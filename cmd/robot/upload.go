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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"google.golang.org/grpc"
)

type uploader interface {
	prepare(context.Context, *grpc.ClientConn) error
	process(context.Context, string) error
}

func upload(ctx context.Context, flags flag.FlagSet, serverAddress string, u uploader) error {
	if flags.NArg() == 0 {
		app.Usage(ctx, "No files given")
		return nil
	}
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		client, err := stashgrpc.Connect(ctx, conn)
		if err != nil {
			return err
		}
		u.prepare(ctx, conn)
		for _, partial := range flags.Args() {
			id, err := client.UploadFile(ctx, file.Abs(partial))
			if err != nil {
				return log.Err(ctx, err, "Failed calling Upload")
			}
			log.I(ctx, "Uploaded %s", id)
			if err := u.process(ctx, id); err != nil {
				return err
			}
		}
		return nil
	}, grpc.WithInsecure())
}
