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
SRC=$PWD/github/gapid/

# Get bazel
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/0.25.1/bazel-0.25.1-installer-linux-x86_64.sh
mkdir bazel
bash bazel-0.25.1-installer-linux-x86_64.sh --prefix=$PWD/bazel

# Get GCC 7
sudo add-apt-repository -y ppa:ubuntu-toolchain-r/test
sudo apt-get -q update
sudo apt-get -qy install gcc-7 g++-7
export CC=/usr/bin/gcc-7

cd $SRC
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

echo $(date): Tests started.
$BUILD_ROOT/bazel/bin/bazel \
    --output_base="${TMP}/bazel_out" \
    test tests -c opt --config symbols \
    --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define GAPID_BUILD_SHA="$BUILD_SHA" \
    --test_tag_filters=-needs_gpu
echo $(date): Tests completed.
