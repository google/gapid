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

# Get bazel.
BAZEL_VERSION=2.0.0
curl -L -k -O -s https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh
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
curl -L -k -s https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y clang-format-6.0

# Setup environment.
export ANDROID_NDK_HOME=/opt/android-ndk-r16b
export BAZEL=$BUILD_ROOT/bazel/bin/bazel
export BUILDIFIER=$BUILD_ROOT/tools/bin/buildifier
export BUILDOZER=$BUILD_ROOT/tools/bin/buildozer
export CLANG_FORMAT=clang-format-6.0

cd $SRC

. ./kokoro/presubmit/presubmit.sh
