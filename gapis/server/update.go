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
	"fmt"

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

	var mostRecent *service.Release
	mostRecentVersion := app.Version
	mostRecentDevVersion := app.Version.GetDevVersion()

	for _, release := range releases {
		// Filter out pre-releases
		if release.GetPrerelease() {
			continue
		}

		// tagName is either:
		// Regular release: v<major>.<minor>.<point>
		// Dev pre-release: v<major>.<minor>.<point>-dev-<dev>
		var version app.VersionSpec
		tagName := release.GetTagName()
		devVersion := -1
		numFields, err := fmt.Sscanf(tagName, "v%d.%d.%d-dev-%d", &version.Major, &version.Minor, &version.Point, &devVersion)
		if err != nil && numFields < 3 {
			// some non-release tags have other format, e.g. libinterceptor-v1.0
			log.I(ctx, "Ignoring tag %s", tagName)
			continue
		}

		// dev-releases are previews of the next release, so e.g. 1.2.3 is more recent than 1.2.3-dev-456
		if version.GreaterThan(mostRecentVersion) ||
			(version.Equal(mostRecentVersion) && mostRecentDevVersion != -1 && mostRecentDevVersion < devVersion) {
			mostRecent = &service.Release{
				Name:         release.GetName(),
				VersionMajor: uint32(version.Major),
				VersionMinor: uint32(version.Minor),
				VersionPoint: uint32(version.Point),
				Prerelease:   release.GetPrerelease(),
				BrowserUrl:   release.GetHTMLURL(),
			}
			mostRecentVersion = version
			mostRecentDevVersion = devVersion
		}
	}
	if mostRecent == nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.NoNewBuildsAvailable(),
			Transient: true,
		}
	}
	return mostRecent, nil
}
