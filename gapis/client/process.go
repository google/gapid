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

package client

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/process"
	"google.golang.org/grpc"
)

const (
	BinaryName = "gapis"
)

type Config struct {
	Path  *file.Path
	Port  int
	Args  []string
	Token auth.Token
}

// Connect attempts to connect to a GAPIS process.
// If port is zero, a new GAPIS server will be started, otherwise a connection
// will be made to the specified port.
func Connect(ctx context.Context, cfg Config) (Client, error) {
	var err error
	if cfg.Path == nil {
		if cfg.Path, err = findGapis(ctx); err != nil {
			return nil, err
		}
	}

	if cfg.Port == 0 {
		cfg.Args = append(cfg.Args,
			"--log-level", logLevel(ctx).String(),
			"--log-style", log.Brief.String(),
		)
		if cfg.Token != auth.NoAuth {
			cfg.Args = append(cfg.Args, "--gapis-auth-token", string(cfg.Token))
		}

		cfg.Port, err = process.StartOnDevice(ctx, cfg.Path.System(), process.StartOptions{
			Args:    append(cfg.Args, layout.GoArgs(ctx)...),
			Verbose: true,
			Device:  bind.Host(ctx),
		})
		if err != nil {
			return nil, err
		}
	}

	target := fmt.Sprintf("localhost:%d", cfg.Port)

	conn, err := grpcutil.Dial(ctx, target,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(auth.UnaryClientInterceptor(cfg.Token)),
		grpc.WithStreamInterceptor(auth.StreamClientInterceptor(cfg.Token)))
	if err != nil {
		return nil, log.Err(ctx, err, "Dialing GAPIS")
	}
	client := Bind(conn)

	return client, nil
}

func logLevel(ctx context.Context) log.Severity {
	f := log.GetFilter(ctx)
	for l := log.Debug; l <= log.Fatal; l++ {
		if f.ShowSeverity(l) {
			return l
		}
	}
	return log.Warning
}

func findGapis(ctx context.Context) (*file.Path, error) {
	if path, err := layout.Gapis(ctx); err == nil {
		return &path, nil
	}

	// Search $PATH.
	if path, err := file.FindExecutable(BinaryName); err == nil {
		return &path, nil
	}

	return nil, fmt.Errorf("Unable to locate the gapis executable '%s'", BinaryName)
}
