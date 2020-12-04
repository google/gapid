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
)

func checkForUpdates(ctx context.Context, includeDevReleases bool) (*service.Release, error) {
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
		return &service.Release{
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
		Transient: true,
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
