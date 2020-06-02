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
SRC=$PWD/github/agi/

# Setup the Android SDK and NDK
# Note: the SDK manager needs Java8, call it before switching to Java11
curl -L -k -O -s https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip
mkdir android
unzip -q tools_r25.2.3-macosx.zip -d android
echo y | ./android/tools/bin/sdkmanager build-tools\;29.0.2 platforms\;android-26
curl -L -k -O -s https://dl.google.com/android/repository/android-ndk-r21-darwin-x86_64.zip
unzip -q android-ndk-r21-darwin-x86_64.zip -d android
export ANDROID_HOME=$PWD/android
export ANDROID_NDK_HOME=$PWD/android/android-ndk-r21

# Get Zulu JDK11 from bazel, see https://mirror.bazel.build/openjdk/index.html
ZULU_JDK="zulu11.31.11-ca-jdk11.0.3"
curl -L -k -O -s https://mirror.bazel.build/openjdk/azul-${ZULU_JDK}/${ZULU_JDK}-macosx_x64.tar.gz
echo "98df91fa49f16b73dbc09e153628190640ff6c3fac2322b8142bc00077a0f738  ${ZULU_JDK}-macosx_x64.tar.gz" | sha256sum --check
tar xzf ${ZULU_JDK}-macosx_x64.tar.gz
export JAVA_HOME=${PWD}/${ZULU_JDK}-macosx_x64

# Get bazel.
BAZEL_VERSION=2.0.0
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-darwin-x86_64.sh
mkdir bazel
sh bazel-${BAZEL_VERSION}-installer-darwin-x86_64.sh --prefix=$PWD/bazel

# Specify the version of XCode
export DEVELOPER_DIR=/Applications/Xcode_11.3.app/Contents/Developer

cd $SRC
BUILD_SHA=${DEV_PREFIX}${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function build {
  echo $(date): Starting build for $@...
  # "--strategy CppLink=local" disables the sandbox when linking, which is
  # required for the symbol dumping to work, as the linker *always* puts absolute
  # paths to the .a files into the debug section of the executable.
  $BUILD_ROOT/bazel/bin/bazel \
      --output_base="${TMP}/bazel_out" \
      build -c opt --config symbols \
      --define AGI_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
      --define AGI_BUILD_SHA="$BUILD_SHA" \
      --strategy CppLink=local \
      $@
  echo $(date): Build completed.
}

# Build each API package separately first, as the go-compiler needs ~8GB of
# RAM for each of the big API packages.
build //gapis/api/vulkan:go_default_library

# Build the package and symbol file.
build //:pkg //:symbols

# Build and run the smoketests.
build //cmd/smoketests:smoketests
echo $(date): Run smoketests...
# Using "bazel run //cmd/smoketests seems to make 'bazel-bin/pkg/gapit'
# disappear, hence we call the binary directly
bazel-bin/cmd/smoketests/darwin_amd64_stripped/smoketests -gapit bazel-bin/pkg/gapit -traces test/traces
echo $(date): Smoketests completed.

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/macos/package.sh $BUILD_ROOT/out
