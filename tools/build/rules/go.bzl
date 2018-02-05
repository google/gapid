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

load("//tools/build/rules:repository.bzl", "github_http_args")
load("@io_bazel_rules_go//go:def.bzl", "go_repository")

def github_go_repository(name, organization, project, commit="", branch="", path="", **kwargs):
  if path:
    print("Override with {}".format(path))
  else:
    github = github_http_args(
        organization = organization,
        project = project,
        commit = commit,
        branch = branch,
      )
    go_repository(
      name = name,
      urls = [github.url],
      type = github.type,
      strip_prefix = github.strip_prefix,
      **kwargs
    )
