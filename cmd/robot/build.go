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
	"github.com/google/gapid/core/data/search/script"
	"github.com/google/gapid/core/git"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/test/robot/build"
	"google.golang.org/grpc"
)

var buildFlags struct {
	// upload flags
	tag         string
	cl          string
	branch      string
	description string
	uploader    string
	name        string
	pkg         string
}

func init() {
	buildUpload := &app.Verb{
		Name:       "build",
		ShortHelp:  "Upload a build to the server",
		ShortUsage: "<filenames>",
		Run:        doUpload(&buildUploader{}),
	}
	buildUpload.Flags.Raw.StringVar(&buildFlags.tag, "tag", "", "The optional build tag")
	buildUpload.Flags.Raw.StringVar(&buildFlags.cl, "cl", "", "The build CL, will be guessed if not set")
	buildUpload.Flags.Raw.StringVar(&buildFlags.branch, "branch", "", "The build branch, will be guessed if not set")
	buildUpload.Flags.Raw.StringVar(&buildFlags.description, "description", "", "An optional build description")
	buildUpload.Flags.Raw.StringVar(&buildFlags.uploader, "uploader", "", "The uploading entity, will be guessed if not set")
	uploadVerb.Add(buildUpload)
	artifactSearch := &app.Verb{
		Name:       "artifact",
		ShortHelp:  "List build artifacts in the server",
		ShortUsage: "<query>",
		Run:        doArtifactSearch,
	}
	searchVerb.Add(artifactSearch)
	packageSearch := &app.Verb{
		Name:       "package",
		ShortHelp:  "List build packages in the server",
		ShortUsage: "<query>",
		Run:        doPackageSearch,
	}
	searchVerb.Add(packageSearch)
	trackSearch := &app.Verb{
		Name:       "track",
		ShortHelp:  "List build tracks in the server",
		ShortUsage: "<query>",
		Run:        doTrackSearch,
	}
	searchVerb.Add(trackSearch)
	trackSet := &app.Verb{
		Name:       "track",
		ShortHelp:  "Sets values on a track",
		ShortUsage: "<id or name>",
		Run:        doTrackUpdate,
	}
	trackSet.Flags.Raw.StringVar(&buildFlags.name, "name", "", "The new name for the track")
	trackSet.Flags.Raw.StringVar(&buildFlags.description, "description", "", "A description of the track")
	trackSet.Flags.Raw.StringVar(&buildFlags.pkg, "package", "", "The id of the package at the head of the track")
	setVerb.Add(trackSet)
}

type buildUploader struct {
	store build.Store
	info  *build.Information
}

func (u *buildUploader) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	// see if we can find a git cl in the cwd
	typ := build.BuildBot
	if g, err := git.New("."); err != nil {
		log.E(ctx, "Git failed. Error: %v", err)
	} else {
		typ = build.User
		if cl, err := g.HeadCL(ctx); err != nil {
			log.E(ctx, "CL failed. Error: %v", err)
		} else {
			if buildFlags.cl == "" {
				// guess cl from git
				buildFlags.cl = cl.SHA.String()
				log.I(ctx, "Detected CL %s", buildFlags.cl)
			}
			if buildFlags.description == "" {
				// guess description from git
				buildFlags.description = cl.Subject
				log.I(ctx, "Detected description %s", buildFlags.description)
			}
		}
		if status, err := g.Status(ctx); err != nil {
			log.E(ctx, "Status failed. Error: %v", err)
		} else {
			if !status.Clean() {
				typ = build.Local
			}
		}
		if buildFlags.branch == "" {
			// guess branch from git
			if branch, err := g.CurrentBranch(ctx); err != nil {
				log.E(ctx, "Branch failed. Error: %v", err)
			} else {
				buildFlags.branch = branch
				log.I(ctx, "Dectected branch %s", buildFlags.branch)
			}
		}
	}
	if buildFlags.uploader == "" {
		// guess uploader from environment
		if user, err := user.Current(); err == nil {
			buildFlags.uploader = user.Username
			log.I(ctx, "Dectected uploader %s", buildFlags.uploader)
		}
	}
	log.I(ctx, "Dectected build type %s", typ)
	u.store = build.NewRemote(ctx, conn)
	host := host.Instance(ctx)
	u.info = &build.Information{
		Type:        typ,
		Branch:      buildFlags.branch,
		Cl:          buildFlags.cl,
		Tag:         buildFlags.tag,
		Description: buildFlags.description,
		Builder:     host,
		Uploader:    buildFlags.uploader,
	}
	return nil
}

func (u *buildUploader) process(ctx context.Context, id string) error {
	id, merged, err := u.store.Add(ctx, id, u.info)
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

func doArtifactSearch(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
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

func doPackageSearch(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
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

func doTrackSearch(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
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

var (
	idOrName = script.MustParse("Id == $ or Name == $").Using("$")
)

func doTrackUpdate(ctx context.Context, flags flag.FlagSet) error {
	return grpcutil.Client(ctx, serverAddress, func(ctx context.Context, conn *grpc.ClientConn) error {
		b := build.NewRemote(ctx, conn)
		args := flags.Args()
		track := &build.Track{
			Name:        buildFlags.name,
			Description: buildFlags.description,
			Head:        buildFlags.pkg,
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
