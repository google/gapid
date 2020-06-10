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

# Creates a distributable copy of the JRE.

if [ $# -ne 1 ]; then
	echo Expected the destination folder as an argument.
	exit
fi

cp -r $(/usr/libexec/java_home -v 1.8)/jre/ $1/

# Remove unnecessary files.
# See http://www.oracle.com/technetwork/java/javase/jre-8-readme-2095710.html
rm -f $1/THIRDPARTYLICENSEREADME-JAVAFX.txt

rm -f $1/bin/jjs
rm -f $1/bin/keytool
rm -f $1/bin/orbd
rm -f $1/bin/pack200
rm -f $1/bin/policytool
rm -f $1/bin/rmid
rm -f $1/bin/rmiregistry
rm -f $1/bin/servertool
rm -f $1/bin/tnameserv
rm -f $1/bin/unpack200

rm -rf $1/lib/deploy/
rm -f $1/lib/ant?javafx.jar
rm -f $1/lib/deploy.jar
rm -f $1/lib/javafx.properties
rm -f $1/lib/javaws.jar
rm -rf $1/lib/jfr/
rm -f $1/lib/jfr.jar
rm -f $1/lib/jfxswt.jar
rm -f $1/lib/libdecora*.*
rm -f $1/lib/libdeploy.dylib
rm -f $1/lib/libfxplugins.dylib
rm -f $1/lib/libglass.dylib
rm -f $1/lib/libglib?lite.dylib
rm -f $1/lib/libgstreamer?lite.dylib
rm -f $1/lib/libjava?crw?demo.dylib
rm -f $1/lib/libjavafx*.dylib
rm -f $1/lib/libjfr.dylib
rm -f $1/lib/libjfxmedia*.dylib
rm -f $1/lib/libjfxwebkit.dylib
rm -f $1/lib/libprism?common.dylib
rm -f $1/lib/libprism?sw.dylib
rm -f $1/lib/libprism?es2.dylib
rm -f $1/lib/plugin.jar

rm -rf $1/man/
