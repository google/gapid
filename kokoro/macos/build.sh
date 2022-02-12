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
CURL="curl -fksLS --http1.1 --retry 3"

# Android SDK Manager requires an older version of Java
# Therefore, temporarily downgrade java
export JAVA_HOME=`/usr/libexec/java_home -v 1.8`

# Setup the Android SDK and NDK.
$CURL -O https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip
echo "593544d4ca7ab162705d0032fb0c0c88e75bd0f42412d09a1e8daa3394681dc6  tools_r25.2.3-macosx.zip" | shasum --check
mkdir android
unzip -q tools_r25.2.3-macosx.zip -d android
echo y | ./android/tools/bin/sdkmanager build-tools\;30.0.3 platforms\;android-26
$CURL -O https://dl.google.com/android/repository/android-ndk-r21d-darwin-x86_64.zip
echo "5851115c6fc4cce26bc320295b52da240665d7ff89bda2f5d5af1887582f5c48  android-ndk-r21d-darwin-x86_64.zip" | shasum --check
unzip -q android-ndk-r21d-darwin-x86_64.zip -d android
export ANDROID_HOME=$PWD/android
export ANDROID_NDK_HOME=$PWD/android/android-ndk-r21d

# Get the JDK from our mirror.
JDK_BUILD=zulu11.39.15-ca
JDK_VERSION=11.0.7
JDK_NAME=$JDK_BUILD-jdk$JDK_VERSION-macosx_x64
JRE_NAME=$JDK_BUILD-jre$JDK_VERSION-macosx_x64
$CURL -O https://storage.googleapis.com/jdk-mirror/$JDK_BUILD/$JDK_NAME.zip
echo "e5ea71dd86eefe6e2ef78720ea729f26a8b5c279bcb2f1770745698ef374f9b8  $JDK_NAME.zip" | shasum --check
unzip -q $JDK_NAME.zip
export JAVA_HOME=$PWD/$JDK_NAME/zulu-11.jdk/Contents/Home

$CURL -O https://storage.googleapis.com/jdk-mirror/$JDK_BUILD/$JRE_NAME.zip
echo "d5f40f9a221816e3f4c3219ac658d184d8cb4f99c7a1fb19f4ffc45d88bafd73  $JRE_NAME.zip" | shasum --check
unzip -q $JRE_NAME.zip
export JRE_HOME=$PWD/$JRE_NAME/zulu-11.jre/Contents/Home

# Get bazel.
BAZEL_VERSION=4.2.0
$CURL -O https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-darwin-x86_64.sh
echo "ee86e5bcf8661af7a08ea49378db1977bedb9a391841158b0610d27c4f601ad1  bazel-${BAZEL_VERSION}-installer-darwin-x86_64.sh" | shasum --check
mkdir bazel
sh bazel-${BAZEL_VERSION}-installer-darwin-x86_64.sh --prefix=$PWD/bazel

# Specify the version of XCode.
export DEVELOPER_DIR=/Applications/Xcode_12.4.app/Contents/Developer

cd $SRC
BUILD_SHA=${DEV_PREFIX}${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function run_bazel {
  ACTION=$1
  shift
  # "--strategy CppLink=local" disables the sandbox when linking, which is
  # required for the symbol dumping to work, as the linker *always* puts absolute
  # paths to the .a files into the debug section of the executable.
  $BUILD_ROOT/bazel/bin/bazel \
      --output_base="${TMP}/bazel_out" \
      $ACTION -c opt \
      --define AGI_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
      --define AGI_BUILD_SHA="$BUILD_SHA" \
      --strategy CppLink=local \
      --show_timestamps \
      $@
}

# Build each API package separately first, as the go-compiler needs ~8GB of
# RAM for each of the big API packages.
run_bazel build //gapis/api/vulkan:go_default_library

# Build the package and symbol file.
run_bazel build //:pkg

# Build and run the smoketests.
run_bazel run //cmd/smoketests:smoketests -- --traces test/traces

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/macos/package.sh $BUILD_ROOT/out
