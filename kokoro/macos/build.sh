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
curl -L -k -O -s https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip
mkdir android
unzip -q tools_r25.2.3-macosx.zip -d android
echo y | ./android/tools/bin/sdkmanager build-tools\;29.0.2 platforms\;android-26
curl -L -k -O -s https://dl.google.com/android/repository/android-ndk-r21-darwin-x86_64.zip
unzip -q android-ndk-r21-darwin-x86_64.zip -d android
export ANDROID_HOME=$PWD/android
export ANDROID_NDK_HOME=$PWD/android/android-ndk-r21

# Get the JDK and JRE from our mirror. This needs to be after the Android updates above (needs 1.8).
JDK_BUILD=zulu11.39.15-ca
JDK_VERSION=11.0.7
JDK_NAME=$JDK_BUILD-jdk$JDK_VERSION-macosx_x64
JRE_NAME=$JDK_BUILD-jre$JDK_VERSION-macosx_x64
curl -L -k -O -s https://storage.googleapis.com/jdk-mirror/$JDK_BUILD/$JDK_NAME.zip
echo "e5ea71dd86eefe6e2ef78720ea729f26a8b5c279bcb2f1770745698ef374f9b8  $JDK_NAME.zip" | shasum --check
unzip -q $JDK_NAME.zip
export JAVA_HOME=$PWD/$JDK_NAME/zulu-11.jdk/Contents/Home

curl -L -k -O -s https://storage.googleapis.com/jdk-mirror/$JDK_BUILD/$JRE_NAME.zip
echo "d5f40f9a221816e3f4c3219ac658d184d8cb4f99c7a1fb19f4ffc45d88bafd73  $JRE_NAME.zip" | shasum --check
unzip -q $JRE_NAME.zip
export JRE_HOME=$PWD/$JRE_NAME/zulu-11.jre/Contents/Home

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
