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

# Get GO 1.8 - Works around the cgo race condition issues.
wget -q https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
tar -xzf go1.8.linux-amd64.tar.gz

# Setup GO paths (remove old, add new).
export GOROOT=$BUILD_ROOT/go
export PATH=${PATH/\/usr\/local\/go\/bin:/}
export PATH=${PATH/\/usr\/local\/go\/packages\/bin:/}
export PATH=$PATH:$GOROOT/bin

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
mkdir $BUILD_ROOT/out/dist
cd $BUILD_ROOT/out/dist
VERSION=$(awk -F= 'BEGIN {major=0; minor=0; micro=0}
                  /Major/ {major=$2}
                  /Minor/ {minor=$2}
                  /Micro/ {micro=$2}
                  END {print major"."minor"."micro}' ../pkg/build.properties)

# Combine package contents.
mkdir -p gapid/DEBIAN gapid/opt/gapid
cp -r ../pkg/* gapid/opt/gapid
cp -r ../current/java/gapic-linux.jar gapid/opt/gapid/lib/gapic.jar
cp $SRC/kokoro/linux/gapid.sh gapid/opt/gapid/gapid

# Create the dpkg config file.
cat > gapid/DEBIAN/control <<EOF
Package: gapid
Version: $VERSION
Section: development
Priority: optional
Architecture: amd64
Depends: openjdk-8-jre
Maintainer: Google, Inc <gapid-team@google.com>
Description: GAPID is a collection of tools that allows you to inspect, tweak
 and replay calls from an application to a graphics driver.
 .
 GAPID can trace any Android debuggable application, or if you have root access
 to the device any application can be traced.
Homepage: https://github.com/google/gapid
EOF

# Fix up permissions and ownership.
chmod 755 gapid/opt/gapid/gapi[drst]
chmod 644 gapid/opt/gapid/lib/gapic.jar
find gapid/ -type d -exec chmod 755 {} +
find gapid/ -type d -exec chmod g-s {} +
sudo chown -R root.root gapid/*

# Package up zip file.
cd gapid/opt/
zip -r ../../gapid-$VERSION-linux.zip gapid/
cd ../../

# Build the .deb package.
echo "$(date): Building package."
dpkg-deb -v --build  gapid
mv gapid.deb gapid-$VERSION.deb
echo "$(date): Done."

# Clean up - this prevents kokoro from rsyncing many unneeded files
shopt -s extglob
cd $BUILD_ROOT
rm -rf github/src/github.com/google/gapid/third_party
rm -rf out/release
rm -rf -- !(github|out)
