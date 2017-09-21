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
	"archive/zip"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/layout"
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
	CL           string `help:"The build CL, will be guessed if not set"`
	Description  string `help:"An optional build description"`
	Tag          string `help:"The optional build tag"`
	Track        string `help:"The package's track, will be guessed if not set"`
	Uploader     string `help:"The uploading entity, will be guessed if not set"`
	BuilderAbi   string `help:"The abi of the builder device, will assume this device if not set"`
	ArtifactPath string `help:"The file path where the zipped artifact will be stored"`
}

func init() {
	uploadVerb.Add(&app.Verb{
		Name:       "build",
		ShortHelp:  "Upload a build to the server",
		ShortUsage: "<filenames>",
		Action:     &buildUploadVerb{UploadOptions: UploadOptions{RobotOptions: defaultRobotOptions}},
	})
	uploadVerb.Add(&app.Verb{
		Name:       "package",
		ShortHelp:  "Package and upload a build to the server",
		ShortUsage: "<filename>",
		Action:     &packageUploadVerb{UploadOptions: UploadOptions{RobotOptions: defaultRobotOptions}},
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

func (v *buildUploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return upload(ctx, flags, v.ServerAddress, v)
}

func (v *buildUploadVerb) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	v.store = build.NewRemote(ctx, conn)
	v.info = v.UploadOptions.createBuildInfo(ctx)
	return nil
}

func (v *buildUploadVerb) process(ctx context.Context, id string) error {
	return storeBuild(ctx, v.store, v.info, id)
}

type packageUploadVerb struct {
	UploadOptions

	store build.Store
	info  *build.Information
}

func (v *packageUploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if v.ArtifactPath == "" {
		if len(flags.Args()) != 1 {
			err := errors.New("Missing expected argument")
			return log.Err(ctx, err, "`do robot upload package` expects a single filepath as argument")
		}
		log.I(ctx, "Running packageUploadVerb, artifact arg is %s", flags.Args()[0])
		v.ArtifactPath = flags.Args()[0]
		log.I(ctx, "artifact path is %s", v.ArtifactPath)
	}

	return upload(ctx, flags, v.ServerAddress, v)
}

func (v *packageUploadVerb) prepare(ctx context.Context, conn *grpc.ClientConn) error {
	if err := zipArtifacts(ctx, file.Abs(v.ArtifactPath)); err != nil {
		return err
	}
	v.store = build.NewRemote(ctx, conn)
	v.info = v.UploadOptions.createBuildInfo(ctx)
	return nil
}

func (v *packageUploadVerb) process(ctx context.Context, id string) error {
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

func (o *UploadOptions) createBuildInfo(ctx context.Context) *build.Information {
	// see if we can find a git cl in the cwd
	typ := build.BuildBot
	if g, err := git.New("."); err != nil {
		log.E(ctx, "Git failed. Error: %v", err)
	} else if o.CL != "" {
		if o.Track == "" {
			log.W(ctx, "Cannot detect track from CL, defaulting to auto")
			o.Track = "auto"
		}
	} else {
		typ = build.User
		if cl, err := g.HeadCL(ctx); err != nil {
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
		} else {
			if !status.Clean() {
				typ = build.Local
			}
		}
		if o.Track == "" {
			// guess branch from git
			if branch, err := g.CurrentBranch(ctx); err != nil {
				log.E(ctx, "Branch failed. Error: %v", err)
			} else {
				o.Track = branch
				log.I(ctx, "Detected track %s", o.Track)
			}
		}
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
	}
}

func zipFile(zipWriter *zip.Writer, zipVirtualPath string, filePath file.Path) error {
	fileReader, err := os.Open(filePath.String())
	if err != nil {
		return err
	}
	defer fileReader.Close()

	fileHeader, err := fileReader.Stat()
	if err != nil {
		return err
	}

	zipHeader, err := zip.FileInfoHeader(fileHeader)
	if err != nil {
		return err
	}
	zipHeader.Name = zipVirtualPath

	zipFile, err := zipWriter.CreateHeader(zipHeader)
	if err != nil {
		return err
	}

	_, err = io.Copy(zipFile, fileReader)
	if err != nil {
		return err
	}

	return nil
}

func zipArtifacts(ctx context.Context, artifactFile file.Path) error {
	outputZipFile, err := os.Create(artifactFile.String())
	if err != nil {
		return err
	}
	artifacts := zip.NewWriter(outputZipFile)
	defer artifacts.Close()

	toolSetPathFunc := map[string]func(context.Context) (file.Path, error){
		"gapid/gapis": layout.Gapis,
		"gapid/gapit": layout.Gapit,
		"gapid/gapir": layout.Gapir,
		"gapid/libVkLayer_VirtualSwapchain.so": func(ctx context.Context) (file.Path, error) {
			return layout.Library(ctx, layout.LibVirtualSwapChain)
		},
		"gapid/VirtualSwapchainLayer.json": func(ctx context.Context) (file.Path, error) {
			return layout.Json(ctx, layout.LibVirtualSwapChain)
		},
	}
	for toolName, pathFunc := range toolSetPathFunc {
		path, err := pathFunc(ctx)
		if err != nil {
			return log.Errf(ctx, err, "Couldn't get layout path for tool %s", toolName)
		}
		if err := zipFile(artifacts, toolName, path); err != nil {
			return log.Errf(ctx, err, "Failed to Zip the tool %s at path %s", toolName, path)
		}
	}

	// TODO(baldwinn): these hardcoded architectures come from core/app/layout/layout.go, move this to a better place
	androidAbiList := []*device.ABI{
		device.AndroidARMv7a,
		device.AndroidARM64v8a,
		device.AndroidX86,
	}
	for _, abi := range androidAbiList {
		gapidApkPath, err := layout.GapidApk(ctx, abi)
		zipApkPath, err := layout.BinLayout(file.Abs("/gapid/")).GapidApk(ctx, abi)
		zipApkVirtualPath, err := zipApkPath.RelativeTo(file.Abs("/"))
		if err != nil || !gapidApkPath.Exists() {
			continue
		}
		if err := zipFile(artifacts, filepath.ToSlash(zipApkVirtualPath), gapidApkPath); err != nil {
			return log.Errf(ctx, err, "Failed to Zip the gapid.apk for abi %s at path %s", abi.Name, gapidApkPath)
		}
	}

	return nil
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
