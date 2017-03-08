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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/schema"
	"google.golang.org/grpc"
)

const (
	BinaryName = "gapis"
)

var (
	// GapisPath is the full filepath to the gapir executable.
	GapisPath file.Path
)

func init() {
	// Search directory that this executable is in.
	if path, err := file.FindExecutable(file.ExecutablePath().Parent().Join(BinaryName).System()); err == nil {
		GapisPath = path
		return
	}
	// Search standard package structure
	packagePath := file.Abs(".")
	switch device.Host(context.Background()).Configuration.OS.Kind {
	case device.Windows:
		packagePath = packagePath.Join("windows")
	case device.OSX:
		packagePath = packagePath.Join("osx")
	case device.Linux:
		packagePath = packagePath.Join("linux")
	}
	packagePath = packagePath.Join("x86_64")
	if path, err := file.FindExecutable(packagePath.Join(BinaryName).System()); err == nil {
		GapisPath = path
		return
	}
	// Search $PATH.
	if path, err := file.FindExecutable(BinaryName); err == nil {
		GapisPath = path
		return
	}
}

type Config struct {
	Path  *file.Path
	Port  int
	Args  []string
	Token auth.Token
}

// Connect attempts to connect to a GAPIS process.
// If port is zero, a new GAPIS server will be started, otherwise a connection
// will be made to the specified port.
func Connect(ctx context.Context, cfg Config) (Client, *schema.Message, error) {
	if cfg.Path == nil {
		cfg.Path = &GapisPath
	}

	var err error
	if cfg.Port == 0 || len(cfg.Args) > 0 {
		cfg.Args = append(cfg.Args, "--log-level", logLevel(ctx).String())
		if cfg.Token != auth.NoAuth {
			cfg.Args = append(cfg.Args, "--gapis-auth-token", string(cfg.Token))
		}
		cfg.Port, err = process.Start(ctx, cfg.Path.System(), nil, cfg.Args...)
		if err != nil {
			return nil, nil, err
		}
	}

	target := fmt.Sprintf("localhost:%d", cfg.Port)

	conn, err := grpcutil.Dial(ctx, target,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(auth.ClientInterceptor(cfg.Token)))
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Dialing GAPIS")
	}
	client := Bind(conn)

	message, err := client.GetSchema(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("Error resolving schema: %v", err)
	}

	for _, entity := range message.Entities {
		registry.Global.Add((*schema.ObjectClass)(entity))
	}

	return client, message, nil
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
