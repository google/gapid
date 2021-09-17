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

# Linux Package Script.
set -ex

if [ $# -ne 1 -o ! -d "$1" ]; then
	echo Expected the build folder as an argument.
	exit 1
fi

function absname {
  echo $(cd "$1" && pwd)
}

BUILD_OUT=$1
SRC=$(absname "$(dirname "${BASH_SOURCE[0]}")")
BIN=$SRC/../../bazel-bin

if [ ! -f "$BIN/pkg/build.properties" ]; then
  echo Unable to find pkg/build.properties in $BIN
  exit 1
fi

rm -rf "$BUILD_OUT/dist"
mkdir -p "$BUILD_OUT/dist"
pushd "$BUILD_OUT/dist"
VERSION=$(awk -F= 'BEGIN {major=0; minor=0; micro=0}
                  /Major/ {major=$2}
                  /Minor/ {minor=$2}
                  /Micro/ {micro=$2}
                  END {print major"."minor"."micro}' $BIN/pkg/build.properties)

# Combine package contents.
mkdir -p agi/DEBIAN agi/opt/agi agi/usr/share/applications agi/usr/share/menu agi/usr/share/mime/packages
cp -r $BIN/pkg/* agi/opt/agi
cp "$SRC/../../tools/logo/logo_256.png" agi/opt/agi/icon.png
cp "$SRC/gapid.desktop" agi/usr/share/applications/google-agi.desktop
cp "$SRC/gapid.menu" agi/usr/share/menu/google-agi.menu
cp "$SRC/gapid-mime.xml" agi/usr/share/mime/packages/agi.xml

# Create the dpkg config file.
cat > agi/DEBIAN/control <<EOF
Package: agi
Version: $VERSION
Section: development
Priority: optional
Architecture: amd64
Depends: openjdk-11-jre, libgtk-3-0, libwebkit2gtk-4.0-37
Maintainer: Google, Inc <gapid-team@google.com>
Description: Android Graphics Inspector is a collection of tools that allows you
 to inspect, tweak and replay calls from an application to a graphics driver.
 .
 AGI can trace any Android debuggable application, or if you have root access
 to the device any application can be traced.
Homepage: https://github.com/google/agi
Installed-Size: $(du -s agi/opt | cut -f1)
EOF

# Fix up permissions (root ownership is faked below).
find agi/ -type f -exec chmod 644 {} +
chmod 755 agi/opt/agi/agi agi/opt/agi/gapi[rst] agi/opt/agi/device-info
find agi/ -type d -exec chmod 755 {} +
find agi/ -type d -exec chmod g-s {} +

# Package up zip file.
cd agi/opt/
zip -r ../../agi-$VERSION-linux.zip agi/
cd ../../

# Build the .deb package.
echo "$(date): Building package."
fakeroot dpkg-deb -v --build agi
mv agi.deb agi-$VERSION-linux.deb
echo "$(date): Done."

# Copy the symbol file to the output.
# Warning: the name MUST be gapir-$VERSION-... , as this format is expected in our release script.
[ -f "$BIN/cmd/gapir/cc/gapir.sym" ] && cp "$BIN/cmd/gapir/cc/gapir.sym" gapir-$VERSION-linux.sym
[ -f "$BIN/gapidapk/android/apk/arm64-v8a_gapir.sym" ] && cp "$BIN/gapidapk/android/apk/arm64-v8a_gapir.sym" gapir-$VERSION-android-arm64-v8a.sym
[ -f "$BIN/gapidapk/android/apk/armeabi-v7a_gapir.sym" ] && cp "$BIN/gapidapk/android/apk/armeabi-v7a_gapir.sym" gapir-$VERSION-android-armeabi-v7a.sym
[ -f "$BIN/gapidapk/android/apk/x86_gapir.sym" ] && cp "$BIN/gapidapk/android/apk/x86_gapir.sym" gapir-$VERSION-android-x86.sym

popd
