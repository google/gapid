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
        commit = "2e3e74fa920e7d6278b7a9737ab5c8b7b1480294",  # 29.0.3
        importpath = "github.com/google/go-github",
        sha256 = "5c72eae85dd971e31776cc97e3cac7e57acdef253d896d649d9d520f7b990ea4",
    )

    # Dependency of com_github_google_go_github.
    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        organization = "google",
        project = "go-querystring",
        commit = "c8c88dbee036db4e4808d1f2ec8c2e15e11c3f80",
        importpath = "github.com/google/go-querystring",
        sha256 = "be509de2d315358db459f40262ca34d7dceb0d59d4119addf880562b10710853",
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
        commit = "142182889d38b76209f1d9f1d8e91d7608aff542",  # 1.28.0
        importpath = "google.golang.org/grpc",
        sha256 = "f969e1c33b79d4c03527b8163f257f50257ac9dcb488859182a548ea39724a4d",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        organization = "golang",
        project = "crypto",
        commit = "1b76d66859c6111b3d5c3ea6600ea44dc188bf12",
        importpath = "golang.org/x/crypto",
        sha256 = "daaec7016b1d81d05505bd50534ef62b9fe3cf367b39ad805b14cec62e3648f3",
    )

    # Dependency of org_golang_x_tools.
    _maybe(_github_go_repository,
        name = "org_golang_x_mod",
        organization = "golang",
        project = "mod",
        commit = "e5e73c1b9c72835114eb6daab038373d39515006",
        importpath = "golang.org/x/mod",
        sha256 = "5e727c7ec77372e0a37fc535e81d6f8b9423bdcbc66ee506994d3ac2c3ce704b",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_net",
        organization = "golang",
        project = "net",
        commit = "244492dfa37ae2ce87222fd06250a03160745faa",
        importpath = "golang.org/x/net",
        sha256 = "d3d90167ae827ee4a2fc2db7ce23142410974da41ca2e95f8832b57facdc7190",
    )

    # Dependency of org_golang_x_net.
    _maybe(_github_go_repository,
        name = "org_golang_x_text",
        organization = "golang",
        project = "text",
        commit = "06d492aade888ab8698aad35476286b7b555c961",
        importpath = "golang.org/x/text",
        sha256 = "d0076e2957c45a9ded9853bd398257f4d8fd8fe74d3953a7049a0474629b78a2",
    )

    # Dependency of org_golang_x_mod.
    _maybe(_github_go_repository,
        name = "org_golang_x_xerrors",
        organization = "golang",
        project = "xerrors",
        commit = "9bdfabe68543c54f90421aeb9a60ef8061b5b544",
        importpath = "golang.org/x/xerrors",
        sha256 = "757fe99de4d23e10a3343e9790866211ecac0458c5268da43e664a5abeee27e3",
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
