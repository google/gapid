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
        commit = "fee04e8d84e09e4404f740aef498cead5a9e56fc",  # 38.1.0
        importpath = "github.com/google/go-github",
        sha256 = "e6219a28e136689d5277c3f44139e6117f4eeeb3922c725bbf2e49b9e924777d",
    )

    # Dependency of com_github_google_go_github.
    _maybe(_github_go_repository,
        name = "com_github_google_go_querystring",
        organization = "google",
        project = "go-querystring",
        commit = "f76b16e611e840d52c50bd66dabeef5fe3b5e74d",
        importpath = "github.com/google/go-querystring",
        sha256 = "de4c24364adca4716fcb04213e62142c6530798ee691f9f2045204fd33af601b",
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
        commit = "41e044e1c82fcf6a5801d6cbd7ecf952505eecb1",  # 1.40.0
        importpath = "google.golang.org/grpc",
        sha256 = "e23f4f49f1a431baa8577b48ab8d2b9a0ce1ac76b03439210605f2d87ea19070",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_crypto",
        organization = "golang",
        project = "crypto",
        commit = "32db794688a5a24a23a43f2a984cecd5b3d8da58",
        importpath = "golang.org/x/crypto",
        sha256 = "f6f36d5112831f3e965aeb4f22856fb91a17ea156938f1a4b90d516fef95f2cd",
    )

    # Dependency of org_golang_x_tools.
    _maybe(_github_go_repository,
        name = "org_golang_x_mod",
        organization = "golang",
        project = "mod",
        commit = "6ce8bb3f08e0e47592fe93e007071d86dcf214bb",  # 0.4.1
        importpath = "golang.org/x/mod",
        sha256 = "7b5008b98e341459375f1de23007b09339d1fe5d6f3c5e4b4ed7d12a002471de",
    )

    _maybe(_github_go_repository,
        name = "org_golang_x_net",
        organization = "golang",
        project = "net",
        commit = "60bc85c4be6d32924bcfddb728394cb8713f2c78",
        importpath = "golang.org/x/net",
        sha256 = "7b87068a90d3f1092ec728c73a582d5221ce8a0838ea48f7b53b5654dcb8ec6b",
    )

    # Dependency of org_golang_x_net.
    _maybe(_github_go_repository,
        name = "org_golang_x_text",
        organization = "golang",
        project = "text",
        commit = "383b2e75a7a4198c42f8f87833eefb772868a56f",
        importpath = "golang.org/x/text",
        sha256 = "c9194e8839a8be0a29a53efc7fadfa0da54c35d0f0114fb705e1b2179fe4bb36",
    )

    # Dependency of org_golang_x_mod.
    _maybe(_github_go_repository,
        name = "org_golang_x_xerrors",
        organization = "golang",
        project = "xerrors",
        commit = "5ec99f83aff198f5fbd629d6c8d8eb38a04218ca",
        importpath = "golang.org/x/xerrors",
        sha256 = "cd9de801daf63283be91a76d7f91e8a9541798c5c0e8bcfb7ee804b78a493b02",
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
