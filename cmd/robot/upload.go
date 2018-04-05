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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/stash"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"google.golang.org/grpc"
)

type uploader interface {
	prepare(context.Context, *grpc.ClientConn) error
	process(context.Context, string) error
}

type uploadable interface {
	upload(ctx context.Context, client *stash.Client) (string, error)
}

type path string

func (p path) upload(ctx context.Context, client *stash.Client) (string, error) {
	return client.UploadFile(ctx, file.Abs(string(p)))
}

func paths(paths []string) []uploadable {
	r := make([]uploadable, len(paths))
	for i, p := range paths {
		r[i] = path(p)
	}
	return r
}

type slice struct {
	data []byte
	info stash.Upload
}

func (s slice) upload(ctx context.Context, client *stash.Client) (string, error) {
	return client.UploadBytes(ctx, s.info, s.data)
}

func data(data []byte, name string, executable bool) slice {
	return slice{
		data: data,
		info: stash.Upload{
			Name:       []string{name},
			Executable: executable,
		},
	}
}

func upload(ctx context.Context, uploadables []uploadable, serverAddress string, u uploader) error {
	if len(uploadables) == 0 {
		app.Usage(ctx, "No files given")
		return nil
	}
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		client, err := stashgrpc.Connect(ctx, conn)
		if err != nil {
			return err
		}
		if err := u.prepare(ctx, conn); err != nil {
			return err
		}
		for _, partial := range uploadables {
			id, err := partial.upload(ctx, client)
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
