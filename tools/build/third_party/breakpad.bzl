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

    if repository_ctx.os.name.startswith("windows"):
      # Patch up breakpad on windows and add the dump_syms src.
      repository_ctx.symlink(Label("@gapid//tools/build/third_party/breakpad:windows.patch"), "windows.patch")
      repository_ctx.symlink(Label("@gapid//tools/build/third_party/breakpad:dump_syms_pe.cc"), "src/tools/windows/dump_syms/dump_syms_pe.cc")

      bash_exe = repository_ctx.os.environ["BAZEL_SH"] if "BAZEL_SH" in repository_ctx.os.environ else "c:/tools/msys64/usr/bin/bash.exe"
      result = repository_ctx.execute([bash_exe, "--login", "-c",
           "cd \"{}\" && /usr/bin/patch -p1 -i windows.patch".format(repository_ctx.path("."))])
      if result.return_code:
        fail("Failed to apply patch: (%d)\n%s" % (result.return_code, result.stderr))

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
