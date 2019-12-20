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

# Exposes the "user_local_repos" macro, which can be used in the WORKSPACE
# file to allow GAPID developers to override 3rd party repositories with local
# versions. A developer can create a "user.locals" file in their WORKSPACE root
# which contains lines in the following format:
#
#   # comments are ignored
#   repo_name: /path/to/override  # comment
#
# Alternatively, the path to the overrides file can be set via the GAPID_LOCALS
# environment variable. Note: if using the environment variable, the path has
# to be an absolute path.

def _user_local_repos_impl(ctx):
  path = ctx.os.environ.get("GAPID_LOCALS")
  if path == None:
    path = ctx.path(ctx.attr.dir).get_child("user.locals")
  else:
    path = ctx.path(path)

  locals = {}
  if path.exists:
    for line in ctx.read(path).splitlines():
      idx = line.find("#")
      if idx >= 0:
        line = line[:idx]
      line = line.strip()
      if len(line) == 0:
        continue
      toks = line.split(":")
      if len(toks) != 2:
        fail(str(path) + ": Invalid syntax (expected <name>: <path>) on line: " + line)
      locals[toks[0].strip()] = toks[1].strip()

  if len(locals) > 0:
    print("Using the following local overrides:")
    for name in locals:
      print("  " + name + ": " + locals[name])

  ctx.file("BUILD.bazel")
  ctx.file("locals.bzl",
    content = "LOCALS = " + str(locals),
    executable = False,
  )


_user_local_repos = repository_rule(
  implementation = _user_local_repos_impl,
  attrs = {
    "dir": attr.string(),
  },
  environ = [
    "GAPID_LOCALS",
  ],
  local = True,
  configure = True,
)

def user_local_repos(dir):
  _user_local_repos(
    name = "user_locals",
    dir = dir,
  )
