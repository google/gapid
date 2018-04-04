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
	"errors"
	"flag"
	"os"
	"os/user"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/git"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/search/script"
	"google.golang.org/grpc"
)

type UploadOptions struct {
	RobotOptions
	CL          string `help:"The build CL, will be guessed if not set"`
	Description string `help:"An optional build description"`
	Tag         string `help:"The optional build tag"`
	Track       string `help:"The package's track, will be guessed if not set"`
	Uploader    string `help:"The uploading entity, will be guessed if not set"`
	BuilderAbi  string `help:"The abi of the builder device, will assume this device if not set"`
	BuildBot    bool   `help:"Whether this package was built by a build bot"`
}

func init() {
	uploadVerb.Add(&app.Verb{
		Name:       "build",
		ShortHelp:  "Upload a build to the server",
		ShortUsage: "<zip file|directory>",
		Action:     &buildUploadVerb{UploadOptions: UploadOptions{RobotOptions: defaultRobotOptions}},
	})
	searchVerb.Add(&app.Verb{
		Name:       "artifact",
		ShortHelp:  "List build artifacts in the server",
		ShortUsage: "<query>",
		Action:     &artifactSearchVerb{RobotOptions: defaultRobotOptions},
	})
	searchVerb.Add(&app.Verb{
		Name:       "package",
		ShortHelp:  "List build packages in the server",
		ShortUsage: "<query>",
		Action:     &packageSearchVerb{RobotOptions: defaultRobotOptions},
	})
	searchVerb.Add(&app.Verb{
		Name:       "track",
		ShortHelp:  "List build tracks in the server",
		ShortUsage: "<query>",
		Action:     &trackSearchVerb{RobotOptions: defaultRobotOptions},
	})
	setVerb.Add(&app.Verb{
		Name:       "track",
		ShortHelp:  "Sets values on a track",
		ShortUsage: "<id or name>",
		Action:     &trackUpdateVerb{RobotOptions: defaultRobotOptions},
	})
	setVerb.Add(&app.Verb{
		Name:       "package",
		ShortHelp:  "Sets values on a package",
		ShortUsage: "<id>",
		Action:     &packageUpdateVerb{RobotOptions: defaultRobotOptions},
	})
}

type buildUploadVerb struct {
	UploadOptions

	store build.Store
	info  *build.Information
}

func (v *buildUploadVerb) Run(ctx context.Context, flags flag.FlagSet) (err error) {
	if flags.NArg() != 1 {
		err = errors.New("Missing expected argument")
		return log.Err(ctx, err, "build upload expects a single filepath as argument")
	}

	p := file.Abs(flags.Arg(0))
	u := make([]uploadable, 1)
	if p.IsDir() {
		b, err := zip(p)
		if err != nil {
			return log.Err(ctx, err, "failed to create artifact zip")
		}
		u[0] = data(b.Bytes(), p.Basename()+".zip", false)
	} else {
		u[0] = path(p.System())
	}

	return upload(ctx, u, v.ServerAddress, v)
}

func (v *buildUploadVerb) prepare(ctx context.Context, conn *grpc.ClientConn) (err error) {
	v.store = build.NewRemote(ctx, conn)
	v.info, err = v.UploadOptions.createBuildInfo(ctx)
	return
}

func (v *buildUploadVerb) process(ctx context.Context, id string) error {
	return storeBuild(ctx, v.store, v.info, id)
}

func storeBuild(ctx context.Context, store build.Store, info *build.Information, id string) error {
	id, merged, err := store.Add(ctx, id, info)
	if err != nil {
		return log.Err(ctx, err, "Failed processing build")
	}
	if merged {
		log.I(ctx, "Merged with build set %s", id)
	} else {
		log.I(ctx, "New build set %s", id)
	}
	return nil
}

func (o *UploadOptions) createBuildInfo(ctx context.Context) (*build.Information, error) {
	// see if we can find a git cl in the cwd
	var typ build.Type
	if o.BuildBot {
		typ = build.BuildBot
		if o.CL == "" || o.Description == "" || o.BuilderAbi == "" {
			err := errors.New("Missing expected argument")
			return nil, log.Err(ctx, err, "build bot packages require the CL, desciption and builder ABI")
		}
		if o.Track == "" {
			o.Track = "master"
		}
	} else {
		typ = o.initFromGit(ctx)
	}

	if o.Uploader == "" {
		// guess uploader from environment
		if user, err := user.Current(); err == nil {
			o.Uploader = user.Username
			log.I(ctx, "Detected uploader %s", o.Uploader)
		}
	}
	log.I(ctx, "Detected build type %s", typ)
	builder := &device.Instance{Configuration: &device.Configuration{}}
	if o.BuilderAbi != "" {
		if typ == build.BuildBot {
			builder.Name = "BuildBot"
		}
		builder.Configuration.ABIs = []*device.ABI{device.ABIByName(o.BuilderAbi)}
	}
	if len(builder.Configuration.ABIs) == 0 {
		builder = host.Instance(ctx)
	}
	return &build.Information{
		Type:        typ,
		Branch:      o.Track,
		Cl:          o.CL,
		Tag:         o.Tag,
		Description: o.Description,
		Builder:     builder,
		Uploader:    o.Uploader,
	}, nil
}

