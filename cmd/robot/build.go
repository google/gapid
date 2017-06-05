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
	"os/user"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/git"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/search/script"
	"google.golang.org/grpc"
)

func init() {
	uploadVerb.Add(&app.Verb{
		Name:       "build",
		ShortHelp:  "Upload a build to the server",
		ShortUsage: "<filenames>",
		Action:     &buildUploadVerb{ServerAddress: defaultMasterAddress},
	})
	searchVerb.Add(&app.Verb{
		Name:       "artifact",
		ShortHelp:  "List build artifacts in the server",
		ShortUsage: "<query>",
		Action:     &artifactSearchVerb{ServerAddress: defaultMasterAddress},
	})
	searchVerb.Add(&app.Verb{
		Name:       "package",
		ShortHelp:  "List build packages in the server",
		ShortUsage: "<query>",
		Action:     &packageSearchVerb{ServerAddress: defaultMasterAddress},
	})
	searchVerb.Add(&app.Verb{
		Name:       "track",
		ShortHelp:  "List build tracks in the server",
		ShortUsage: "<query>",
		Action:     &trackSearchVerb{ServerAddress: defaultMasterAddress},
	})
	setVerb.Add(&app.Verb{
		Name:       "track",
		ShortHelp:  "Sets values on a track",
		ShortUsage: "<id or name>",
		Action:     &trackUpdateVerb{ServerAddress: defaultMasterAddress},
	})
}

type buildUploadVerb struct {
	CL            string `help:"The build CL, will be guessed if not set"`
	Description   string `help:"An optional build description"`
	Tag           string `help:"The optional build tag"`
	Branch        string `help:"The build branch, will be guessed if not set"`
	Uploader      string `help:"The uploading entity, will be guessed if not set"`
	ServerAddress string `help:"The master server address"`

	store build.Store
	info  *build.Information
}

func (v *buildUploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return upload(ctx, flags, v.ServerAddress, v)
}

func (v *buildUploadVerb) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	// see if we can find a git cl in the cwd
	typ := build.BuildBot
	if g, err := git.New("."); err != nil {
		log.E(ctx, "Git failed. Error: %v", err)
	} else {
		typ = build.User
		if cl, err := g.HeadCL(ctx); err != nil {
			log.E(ctx, "CL failed. Error: %v", err)
		} else {
			if v.CL == "" {
				// guess cl from git
				v.CL = cl.SHA.String()
				log.I(ctx, "Detected CL %s", v.CL)
			}
			if v.Description == "" {
				// guess description from git
				v.Description = cl.Subject
				log.I(ctx, "Detected description %s", v.Description)
			}
		}
		if status, err := g.Status(ctx); err != nil {
			log.E(ctx, "Status failed. Error: %v", err)
		} else {
			if !status.Clean() {
				typ = build.Local
			}
		}
		if v.Branch == "" {
			// guess branch from git
			if branch, err := g.CurrentBranch(ctx); err != nil {
				log.E(ctx, "Branch failed. Error: %v", err)
			} else {
				v.Branch = branch
				log.I(ctx, "Dectected branch %s", v.Branch)
			}
		}
	}
	if v.Uploader == "" {
		// guess uploader from environment
		if user, err := user.Current(); err == nil {
			v.Uploader = user.Username
			log.I(ctx, "Dectected uploader %s", v.Uploader)
		}
	}
	log.I(ctx, "Dectected build type %s", typ)
	v.store = build.NewRemote(ctx, conn)
	host := host.Instance(ctx)
	v.info = &build.Information{
		Type:        typ,
		Branch:      v.Branch,
		Cl:          v.CL,
		Tag:         v.Tag,
		Description: v.Description,
		Builder:     host,
		Uploader:    v.Uploader,
	}
	return nil
}

func (v *buildUploadVerb) process(ctx context.Context, id string) error {
	id, merged, err := v.store.Add(ctx, id, v.info)
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

type artifactSearchVerb struct {
	ServerAddress string `help:"The master server address"`
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
	ServerAddress string `help:"The master server address"`
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
	ServerAddress string `help:"The master server address"`
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
	Name          string `help:"The new name for the track"`
	Description   string `help:"A description of the track"`
	Pkg           string `help:"The id of the package at the head of the track"`
	ServerAddress string `help:"The master server address"`
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
