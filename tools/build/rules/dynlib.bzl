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

# This is needed due to https://github.com/bazelbuild/bazel/issues/914

def cc_dynamic_library(name, visibility, **kwargs):
    # All but one of these will fail, but the select in the filegroup
    # will pick up the correct one.
    native.cc_binary(
        name = name + ".so",
        linkshared = 1,
        **kwargs
    )
    native.cc_binary(
        name = name + ".dylib",
        linkshared = 1,
        **kwargs
    )
    native.cc_binary(
        name = name + ".dll",
        linkshared = 1,
        **kwargs
    )

    native.filegroup(
        name = name,
        visibility = visibility,
        srcs = select({
            "//tools/build:linux": [":" + name + ".so"],
            "//tools/build:darwin": [":" + name + ".dylib"],
            "//tools/build:windows": [":" + name + ".dll"],
        })
    )
