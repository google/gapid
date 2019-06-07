# Copyright (C) 2019 Google Inc.
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

# Repository rule to download, extract and setup Perfetto and all it's
# dependencies.

PACKAGES = [
    struct(
        url = "https://android.googlesource.com/platform/external/perfetto/+archive/a8e2b77ed47d432d99edb6ddaaa7e5b61d3f2041.tar.gz",
        sha = "",
        strip = "",
        out = ".",
    ),
    struct(
        url = "https://storage.googleapis.com/perfetto/sqlite-amalgamation-3250300.zip",
        sha = "2ad5379f3b665b60599492cc8a13ac480ea6d819f91b1ef32ed0e1ad152fafef",
        strip = "sqlite-amalgamation-3250300",
        out = "third_party/sqlite",
    ),
    struct(
        url = "https://storage.googleapis.com/perfetto/sqlite-src-3250300.zip",
        sha = "c7922bc840a799481050ee9a76e679462da131adba1814687f05aa5c93766421",
        strip = "sqlite-src-3250300",
        out = "third_party/sqlite/src",
    ),
]

def _perfetto_impl(ctx):
    for pkg in PACKAGES:
        ctx.download_and_extract(
            url = pkg.url,
            output = pkg.out,
            sha256 = pkg.sha,
            stripPrefix = pkg.strip,
        )

    ctx.symlink(Label("@gapid//tools/build/third_party/perfetto:perfetto.BUILD"), "BUILD.bazel")
    ctx.symlink(Label("@gapid//tools/build/third_party/perfetto:sqlite.BUILD"), "third_party/sqlite/BUILD.bazel")

    # Link protos into a bazel C++ friendly namespace.
    ctx.symlink("protos/perfetto", "perfetto")

perfetto = repository_rule(
    implementation = _perfetto_impl,
)
