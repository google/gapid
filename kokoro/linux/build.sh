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
CURL="curl -fksLS --http1.1 --retry 3"

# Get bazel.
BAZEL_VERSION=5.2.0
$CURL -O https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh
echo "7d9ef51beab5726c55725fb36675c6fed0518576d3ba51fb4067580ddf7627c4  bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh" | sha256sum --check
mkdir bazel
bash bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh --prefix=$PWD/bazel

# Get Clang-12.
sudo add-apt-repository 'deb http://apt.llvm.org/xenial/  llvm-toolchain-xenial-12 main'
sudo apt-get update
sudo apt-get install -y clang-12
export CC=/usr/bin/clang-12

# Upgrade libstdc++6 for Swiftshader
sudo add-apt-repository ppa:ubuntu-toolchain-r/test
sudo apt-get update
sudo apt-get upgrade -y libstdc++6

# Get the Android NDK.
$CURL -O https://dl.google.com/android/repository/android-ndk-r21d-linux-x86_64.zip
echo "dd6dc090b6e2580206c64bcee499bc16509a5d017c6952dcd2bed9072af67cbd  android-ndk-r21d-linux-x86_64.zip" | sha256sum --check
unzip -q android-ndk-r21d-linux-x86_64.zip
export ANDROID_NDK_HOME=$PWD/android-ndk-r21d

# Get recent build tools.
echo y | $ANDROID_HOME/tools/bin/sdkmanager --install 'build-tools;30.0.3'

# Get the JDK from our mirror.
JDK_BUILD=zulu11.39.15-ca
JDK_VERSION=11.0.7
JDK_NAME=$JDK_BUILD-jdk$JDK_VERSION-linux_x64
$CURL -O https://storage.googleapis.com/jdk-mirror/$JDK_BUILD/$JDK_NAME.zip
echo "afbaa594447596a7fcd78df4ee59436ee19b43e27111e2e5a21a3272a89074cf  $JDK_NAME.zip" | sha256sum --check
unzip -q $JDK_NAME.zip
export JAVA_HOME=$PWD/$JDK_NAME

cd $SRC
BUILD_SHA=${DEV_PREFIX}${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}

function run_bazel {
  local ACTION=$1
  shift
  $BUILD_ROOT/bazel/bin/bazel \
    --output_base="${TMP}/bazel_out" \
    $ACTION -c opt --config symbols \
    --define AGI_BUILD_NUMBER="$KOKORO_BUILD_NUMBER" \
    --define AGI_BUILD_SHA="$BUILD_SHA" \
    --show_timestamps \
    $@
}

# Build the Vulkan API package separately first, as the go-compiler needs ~8GB
# of RAM for this
run_bazel build //gapis/api/vulkan:go_default_library

# Build the package and symbol file.
run_bazel build //:pkg //:symbols

# Build the Vulkan sample
run_bazel build //cmd/vulkan_sample:vulkan_sample

# Build and run the smoketests.
run_bazel run //cmd/smoketests:smoketests -- -traces test/traces

# Build the release packages.
mkdir $BUILD_ROOT/out
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out

###############################################################################
## Build is done, run some tests

##
## Swarming tests, see test/swarming/README.md
##

# "Swarming" is a tool from Chromium LUCI, install using the recommended CIPD
# from depot_tool
export LUCI_ROOT="`pwd`/luci"
mkdir -p ${LUCI_ROOT}

git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="`pwd`/depot_tools:$PATH"

# You can get valid git revision by looking up infra/tools/... at:
# https://chrome-infra-packages.appspot.com/
# These corresponds to git revision of the infra repo, where luci-go commits
# are regularly rolled out:
# https://chromium.googlesource.com/infra/infra
# Note that the '${platform}' must appear as-is in the ensure file,
# hence the single quotes.
(
  INFRA_GIT_REVISION=3e441e9ac2b55457d69c37136b5d6f02d380da79
  echo 'infra/tools/luci/isolate/${platform} git_revision:'"${INFRA_GIT_REVISION}"
  echo 'infra/tools/luci/swarming/${platform} git_revision:'"${INFRA_GIT_REVISION}"
) > ${LUCI_ROOT}/ensure_file.txt
cipd ensure -ensure-file ${LUCI_ROOT}/ensure_file.txt -root ${LUCI_ROOT}

# Initialize some environment variables, unless they have already been set
# (e.g. by build-nightly.sh)
if [ -z "${SWARMING_TIMESTAMP}" ] ; then
  export SWARMING_TIMESTAMP=`date '+%Y%m%d-%H%M%S'`
fi

