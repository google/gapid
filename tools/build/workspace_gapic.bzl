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
# Java client dependencies.

load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository")
load("@gapid//tools/build/third_party:jface.bzl", "jface")
load("@gapid//tools/build/third_party:swt.bzl", "swt")

# Defines the repositories for GAPID's Java client's dependencies.
#  locals - can be used to provide local path overrides for repos:
#     {"foo": "/path/to/foo"} would cause @foo to be a local repo based on /path/to/foo.
def gapic_dependencies(locals = {}):
    maybe_repository(
        github_repository,
        name = "com_github_grpc_java",
        locals = locals,
        organization = "grpc",
        project = "grpc-java",
        commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
        build_file = "@gapid//tools/build/third_party:grpc_java.BUILD",
    )

    maybe_repository(
        native.maven_jar,
        name = "com_google_guava",
        locals = locals,
        artifact = "com.google.guava:guava:27.0-jre",
        sha1 = "c6ad87d2575af8ac8ec38e28e75aefa882cc3a1f",
    )

    maybe_repository(
        swt,
        name = "swt",
        locals = locals,
    )

    maybe_repository(
        jface,
        name = "jface",
        locals = locals,
    )
