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
SRC=$PWD/github/src/github.com/google/gapid/

# Get NINJA.
curl -L -k -O -s https://github.com/ninja-build/ninja/releases/download/v1.7.2/ninja-mac.zip
unzip -q ninja-mac.zip

# Get GO 1.8.3
GO_ARCHIVE=go1.8.3.darwin-amd64.tar.gz
curl -L -k -O -s https://storage.googleapis.com/golang/$GO_ARCHIVE
tar -xzf $GO_ARCHIVE

# Setup GO paths (remove old, add new).
export GOROOT=$BUILD_ROOT/go
export PATH=${PATH//:\/usr\/local\/go\/bin/}
export PATH=${PATH//:\/usr\/local\/go\/packages\/bin/}
export PATH=$GOROOT/bin:$PATH

# Setup the Android SDK and NDK
curl -L -k -O -s https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip
mkdir android
unzip -q tools_r25.2.3-macosx.zip -d android
echo y | ./android/tools/bin/sdkmanager build-tools\;21.1.2 platforms\;android-21 ndk-bundle

# Create the debug keystore, so gradle won't try to and fail due to a race.
mkdir -p ~/.android
$(/usr/libexec/java_home -v 1.8)/bin/keytool -genkey -keystore ~/.android/debug.keystore \
  -storepass android -alias androiddebugkey -keypass android -keyalg RSA -keysize 2048 \
  -validity 10950 -dname "CN=Android Debug,O=Android,C=US"


# Setup the build config file.
cat <<EOF>gapid-config
{
    "Flavor": "release",
    "OutRoot": "$BUILD_ROOT/out",
    "JavaHome": "$(/usr/libexec/java_home -v 1.8)",
    "AndroidSDKRoot": "$BUILD_ROOT/android",
    "AndroidNDKRoot": "$BUILD_ROOT/android/ndk-bundle",
    "CMakePath": "/usr/local/bin/cmake",
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

# Disable ccache as it seems to be flaky on mac build server
export CCACHE_DISABLE=1

# Specify the version of XCode
export DEVELOPER_DIR=/Applications/Xcode_8.2.app/Contents/Developer

# Invoke the build. At this point, only ensure that the tests build, but don't
# execute the tests.
BUILD_SHA=${KOKORO_GITHUB_COMMIT:-$KOKORO_GITHUB_PULL_REQUEST_COMMIT}
echo $(date): Starting build...
./do build --test build --buildnum $KOKORO_BUILD_NUMBER --buildsha "$BUILD_SHA"
echo $(date): Build completed.

# Build the release packages.
$SRC/kokoro/macos/package.sh $BUILD_ROOT/out

# Clean up - this prevents kokoro from rsyncing many unneeded files
shopt -s extglob
cd $BUILD_ROOT
rm -rf github/src/github.com/google/gapid/third_party
rm -rf out/release
rm -rf -- !(github|out)
