#!/bin/bash
# Copyright (C) 2018 Google Inc.
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

# Presubmit Checks Build Script.
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

# Get bazel build tools.
mkdir tools
export GOPATH=$PWD/tools
go get github.com/bazelbuild/buildtools/buildifier github.com/bazelbuild/buildtools/buildozer

# Get clang-format.
sudo rm /etc/apt/sources.list.d/cuda.list*
sudo add-apt-repository "deb http://apt.llvm.org/trusty/ llvm-toolchain-trusty-6.0 main"
sudo add-apt-repository "deb http://ppa.launchpad.net/ubuntu-toolchain-r/test/ubuntu trusty main"
$CURL -O https://apt.llvm.org/llvm-snapshot.gpg.key
echo "ce6eee4130298f79b0e0f09a89f93c1bc711cd68e7e3182d37c8e96c5227e2f0  llvm-snapshot.gpg.key" | sha256sum --check
sudo apt-key add llvm-snapshot.gpg.key
sudo apt-get update
sudo apt-get install -y clang-format-6.0

# Get recent Android build tools.
echo y | $ANDROID_HOME/tools/bin/sdkmanager --install 'build-tools;30.0.3'

# Python Format tool
python3 -m pip install autopep8==1.6.0 --user

# Setup environment.
export ANDROID_NDK_HOME=/opt/android-ndk-r16b
export BAZEL=$BUILD_ROOT/bazel/bin/bazel
export BUILDIFIER=$BUILD_ROOT/tools/bin/buildifier
export BUILDOZER=$BUILD_ROOT/tools/bin/buildozer
export CLANG_FORMAT=clang-format-6.0
export AUTOPEP8=~/.local/bin/autopep8

cd $SRC

. ./kokoro/presubmit/presubmit.sh