func (o *UploadOptions) initFromGit(ctx context.Context) (typ build.Type) {
	// Assume not clean, until we can verify.
	typ = build.Local

	g, err := git.New(".")
	if err != nil {
		log.E(ctx, "Git failed. Error: %v", err)
		return
	}

	if cl, err := g.HeadCL(ctx, o.CL); err != nil {
		log.E(ctx, "CL failed. Error: %v", err)
	} else {
		if o.CL == "" {
			// guess cl from git
			o.CL = cl.SHA.String()
			log.I(ctx, "Detected CL %s", o.CL)
		}
		if o.Description == "" {
			// guess description from git
			o.Description = cl.Subject
			log.I(ctx, "Detected description %s", o.Description)
		}
	}

	if status, err := g.Status(ctx); err != nil {
		log.E(ctx, "Status failed. Error: %v", err)
	} else if status.Clean() {
		typ = build.User
	}

	if o.Track == "" {
		// guess branch from git
		if branch, err := g.CurrentBranch(ctx); err != nil {
			log.E(ctx, "Branch failed. Error: %v", err)
			o.Track = "auto"
		} else {
			o.Track = branch
			log.I(ctx, "Detected track %s", o.Track)
		}
	}
	return
}

func zip(in file.Path) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := file.ZIP(buf, in); err != nil {
		return nil, err
	}
	return buf, nil
}

type artifactSearchVerb struct {
	RobotOptions
}

func (v *artifactSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return b.SearchArtifacts(ctx, expr.Query(), func(ctx context.Context, entry *build.Artifact) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

type packageSearchVerb struct {
	RobotOptions
}

func (v *packageSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return b.SearchPackages(ctx, expr.Query(), func(ctx context.Context, entry *build.Package) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

type trackSearchVerb struct {
	RobotOptions
}

func (v *trackSearchVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		expression := strings.Join(flags.Args(), " ")
		out := os.Stdout
		expr, err := script.Parse(ctx, expression)
		if err != nil {
			return log.Err(ctx, err, "Malformed search query")
		}
		return b.SearchTracks(ctx, expr.Query(), func(ctx context.Context, entry *build.Track) error {
			proto.MarshalText(out, entry)
			return nil
		})
	}, grpc.WithInsecure())
}

var idOrName = script.MustParse("Id == $ or Name == $").Using("$")

type trackUpdateVerb struct {
	RobotOptions
	Name        string `help:"The new name for the track"`
	Description string `help:"A description of the track"`
	Pkg         string `help:"The id of the package at the head of the track"`
}

func (v *trackUpdateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		args := flags.Args()
		track := &build.Track{
			Name:        v.Name,
			Description: v.Description,
			Head:        v.Pkg,
		}
		if len(args) != 0 {
			// Updating an existing track, find it first
			err := b.SearchTracks(ctx, idOrName(args[0]).Query(), func(ctx context.Context, entry *build.Track) error {
				if track.Id != "" {
					return log.Err(ctx, nil, "Multiple tracks matched")
				}
				track.Id = entry.Id
				return nil
			})
			if err != nil {
				return err
			}
			if track.Id == "" {
				return log.Err(ctx, nil, "No tracks matched")
			}
		}
		track, err := b.UpdateTrack(ctx, track)
		if err != nil {
			return err
		}
		log.I(ctx, track.String())
		return nil
	}, grpc.WithInsecure())
}

type packageUpdateVerb struct {
	RobotOptions
	Description string `help:"A description of the track"`
	Parent      string `help:"The id of the package that will be the new parent"`
}

func (v *packageUpdateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, v.ServerAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		args := flags.Args()
		pkg := &build.Package{
			Information: &build.Information{Description: v.Description},
			Parent:      v.Parent,
		}
		if len(args) == 0 {
			return log.Err(ctx, nil, "Missing argument, must specify a package to update")
		}
		err := b.SearchPackages(ctx, script.MustParse("Id == $").Using("$")(args[0]).Query(), func(ctx context.Context, entry *build.Package) error {
			if pkg.Id != "" {
				return log.Err(ctx, nil, "Multiple packages matched")
			}
			pkg.Id = entry.Id
			return nil
		})
		if err != nil {
			return err
		}
		if pkg.Id == "" {
			return log.Err(ctx, nil, "No packages matched")
		}
		pkg, err = b.UpdatePackage(ctx, pkg)
		if err != nil {
			return err
		}
		log.I(ctx, pkg.String())
		return nil
	}, grpc.WithInsecure())
}
