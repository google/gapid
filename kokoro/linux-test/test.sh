#!/bin/bash
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

# Linux Build Script.
set -ex

BUILD_ROOT=$PWD
SRC=$PWD/github/agi/
CURL="curl -fksLS --http1.1 --retry 3"

# Get bazel.
BAZEL_VERSION=4.2.0
$CURL -O https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh
echo "396d7031134889647351b9caf947811ecee5c2c1c5772e01b0916439067667fd  bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh" | sha256sum --check
mkdir bazel
bash bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh --prefix=$PWD/bazel

# Get GCC 8.
sudo rm /etc/apt/sources.list.d/cuda.list*
sudo add-apt-repository -y ppa:ubuntu-toolchain-r/test
sudo apt-get -q update
sudo apt-get -qy install gcc-8 g++-8
export CC=/usr/bin/gcc-8

# Get the Android NDK.
$CURL -O https://dl.google.com/android/repository/android-ndk-r21d-linux-x86_64.zip
echo "dd6dc090b6e2580206c64bcee499bc16509a5d017c6952dcd2bed9072af67cbd  android-ndk-r21d-linux-x86_64.zip" | sha256sum --check
unzip -q android-ndk-r21d-linux-x86_64.zip
export ANDROID_NDK_HOME=$PWD/android-ndk-r21d

# Get recent build tools.
echo y | $ANDROID_HOME/tools/bin/sdkmanager --install 'build-tools;30.0.3'

cd $SRC
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function test {
    $BUILD_ROOT/bazel/bin/bazel \
        --output_base="${TMP}/bazel_out" \
        test -c opt --config symbols \
        --define AGI_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
        --define AGI_BUILD_SHA="$BUILD_SHA" \
        --test_tag_filters=-needs_gpu \
        --show_timestamps \
        $@
}

# Running all the tests in one go leads to an out-of-memory error on Kokoro, hence the division in smaller test sets
test tests-core
test tests-gapis-api
test tests-gapis-replay-resolve
test tests-gapis-other
test tests-gapir
test tests-gapil
test tests-general
