# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Defines macros to be called from a WORKSPACE file to setup the GAPID
# go dependencies.

load("@gapid//tools/build/rules:repository.bzl", "github_http_args")
load("@bazel_gazelle//:deps.bzl", "go_repository")

# Defines the repositories for GAPID's go dependencies.
# After calling gapid_dependencies(), load @bazel_gazelle's
# go_repository and call this macro.
def gapid_go_dependencies():
    _maybe(_github_go_repository,
        name = "com_github_google_go_github",
        organization = "google",
        project = "go-github",
        commit = "a89ea1cdf79929726a9416663609269ada774da0",
        importpath = "github.com/google/go-github",
        sha256 = "a7b046d3c50362738d7e535ff1315df94021fd246337101021737a708fd7449d",
    )

    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        organization = "google",
        project = "go-querystring",
        commit = "53e6ce116135b80d037921a7fdd5138cf32d7a8a",
        importpath = "github.com/google/go-querystring",
        sha256 = "d600db9461f2e0ce73b9c7a40ea598e0e128a00db5bf0b731b40585a6851cb12",
    )

    _maybe(_github_go_repository,
        name = "com_github_pkg_errors",
        organization = "pkg",
        project = "errors",
        commit = "248dadf4e9068a0b3e79f02ed0a610d935de5302",
        importpath = "github.com/pkg/errors",
        sha256 = "d19f68fe315e0f06fa050e6b39704da9968b8cad7c6e436d1baee6c647ed7d04",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        organization = "golang",
        project = "crypto",
        commit = "1a580b3eff7814fc9b40602fd35256c63b50f491",
        importpath = "golang.org/x/crypto",
        sha256 = "80b56f1fb9f3f03c3ebd155b6e62c77a5f1309aaa7e747a9e6ff8e560ffd904e",
    )


def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

def _github_go_repository(name, organization, project, commit, **kwargs):
    github = github_http_args(
        organization = organization,
        project = project,
        commit = commit,
        branch = "",
    )
    go_repository(
        name = name,
        urls = [ github.url ],
        type = github.type,
        strip_prefix = github.strip_prefix,
        **kwargs
    )
