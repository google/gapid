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
	"flag"

	"github.com/google/gapid/core/app"
	stashgrpc "github.com/google/gapid/core/data/stash/grpc"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"google.golang.org/grpc"
)

type uploader interface {
	prepare(log.Context, *grpc.ClientConn) error
	process(log.Context, string) error
}

func doUpload(u uploader) func(log.Context, flag.FlagSet) error {
	return func(ctx log.Context, flags flag.FlagSet) error {
		if flags.NArg() == 0 {
			app.Usage(ctx, "No files given")
			return nil
		}
		return grpcutil.Client(ctx, serverAddress, func(ctx log.Context, conn *grpc.ClientConn) error {
			client, err := stashgrpc.Connect(ctx, conn)
			if err != nil {
				return err
			}
			u.prepare(ctx, conn)
			for _, partial := range flags.Args() {
				id, err := client.UploadFile(ctx, file.Abs(partial))
				if err != nil {
					return cause.Explain(ctx, err, "Failed calling Upload")
				}
				ctx.Raw("").Logf("Uploaded %s", id)
				if err := u.process(ctx, id); err != nil {
					return err
				}
			}
			return nil
		}, grpc.WithInsecure())
	}
}
