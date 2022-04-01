# Copyright (C) 2022 Google Inc.
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

def _fuchsia_config_impl(ctx):
  dir = "@gapid//tools/build/fuchsia:disabled"
  if ctx.os.environ.get("AGI_FUCHSIA_BUILD") == "1":
    dir = "@gapid//tools/build/fuchsia:enabled"

  ctx.file("BUILD.bazel", content = "", executable = False)
  ctx.symlink(Label(dir + "/fuchsia_sdk.bzl"), "fuchsia_sdk.bzl")
  ctx.symlink(Label(dir + "/workspace.bzl"), "workspace.bzl")


_fuchsia_config = repository_rule(
  implementation = _fuchsia_config_impl,
  environ = [ "AGI_FUCHSIA_BUILD" ],
  local = True,
  configure = True,
)

def fuchsia_config():
  _fuchsia_config(
    name = "local_config_fuchsia",
  )
