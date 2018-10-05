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

# MacOS Build Script.
set -ex

BUILD_ROOT=$PWD
SRC=$PWD/github/gapid/

# Setup the Android SDK and NDK
curl -L -k -O -s https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip
mkdir android
unzip -q tools_r25.2.3-macosx.zip -d android
echo y | ./android/tools/bin/sdkmanager build-tools\;26.0.1 platforms\;android-26
curl -L -k -O -s https://dl.google.com/android/repository/android-ndk-r16b-darwin-x86_64.zip
unzip -q android-ndk-r16b-darwin-x86_64.zip -d android
export ANDROID_HOME=$PWD/android
export ANDROID_NDK_HOME=$PWD/android/android-ndk-r16b

# Get bazel.
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/0.16.1/bazel-0.16.1-installer-darwin-x86_64.sh
mkdir bazel
sh bazel-0.16.1-installer-darwin-x86_64.sh --prefix=$PWD/bazel

# Specify the version of XCode
export DEVELOPER_DIR=/Applications/Xcode_8.2.app/Contents/Developer

cd $SRC

# Invoke the build.
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}
echo $(date): Starting build...
# "--strategy CppLink=local" disables the sandbox when linking, which is
# required for the symbol dumping to work, as the linker *always* puts absolute
# paths to the .a files into the debug section of the executable.
$BUILD_ROOT/bazel/bin/bazel build -c opt --config symbols \
    --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define GAPID_BUILD_SHA="$BUILD_SHA" \
    --strategy CppLink=local \
    //:pkg //cmd/gapir/cc:gapir.sym
echo $(date): Build completed.

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/macos/package.sh $BUILD_ROOT/out
