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

if [ $# -ne 1 -o ! -f "$1/pkg/build.properties" ]; then
	echo Expected the build folder as an argument.
	exit 1
fi

function absname {
  echo $(cd "$1" && pwd)
}

BUILD_OUT=$1
SRC=$(absname "$(dirname "${BASH_SOURCE[0]}")")

rm -rf "$BUILD_OUT/dist"
mkdir -p "$BUILD_OUT/dist"
pushd "$BUILD_OUT/dist"
VERSION=$(awk -F= 'BEGIN {major=0; minor=0; micro=0}
                  /Major/ {major=$2}
                  /Minor/ {minor=$2}
                  /Micro/ {micro=$2}
                  END {print major"."minor"."micro}' ../pkg/build.properties)

# Combine package contents.
mkdir -p gapid/DEBIAN gapid/opt/gapid gapid/usr/share/applications gapid/usr/share/menu gapid/usr/share/mime/packages
cp -r ../pkg/* gapid/opt/gapid
cp -r ../current/java/gapic-linux.jar gapid/opt/gapid/lib/gapic.jar
cp "$SRC/../../gapic/res/icons/logo_256.png" gapid/opt/gapid/icon.png
cp "$SRC/gapid.desktop" gapid/usr/share/applications/google-gapid.desktop
cp "$SRC/gapid.menu" gapid/usr/share/menu/google-gapid.menu
cp "$SRC/gapid-mime.xml" gapid/usr/share/mime/packages/gapid.xml

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
Installed-Size: $(du -s gapid/opt | cut -f1)
EOF

# Fix up permissions (root ownership is faked below).
find gapid/ -type f -exec chmod 644 {} +
chmod 755 gapid/opt/gapid/gapi[drst]
find gapid/ -type d -exec chmod 755 {} +
find gapid/ -type d -exec chmod g-s {} +

# Package up zip file.
cd gapid/opt/
zip -r ../../gapid-$VERSION-linux.zip gapid/
cd ../../

# Build the .deb package.
echo "$(date): Building package."
fakeroot dpkg-deb -v --build gapid
mv gapid.deb gapid-$VERSION-linux.deb
echo "$(date): Done."

popd
