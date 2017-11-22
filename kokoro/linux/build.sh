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
SRC=$PWD/github/src/github.com/google/gapid/

# Get NINJA.
wget -q https://github.com/ninja-build/ninja/releases/download/v1.7.2/ninja-linux.zip
unzip -q ninja-linux.zip

# Get GO 1.8.3
GO_ARCHIVE=go1.8.3.linux-amd64.tar.gz
wget -q https://storage.googleapis.com/golang/$GO_ARCHIVE
tar -xzf $GO_ARCHIVE

# Setup GO paths (remove old, add new).
export GOROOT=$BUILD_ROOT/go
export PATH=${PATH//:\/usr\/local\/go\/bin/}
export PATH=${PATH//:\/usr\/local\/go\/packages\/bin/}
export PATH=$GOROOT/bin:$PATH

# Setup the build config file.
cat <<EOF>gapid-config
{
    "Flavor": "release",
    "OutRoot": "$BUILD_ROOT/out",
    "JavaHome": "$JAVA_HOME",
    "AndroidSDKRoot": "$ANDROID_HOME",
    "AndroidNDKRoot": "$ANDROID_HOME/ndk-bundle",
    "CMakePath": "/usr/bin/cmake",
    "NinjaPath": "$BUILD_ROOT/ninja",
    "PythonPath": "/usr/bin/python",
    "MSYS2Path": ""
}
EOF
cat gapid-config
cp gapid-config $SRC/.gapid-config

# Fetch the submodules.
cd $SRC
git submodule update --init

# Invoke the build. At this point, only ensure that the tests build, but don't
# execute the tests.
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}
echo $(date): Starting build...
./do build --test build --buildnum $KOKORO_BUILD_NUMBER --buildsha "$BUILD_SHA"
echo $(date): Build completed.

# Build the release packages.
$SRC/kokoro/linux/package.sh $BUILD_ROOT/out

# Clean up - this prevents kokoro from rsyncing many unneeded files
shopt -s extglob
cd $BUILD_ROOT
rm -rf github/src/github.com/google/gapid/third_party
rm -rf out/release
rm -rf -- !(github|out)
