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

load("@gapid//tools/build/rules:cc.bzl", "cc_copts")

cc_library(
    name = "sqlite",
    srcs = glob(["*.c"]) + ["src/ext/misc/percentile.c"],
    hdrs = glob(["*.h"]),
    copts = cc_copts() + [
        "-Iexternal/perfetto/sqlite",
        "-DSQLITE_THREADSAFE=0",
        "-DQLITE_DEFAULT_MEMSTATUS=0",
        "-DSQLITE_LIKE_DOESNT_MATCH_BLOBS",
        "-DSQLITE_OMIT_DEPRECATED",
        "-DSQLITE_OMIT_SHARED_CACHE",
        "-DHAVE_USLEEP",
        "-DHAVE_UTIME",
        "-DSQLITE_BYTEORDER=1234",
        "-DSQLITE_DEFAULT_AUTOVACUUM=0",
        "-DSQLITE_DEFAULT_MMAP_SIZE=0",
        "-DSQLITE_CORE",
        "-DSQLITE_TEMP_STORE=3",
        "-DSQLITE_OMIT_LOAD_EXTENSION",
        "-DSQLITE_OMIT_RANDOMNESS",
        # Always build in optimized mode.
        "-O2",
        "-DNDEBUG",
    ],
    visibility = ["//visibility:public"],
)
