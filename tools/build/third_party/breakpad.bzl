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

# Repository rule to download, extract and patch breakpad.

_BASE = "https://chromium.googlesource.com/breakpad/breakpad";

def _breakpad_impl(repository_ctx):
    repository_ctx.download_and_extract(
        url = _BASE + "/+archive/" + repository_ctx.attr.commit + ".tar.gz",
        output = ".",
    )
    repository_ctx.symlink(repository_ctx.attr.build_file, "BUILD")

breakpad = repository_rule(
    implementation = _breakpad_impl,
    attrs = {
        "commit": attr.string(mandatory = True),
        "build_file": attr.label(
            mandatory = True,
            allow_single_file = True,
        ),
    },
)
