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
    )

    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        organization = "google",
        project = "go-querystring",
        commit = "53e6ce116135b80d037921a7fdd5138cf32d7a8a",
        importpath = "github.com/google/go-querystring",
    )

    _maybe(_github_go_repository,
        name = "com_github_pkg_errors",
        organization = "pkg",
        project = "errors",
        commit = "248dadf4e9068a0b3e79f02ed0a610d935de5302",
        importpath = "github.com/pkg/errors",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        organization = "golang",
        project = "crypto",
        commit = "1a580b3eff7814fc9b40602fd35256c63b50f491",
        importpath = "golang.org/x/crypto",
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
