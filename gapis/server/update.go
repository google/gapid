// Copyright (C) 2020 Google Inc.
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

package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"

	"github.com/google/go-github/github"
)

const (
	githubOrg     = "google"
	githubRepo    = "agi"
	devGithubRepo = "agi-dev-releases"
	angleURL      = "https://agi-angle.storage.googleapis.com/"
	angleJSON     = "current.json"
)

func checkForUpdates(ctx context.Context, includeDevReleases bool) (*service.Releases, error) {
	agi, err := checkForAGIUpdates(ctx, includeDevReleases)
	if err != nil {
		return nil, err
	}

	angle, err := checkForANGLEUpdates(ctx)
	if err != nil {
		return nil, err
	}

	return &service.Releases{
		AGI:   agi,
		ANGLE: angle,
	}, nil
}

func checkForAGIUpdates(ctx context.Context, includeDevReleases bool) (*service.Releases_AGIRelease, error) {
	client := github.NewClient(nil)
	options := &github.ListOptions{}
	releases, _, err := client.Repositories.ListReleases(ctx, githubOrg, githubRepo, options)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to list releases")
	}

	if includeDevReleases {
		devReleases, _, err := client.Repositories.ListReleases(ctx, githubOrg, devGithubRepo, options)
		if err != nil {
			return nil, log.Err(ctx, err, "Failed to list dev-releases")
		}
		releases = append(releases, devReleases...)
	}

	version, release := getLastestRelease(ctx, releases)
	if release != nil {
		return &service.Releases_AGIRelease{
			Name:         release.GetName(),
			VersionMajor: uint32(version.Major),
			VersionMinor: uint32(version.Minor),
			VersionPoint: uint32(version.Point),
			VersionDev:   uint32(version.GetDevVersion()),
			BrowserUrl:   release.GetHTMLURL(),
		}, nil
	}
	return nil, &service.ErrDataUnavailable{
		Reason:    messages.NoNewBuildsAvailable(),
		Transient: false,
	}
}

func getLastestRelease(ctx context.Context, releases []*github.RepositoryRelease) (app.VersionSpec, *github.RepositoryRelease) {
	var maxVersion app.VersionSpec
	var maxRelease *github.RepositoryRelease
	for _, release := range releases {
		// Filter out pre-releases
		if release.GetPrerelease() {
			continue
		}

		version, err := app.VersionSpecFromTag(release.GetTagName())
		if err != nil {
			log.I(ctx, "Ignoring tag %s", release.GetTagName())
			continue
		}

		if version.GreaterThanDevVersion(maxVersion) {
			maxVersion = version
			maxRelease = release
		}
	}
	return maxVersion, maxRelease
}

// This struct is used to parse JSON, so fields need to be public.
type angleVersion struct {
	Version uint32
	Arm32   string
	Arm64   string
	X86     string
}

func checkForANGLEUpdates(ctx context.Context) (*service.Releases_ANGLERelease, error) {
	base, _ := url.Parse(angleURL)

	jsonURL, _ := base.Parse(angleJSON)
	res, err := http.Get(jsonURL.String())
	if err != nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.ErrInternalError(err),
			Transient: true,
		}
	}
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.ErrInternalError(err),
			Transient: true,
		}
	}

	var version angleVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.ErrInternalError(err),
			Transient: false,
		}
	}

	arm32, _ := base.Parse(version.Arm32)
	arm64, _ := base.Parse(version.Arm64)
	x86, _ := base.Parse(version.X86)
	if arm32 == nil || arm64 == nil || x86 == nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.NoNewBuildsAvailable(),
			Transient: false,
		}
	}

	return &service.Releases_ANGLERelease{
		Version: version.Version,
		Arm_32:  arm32.String(),
		Arm_64:  arm64.String(),
		X86:     x86.String(),
	}, nil
}
