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

cc_library(
    name = "spirv-internal",
    hdrs = glob(["include/spirv/**/*.h"]),
    strip_include_prefix = "include/",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "spirv-headers",
    hdrs = [
        "include/spirv/unified1/spirv.hpp",
    ],
    include_prefix = "third_party/SPIRV-Headers/",
    visibility = ["//visibility:public"],
    deps = [
        ":spirv-internal",
    ],
)
