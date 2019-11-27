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

# Get the Android NDK
curl -L -k -O -s https://dl.google.com/android/repository/android-ndk-r20b-linux-x86_64.zip
unzip -q android-ndk-r20b-linux-x86_64.zip
export ANDROID_NDK_HOME=$PWD/android-ndk-r20b

# Get recent build tools
echo y | $ANDROID_HOME/tools/bin/sdkmanager --install 'build-tools;29.0.2'

# Install dependencies
sudo apt-get -qy install libvulkan1 libvulkan-dev xvfb ffmpeg
# Install swiftshader
mkdir $BUILD_ROOT/swiftshader
cd $BUILD_ROOT/swiftshader
SWIFTSHADER_SHA=663dcefa22ea5eec1b108feebaf40a683fb104ff
SWIFTSHADER_URL="https://github.com/google/gfbuild-swiftshader/releases/download/github%2Fgoogle%2Fgfbuild-swiftshader%2F${SWIFTSHADER_SHA}/gfbuild-swiftshader-${SWIFTSHADER_SHA}-Linux_x64_Release.zip"
curl -L -o swiftshader.zip ${SWIFTSHADER_URL}
unzip swiftshader.zip
export VK_ICD_FILENAMES=$BUILD_ROOT/swiftshader/lib/vk_swiftshader_icd.json
export VK_LOADER_DEBUG=all

cd $SRC
BUILD_SHA=${DEV_PREFIX}${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function build {
  echo $(date): Starting build for $@...
  $BUILD_ROOT/bazel/bin/bazel \
    --output_base="${TMP}/bazel_out" \
    build -c opt --config symbols \
    --define GAPID_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define GAPID_BUILD_SHA="$BUILD_SHA" \
    $@
  echo $(date): Build completed.
}

# Build each API package separately first, as the go-compiler needs ~8GB of
# RAM for each of the big API packages.
for api in gles vulkan gvr; do
  build //gapis/api/$api:go_default_library
done

# Build the package and symbol file.
build //:pkg //cmd/gapir/cc:gapir.sym

# Build the Vulkan sample
build //cmd/vulkan_sample:vulkan_sample

# Build and run the smoketests.
set +e
build //cmd/smoketests:smoketests //cmd/vulkan_sample:vulkan_sample
echo $(date): Run smoketests...
# Using "bazel run //cmd/smoketests seems to make 'bazel-bin/pkg/gapit'
# disappear, hence we call the binary directly
bazel-bin/cmd/smoketests/linux_amd64_stripped/smoketests -gapit bazel-bin/pkg/gapit -traces test/traces
for i in smoketests.*/*/*.log
do
  echo "============================================================"
  echo $i
  cat $i
done
echo $(date): Smoketests completed.

# End-to-end smoketest: trace and replay vulkan_sample on top of swiftshader

cd $SRC
echo $(date): Run trace and replay of vulkan_sample
./bazel-bin/pkg/gapit trace -api vulkan -start-at-frame 2 -capture-frames 10 -observe-frames 1 -out vulkan_sample.gfxtrace -uri `which xvfb-run` --additionalargs "-e xfvb.log -a ./bazel-bin/cmd/vulkan_sample/vulkan_sample"
killall vulkan_sample
./bazel-bin/pkg/gapit video -type sxs vulkan_sample.gfxtrace
echo $(date): Run trace and replay of vulkan_sample

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out
