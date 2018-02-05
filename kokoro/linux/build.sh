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

# Setup environment.
export ANDROID_NDK_HOME=/opt/android-ndk-r15c

cd $SRC

# Invoke the build.
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}
echo $(date): Starting build...
bazel build -c opt --strip always --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" --define GAPID_BUILD_SHA="$BUILD_SHA" //:pkg
echo $(date): Build completed.

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out
