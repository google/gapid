#!/bin/bash
# Copyright (C) 2017 Google Inc.
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

# Linux Build Script.
set -ex

BUILD_ROOT=$PWD
SRC=$PWD/github/gapid/

# Get bazel.
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/0.12.0/bazel-0.12.0-without-jdk-installer-linux-x86_64.sh
mkdir bazel
bash bazel-0.12.0-without-jdk-installer-linux-x86_64.sh --prefix=$PWD/bazel

# Setup environment.
export ANDROID_NDK_HOME=/opt/android-ndk-r16b

cd $SRC

# Invoke the build.
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}
echo $(date): Starting build...
$BUILD_ROOT/bazel/bin/bazel build -c opt --config symbols \
    --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define GAPID_BUILD_SHA="$BUILD_SHA" \
    //:pkg //cmd/gapir/cc:gapir.sym
echo $(date): Build completed.

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out
