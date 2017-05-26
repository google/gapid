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

# Get GO 1.8 - Works around the cgo race condition issues.
curl -L -k -O -s https://storage.googleapis.com/golang/go1.8.darwin-amd64.tar.gz
tar -xzf go1.8.darwin-amd64.tar.gz

# Setup GO paths (remove old, add new).
export GOROOT=$BUILD_ROOT/go
export PATH=${PATH/\/usr\/local\/go\/bin:/}
export PATH=$PATH:$GOROOT/bin

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
    "CMakePath": "/Applications/CMake.app/Contents/bin/cmake",
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
mkdir -p gapid/jre
cp -r ../pkg/* gapid/
cp -r ../current/java/gapic-osx.jar gapid/lib/gapic.jar
$SRC/kokoro/macos/copy_jre.sh gapid/jre
cp $SRC/kokoro/macos/gapid.sh gapid/gapid

# Create a zip file.
zip -r gapid-$VERSION-macos.zip gapid/

# Create a .app package
mkdir -p GAPID.app/Contents/MacOS/
cp -r gapid/* GAPID.app/Contents/MacOS/
cp $SRC/kokoro/macos/Info.plist GAPID.app/Contents/

# Create the icon. TODO: need resolution up to 1024 (512@2x)
mkdir -p GAPID.iconset GAPID.app/Contents/Resources
# Ensure the icon has an alpha channel to make iconutil work, sigh.
pip install --user pypng
python -c '
import sys;import png;i=png.Reader(sys.stdin).asRGBA();
png.Writer(width=i[0],height=i[1],alpha=True).write(sys.stdout,i[2])'\
  < $SRC/gapic/res/icons/gapid/logo\@2x.png > logo.png
for i in 128 64 32 16; do
  sips -z $i $i logo.png --out GAPID.iconset/icon_${i}x$i.png
  sips -z $((i*2)) $((i*2)) logo.png --out GAPID.iconset/icon_${i}x$i\@2x.png
done
iconutil -c icns -o GAPID.app/Contents/Resources/GAPID.icns GAPID.iconset

# Make a dmg file.
pip install --user dmgbuild
cp $SRC/kokoro/macos/background\@2x.png .
# Yes, height, then width.... sigh.
sips -z 480 640 background\@2x.png --out background.png
cp $SRC/kokoro/macos/dmg-settings.py .
~/Library/Python/2.7/bin/dmgbuild -s dmg-settings.py GAPID gapid-$VERSION.dmg
