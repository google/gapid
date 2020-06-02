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

DEST=$1

cp -r ${JAVA_HOME}/* ${DEST}/

#### Remove unnecessary files.
# The list of files to remove is obtained by comparing the list of files from
# the Zulu JDK and JRE package from
# https://www.azul.com/downloads/zulu-community. The JRE can be approximated as a subset
# of the JDK. We get the list of files to remove with:
#
# diff -r zulu11...jdk11... zulu11...jre11... | grep '^Only in' | grep jdk11 | grep -v legal
#
# Also, the macOS Zulu JDK package contains symlinks, hence the bespoke remove().
function remove() {
  rm -rf ${DEST}/zulu-11.jdk/Contents/Home/$1
  if [ -L ${DEST}/$1 ]; then
    unlink ${DEST}/$1
  fi
}

remove bin/jar
remove bin/jarsigner
remove bin/javac
remove bin/javadoc
remove bin/javap
remove bin/jcmd
remove bin/jconsole
remove bin/jdb
remove bin/jdeprscan
remove bin/jdeps
remove bin/jhsdb
remove bin/jimage
remove bin/jinfo
remove bin/jlink
remove bin/jmap
remove bin/jmod
remove bin/jps
remove bin/jshell
remove bin/jstack
remove bin/jstat
remove bin/jstatd
remove bin/rmic
remove bin/serialver
remove demo
remove include
remove jmods
remove lib/ct.sym
remove lib/libattach.dylib
remove lib/libsaproc.dylib
remove lib/src.zip
remove man/ja/man1/jar.1
remove man/ja/man1/jarsigner.1
remove man/ja/man1/javac.1
remove man/ja/man1/javadoc.1
remove man/ja/man1/javap.1
remove man/ja/man1/jcmd.1
remove man/ja/man1/jconsole.1
remove man/ja/man1/jdb.1
remove man/ja/man1/jdeps.1
remove man/ja/man1/jinfo.1
remove man/ja/man1/jmap.1
remove man/ja/man1/jps.1
remove man/ja/man1/jrunscript.1
remove man/ja/man1/jstack.1
remove man/ja/man1/jstat.1
remove man/ja/man1/jstatd.1
remove man/ja/man1/rmic.1
remove man/ja/man1/serialver.1
remove man/ja_JP.UTF-8/man1/jar.1
remove man/ja_JP.UTF-8/man1/jarsigner.1
remove man/ja_JP.UTF-8/man1/javac.1
remove man/ja_JP.UTF-8/man1/javadoc.1
remove man/ja_JP.UTF-8/man1/javap.1
remove man/ja_JP.UTF-8/man1/jcmd.1
remove man/ja_JP.UTF-8/man1/jconsole.1
remove man/ja_JP.UTF-8/man1/jdb.1
remove man/ja_JP.UTF-8/man1/jdeps.1
remove man/ja_JP.UTF-8/man1/jinfo.1
remove man/ja_JP.UTF-8/man1/jmap.1
remove man/ja_JP.UTF-8/man1/jps.1
remove man/ja_JP.UTF-8/man1/jrunscript.1
remove man/ja_JP.UTF-8/man1/jstack.1
remove man/ja_JP.UTF-8/man1/jstat.1
remove man/ja_JP.UTF-8/man1/jstatd.1
remove man/ja_JP.UTF-8/man1/rmic.1
remove man/ja_JP.UTF-8/man1/serialver.1
remove man/man1/jar.1
remove man/man1/jarsigner.1
remove man/man1/javac.1
remove man/man1/javadoc.1
remove man/man1/javap.1
remove man/man1/jcmd.1
remove man/man1/jconsole.1
remove man/man1/jdb.1
remove man/man1/jdeps.1
remove man/man1/jinfo.1
remove man/man1/jmap.1
remove man/man1/jps.1
remove man/man1/jrunscript.1
remove man/man1/jstack.1
remove man/man1/jstat.1
remove man/man1/jstatd.1
remove man/man1/rmic.1
remove man/man1/serialver.1
