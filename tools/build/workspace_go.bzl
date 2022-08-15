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
        commit = "8309da114464ce3686f71058945ff561a6189b24",  # 45.2.0
        importpath = "github.com/google/go-github",
        sha256 = "7f0bc1cfbe14cd6684385576a4835969f182200c784ab81db24ce0cfa934e2b7",
    )

    # Dependency of com_github_google_go_github.
    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        organization = "google",
        project = "go-querystring",
        commit = "934da1706275a38ef4d8647320f723db0c208595",
        importpath = "github.com/google/go-querystring",
        sha256 = "dec5519b2c4f3d8b2370d539d13ad22d688743df9a19b1fabc42217c00f02505",
    )

    _maybe(_github_go_repository,
        name = "com_github_pkg_errors",
        organization = "pkg",
        project = "errors",
        commit = "614d223910a179a466c1767a985424175c39b465",  # 0.9.1
        importpath = "github.com/pkg/errors",
        sha256 = "49c7041442cc15211ee85175c06ffa6520c298b1826ed96354c69f16b6cfd13b",
    )

    _maybe(_github_go_repository,
        name = "org_golang_google_grpc",
        organization = "grpc",
        project = "grpc-go",
        commit = "64174955202ffb5ea4122e25d1aaece49cc5a3ed",  # 1.48.0
        importpath = "google.golang.org/grpc",
        sha256 = "bd68eca5fd6d61f163a947184047375572a2e6976f13cb24ef05d02b4f4c5f2e",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        organization = "golang",
        project = "crypto",
        commit = "630584e8d5aaa1472863b49679b2d5548d80dcba",
        importpath = "golang.org/x/crypto",
        sha256 = "4f7dfb4db2a7f31711dfbbf6f299070501e34a6872d00750e2ba85661632fbde",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_net",
        organization = "golang",
        project = "net",
        commit = "3211cb9802344f37a243bc81d005fb6e97f4f8f5",
        importpath = "golang.org/x/net",
        sha256 = "cfd01f207c96eeeb5657b9fbf04cf2a41b4313379208c87a538d028b89838a24",
    )


def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

def _github_go_repository(name, organization, project, commit, **kwargs):
    github = github_http_args(
        organization = organization,
        project = project,
        commit = commit,
    )
    go_repository(
        name = name,
        urls = [ github.url ],
        type = github.type,
        strip_prefix = github.strip_prefix,
        **kwargs
    )
