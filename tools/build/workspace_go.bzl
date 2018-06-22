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
# After calling gapid_dependencies(), load @io_bazel_rules_go's
# go_repository and call this macro.
def gapid_go_dependencies():

    # TODO: Cannot override rules_go's golang protobuf. Would
    # cause a cycle. This may be fixed with
    # https://github.com/bazelbuild/rules_go/issues/1548
    #
    #_maybe(_github_go_repository,
    #    name = "com_github_golang_protobuf",
    #    organization = "golang",
    #    project = "protobuf",
    #    commit = "b4deda0973fb4c70b50d226b1af49f3da59f5265",
    #    importpath = "github.com/golang/protobuf",
    #)

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
        name = "org_golang_google_grpc",
        organization = "grpc",
        project = "grpc-go",
        commit = "50955793b0183f9de69bd78e2ec251cf20aab121",
        importpath = "google.golang.org/grpc",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_net",
        organization = "golang",
        project = "net",
        commit = "f2499483f923065a842d38eb4c7f1927e6fc6e6d",
        importpath = "golang.org/x/net",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_sys",
        organization = "golang",
        project = "sys",
        commit = "d75a52659825e75fff6158388dddc6a5b04f9ba5",
        importpath = "golang.org/x/sys",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_tools",
        organization = "golang",
        project = "tools",
        commit = "3da34b1b520a543128e8441cd2ffffc383111d03",
        importpath = "golang.org/x/tools",
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
