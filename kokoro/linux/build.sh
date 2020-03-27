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
SRC=$PWD/github/agi/

# Get bazel.
BAZEL_VERSION=2.0.0
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
curl -L -k -O -s https://dl.google.com/android/repository/android-ndk-r21-linux-x86_64.zip
unzip -q android-ndk-r21-linux-x86_64.zip
export ANDROID_NDK_HOME=$PWD/android-ndk-r21

# Get recent build tools
echo y | $ANDROID_HOME/tools/bin/sdkmanager --install 'build-tools;29.0.2'

cd $SRC
BUILD_SHA=${DEV_PREFIX}${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function build {
  echo $(date): Starting build for $@...
  $BUILD_ROOT/bazel/bin/bazel \
    --output_base="${TMP}/bazel_out" \
    build -c opt --config symbols \
    --define AGI_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define AGI_BUILD_SHA="$BUILD_SHA" \
    $@
  echo $(date): Build completed.
}

# Build the Vulkan API package separately first, as the go-compiler needs ~8GB
# of RAM for this
build //gapis/api/vulkan:go_default_library

# Build the package and symbol file.
build //:pkg //:symbols

# Build the Vulkan sample
build //cmd/vulkan_sample:vulkan_sample

# Build and run the smoketests.
set +e
build //cmd/smoketests:smoketests
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

set -e

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out

###############################################################################
## Build is done, run some tests

##
## Test on a real device using swarming. APKs are stoed on x20, under:
## teams/android-graphics-tools/agi/kokoro/swarming/apk/*.apk
##

# Install LUCI
curl -fsSL -o luci-py.tar.gz https://chromium.googlesource.com/infra/luci/luci-py.git/+archive/0b027452e658080df1f174c403946914443d2aa6.tar.gz
mkdir luci-py
tar xzvf luci-py.tar.gz --directory luci-py
LUCI_CLIENT_ROOT="$PWD/luci-py/client"

# Credentials come from Keystore
SWARMING_AUTH_TOKEN_FILE=${KOKORO_KEYSTORE_DIR}/74894_kokoro_swarming_access_key

# Prepare task files
TASK_FILES_DIR=${SRC}/test/swarming/task-files
cp -r bazel-bin/pkg ${TASK_FILES_DIR}/agi
cp -r ${KOKORO_GFILE_DIR}/apk ${TASK_FILES_DIR}/

# Trigger task
AUTH_FLAG="--auth-service-account-json=$SWARMING_AUTH_TOKEN_FILE"
TASK_NAME="Kokoro_PR${KOKORO_GITHUB_PULL_REQUEST_NUMBER}"
ISOLATE_SERVER='https://chrome-isolated.appspot.com'
SWARMING_SERVER='https://chrome-swarming.appspot.com'
SWARMING_POOL='SkiaInternal'
DEVICE_TYPE="flame" # pixel4

$LUCI_CLIENT_ROOT/isolate.py archive $AUTH_FLAG --isolate-server $ISOLATE_SERVER --isolate ${SRC}/test/swarming/task.isolate --isolated task.isolated
ISOLATED_SHA=`sha1sum task.isolated | awk '{ print $1 }' `

# Priority: lower is more priority, defaults to 200: PR short test tasks should be of higher priority than the default
PRIORITY=100
# Hard timeout: maximum number of seconds for the task to terminate
HARD_TIMEOUT=300
# Expiration: number of seconds to wait for a bot to be available
EXPIRATION=600

$LUCI_CLIENT_ROOT/swarming.py trigger $AUTH_FLAG --swarming $SWARMING_SERVER --isolate-server $ISOLATE_SERVER --isolated $ISOLATED_SHA --task-name ${TASK_NAME} --dump-json task.json --dimension pool $SWARMING_POOL --dimension device_type "$DEVICE_TYPE" --priority=$PRIORITY --expiration=$EXPIRATION --hard-timeout=$HARD_TIMEOUT

# Collect the Swarming test results after the swiftshader tests, as swarming
# will take a few minutes to schedule+run the task anyway.

##
## Test capture and replay of the Vulkan Sample App.
##

# Install the Vulkan loader and xvfb (X virtual framebuffer).
sudo apt-get -qy install libvulkan1 xvfb

# Get prebuilt SwiftShader.
# This is the latest commit at the time of writing.
# Should be updated periodically.
curl -fsSL -o swiftshader.zip https://github.com/google/gfbuild-swiftshader/releases/download/github%2Fgoogle%2Fgfbuild-swiftshader%2F0bbf7ba9f909092f0328b1d519d5f7db1773be57/gfbuild-swiftshader-0bbf7ba9f909092f0328b1d519d5f7db1773be57-Linux_x64_Debug.zip
unzip -d swiftshader swiftshader.zip

# Use SwiftShader.
export VK_ICD_FILENAMES="$(pwd)/swiftshader/lib/vk_swiftshader_icd.json"
export VK_LOADER_DEBUG=all

# Just try running the app first.

# Allow non-zero exit status.
set +e

xvfb-run -e xvfb.log -a timeout --preserve-status -s INT -k 5 5 bazel-bin/cmd/vulkan_sample/vulkan_sample

APP_EXIT_STATUS="${?}"

set -e

if test -f xvfb.log; then
  cat xvfb.log
fi

# This line will exit with status 1 if the app's exit status
# was anything other than 130 (128+SIGINT).
test "${APP_EXIT_STATUS}" -eq 130

# TODO(https://github.com/google/gapid/issues/3163): The coherent memory
#  tracker must be disabled with SwiftShader for now.
xvfb-run -e xvfb.log -a bazel-bin/pkg/gapit trace -device host -disable-coherentmemorytracker -disable-pcs -disable-unknown-extensions -record-errors -no-buffer -api vulkan -start-at-frame 5 -capture-frames 10 -observe-frames 1 -out vulkan_sample.gfxtrace bazel-bin/cmd/vulkan_sample/vulkan_sample

xvfb-run -e xvfb.log -a bazel-bin/pkg/gapit video -gapir-nofallback -type sxs -frames-minimum 10 -out vulkan_sample.mp4  vulkan_sample.gfxtrace

##
## Collect swarming test result
##

# The "swarming.py collect" call returns the task's exit code, which is non-zero
# if the task has expired (it was never scheduled). Allow for non-zero return
# code, and manually check the task status afterward
set +e
$LUCI_CLIENT_ROOT/swarming.py collect $AUTH_FLAG --swarming $SWARMING_SERVER --json task.json --task-summary-json summary.json
SWARMING_COLLECT_EXIT_CODE=$?
set -e

# Ignore failures that are not due to the test itself
if [ "$SWARMING_COLLECT_EXIT_CODE" -ne "0" ] ; then
  if grep '"state": "EXPIRED"' summary.json > /dev/null ; then
    echo "Swarming test was never scheduled, ignoring it"
  elif grep '"internal_failure": true' summary.json > /dev/null ; then
    echo "Swarming internal failure, ignore the swarming test"
  else
    echo "Swarming test failed"
    exit 1
  fi
fi
