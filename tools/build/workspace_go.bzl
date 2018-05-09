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
load("@io_bazel_rules_go//go:def.bzl", "go_repository")

def _github_url_gen(name, organization, project, commit):
    return github_http_args(
        organization = organization,
        project = project,
        commit = commit,
        branch = "",
    )

# Defines the repositories for GAPID's go dependencies.
# After calling gapid_dependencies(), load @io_bazel_rules_go's
# go_repository and call this macro.
#  url_gen - allows the overriding of the generation of repository URLs.
#      Pass a function with the following parameters:
#          name - the name of the repository
#          organization - the github organization of the project.
#          project - the github project
#          commit - the commit sha
#      The function should return a struct with the following fields:
#          url - the url to fetch, which answers with a blob.
#          type - the type of the blob (see repository_ctx.download_and_extract)
#          strip_prefix - directory prefix to strip (see repository_ctx.download_and_extract)
def gapid_go_dependencies(url_gen = _github_url_gen):
    _maybe(_github_go_repository,
        name = "com_github_golang_protobuf",
        url_gen = url_gen,
        organization = "golang",
        project = "protobuf",
        commit = "8ee79997227bf9b34611aee7946ae64735e6fd93",
        importpath = "github.com/golang/protobuf",
    )

    _maybe(_github_go_repository,
        name = "com_github_google_go_github",
        url_gen = url_gen,
        organization = "google",
        project = "go-github",
        commit = "a89ea1cdf79929726a9416663609269ada774da0",
        importpath = "github.com/google/go-github",
    )

    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        url_gen = url_gen,
        organization = "google",
        project = "go-querystring",
        commit = "53e6ce116135b80d037921a7fdd5138cf32d7a8a",
        importpath = "github.com/google/go-querystring",
    )

    _maybe(_github_go_repository,
        name = "com_github_pkg_errors",
        url_gen = url_gen,
        organization = "pkg",
        project = "errors",
        commit = "248dadf4e9068a0b3e79f02ed0a610d935de5302",
        importpath = "github.com/pkg/errors",
    )

    _maybe(_github_go_repository,
        name = "org_golang_google_grpc",
        url_gen = url_gen,
        organization = "grpc",
        project = "grpc-go",
        commit = "50955793b0183f9de69bd78e2ec251cf20aab121",
        importpath = "google.golang.org/grpc",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        url_gen = url_gen,
        organization = "golang",
        project = "crypto",
        commit = "dc137beb6cce2043eb6b5f223ab8bf51c32459f4",
        importpath = "golang.org/x/crypto",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_net",
        url_gen = url_gen,
        organization = "golang",
        project = "net",
        commit = "f2499483f923065a842d38eb4c7f1927e6fc6e6d",
        importpath = "golang.org/x/net",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_sys",
        url_gen = url_gen,
        organization = "golang",
        project = "sys",
        commit = "d75a52659825e75fff6158388dddc6a5b04f9ba5",
        importpath = "golang.org/x/sys",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_tools",
        url_gen = url_gen,
        organization = "golang",
        project = "tools",
        commit = "3da34b1b520a543128e8441cd2ffffc383111d03",
        importpath = "golang.org/x/tools",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

def _github_go_repository(name, url_gen, organization, project, commit, **kwargs):
    url = url_gen(name, organization, project, commit)
    go_repository(
        name = name,
        urls = [url.url],
        type = url.type,
        strip_prefix = url.strip_prefix,
        **kwargs
    )