if [ -z "${SWARMING_TASK_PREFIX}" ] ; then
  export SWARMING_TASK_PREFIX="Kokoro_PR${KOKORO_GITHUB_PULL_REQUEST_NUMBER}"
fi

if [ -z "${SWARMING_TEST_DIR}" ] ; then
  # By default, use the "tests" directory which is loaded by common.cfg
  export SWARMING_TEST_DIR=${KOKORO_GFILE_DIR}/tests
fi

export SWARMING_AUTH_FLAG="--service-account-json=${KOKORO_KEYSTORE_DIR}/74894_kokoro_swarming_access_key"

# Prepare Swarming files
SWARMING_DIR=${SRC}/test/swarming
cp -r bazel-bin/pkg ${SWARMING_DIR}/agi
cp -r ${SWARMING_TEST_DIR} ${SWARMING_DIR}/tests

# Swarming environment

# Trigger the tests. Record trigger failures, but continue to trigger other
# tests and collect their results, such that a single trigger failure does
# not ruin a whole Swarming run.
SWARMING_TRIGGER_ERROR=0
pushd ${SWARMING_DIR}
for t in tests/* ; do
  set +e
  ./trigger.py --prefix ${SWARMING_TASK_PREFIX} ${t}
  EXIT_CODE=$?
  set -e
  if [ ${EXIT_CODE} -ne 0 ] ; then
    echo "Swarming trigger error on test: ${TEST_NAME}"
    SWARMING_TRIGGER_ERROR=1
  fi
done
popd

# Run the swiftshader test while Swarming tests are being scheduled+run, and
# collect Swarming test results after the Swiftshader test.

##
## Test capture and replay of the Vulkan Sample App.
##

# Install the Vulkan loader and xvfb (X virtual framebuffer).
sudo apt-get -qy install libvulkan1 xvfb

# Get prebuilt SwiftShader.
# This is the latest commit at the time of writing.
# Should be updated periodically.
$CURL -o swiftshader.zip https://github.com/google/gfbuild-swiftshader/releases/download/github%2Fgoogle%2Fgfbuild-swiftshader%2F0bbf7ba9f909092f0328b1d519d5f7db1773be57/gfbuild-swiftshader-0bbf7ba9f909092f0328b1d519d5f7db1773be57-Linux_x64_Debug.zip
echo "0b9fc77c469da6f047df6bf2b9103350551c93cde21eee5d51013c1cda046619  swiftshader.zip" | sha256sum --check
unzip -d swiftshader swiftshader.zip

# Use SwiftShader.
export VK_ICD_FILENAMES="$(pwd)/swiftshader/lib/vk_swiftshader_icd.json"
# For extensive Vulkan loader logs, set to VK_LOADER_DEBUG=all
export VK_LOADER_DEBUG=warn

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
# TODO(b/144158856): workaround: force GAPIS idle timeout to 0 (infinite) to
# avoid build flakes due to b/144158856
xvfb-run -e xvfb.log -a bazel-bin/pkg/gapit trace -gapis-args "--idle-timeout 0" -device host -disable-coherentmemorytracker -disable-unknown-extensions -no-buffer -api vulkan -start-at-frame 5 -capture-frames 10 -observe-frames 1 -out vulkan_sample.gfxtrace bazel-bin/cmd/vulkan_sample/vulkan_sample

xvfb-run -e xvfb.log -a bazel-bin/pkg/gapit video -gapis-args "--idle-timeout 0" -gapir-nofallback -type sxs -frames-minimum 10 -out vulkan_sample.mp4  vulkan_sample.gfxtrace

##
## Collect swarming test results
##

# By convention, results are stored in ${KOKORO_ARTIFACTS_DIR}/swarming/results.json

pushd ${SWARMING_DIR}

SWARMING_FAILURE=0
for TEST_NAME in triggered/*/*.json ; do
  set +e
  ./collect.py ${SWARMING_TIMESTAMP} ${KOKORO_GIT_COMMIT} `basename ${TEST_NAME} .json` ${TEST_NAME} ${KOKORO_ARTIFACTS_DIR}/swarming/results.json
  EXIT_CODE=$?
  set -e
  if [ ${EXIT_CODE} -eq 0 ] ; then
    echo "PASS ${TEST_NAME}"
  else
    echo "FAIL ${TEST_NAME}"
    SWARMING_FAILURE=1
  fi
done

popd

if [ ${SWARMING_FAILURE} -eq 1 ] ; then
  echo "Error: some Swarming test failed"
#  exit 1
fi

if [ ${SWARMING_TRIGGER_ERROR} -eq 1 ] ; then
  echo "Error: could not trigger some Swarming tests"
#  exit 1
fi
