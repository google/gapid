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

lsb_release -a

# Linux Build Script.
#set -ex
set -x

BUILD_ROOT=$PWD
SRC=$PWD/github/gapid

# Get bazel
BAZEL_VERSION=1.2.0
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh
mkdir bazel
bash bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh --prefix=$PWD/bazel

# Get GCC 8
sudo rm /etc/apt/sources.list.d/cuda.list*
sudo add-apt-repository -y ppa:ubuntu-toolchain-r/test
sudo apt-get -q update
sudo apt-get -qy install gcc-8 g++-8
export CC=/usr/bin/gcc-8

######################################################################
# Try our vulkan sample

# use xvfb to have virtual framebuffer
# alternative: use gapir to replay in headless, and use this replay as an app.
# store binaries, if need be, could be done on GCP.


cd $SRC
$BUILD_ROOT/bazel/bin/bazel build -c dbg //cmd/vulkan_sample

export VK_ICD_FILENAMES=$SRC/tools/build/third_party/swiftshader/vk_swiftshader_icd.json
export VK_LOADER_DEBUG=all

sudo apt-get -qy install libvulkan1 libvulkan-dev xvfb
Xvfb +extension GLX :99 &
pid_xvfb=$!
export DISPLAY=:99

(
  catchsegv $SRC/bazel-bin/cmd/vulkan_sample/vulkan_sample &
  pid=$!
  sleep 1
  kill $pid
) 2>&1 | tee _log

$SRC/bazel-bin/cmd/vulkan_sample/vulkan_sample &
pid_cube=$!

sleep 2

kill $pid_cube





kill $pid_xvfb

exit 1

######################################################################

BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function test {
    echo $(date): Starting test for $@...
    $BUILD_ROOT/bazel/bin/bazel \
        --output_base="${TMP}/bazel_out" \
        test -c opt --config symbols \
        --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
        --define GAPID_BUILD_SHA="$BUILD_SHA" \
        --test_tag_filters=-needs_gpu \
        $@
    echo $(date): Tests completed.
}

# Running all the tests in one go leads to an out-of-memory error on Kokoro, hence the division in smaller test sets
test tests-core
test tests-gapis-api
test tests-gapis-replay-resolve
test tests-gapis-other
test tests-gapir
test tests-gapil
test tests-general
