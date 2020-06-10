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

if [ -z $JRE_HOME ]; then
  echo Expected the JRE_HOME env variable.
  exit
fi

cp -r $JRE_HOME/* $1

# Remove unnecessary files.
rm -f $1/bin/jaotc
rm -f $1/bin/jfr
rm -f $1/bin/jjs
rm -f $1/bin/jrunscript
rm -f $1/bin/keytool
rm -f $1/bin/pack200
rm -f $1/bin/rmid
rm -f $1/bin/rmiregistry
rm -f $1/bin/unpack200

rm -rf $1/lib/jfr/

rm -rf $1/man/

# "Work-around" for some strange OSX behavior. Without this folder next to the java binary, when
# launching the app, the menu bar is unresponsive. Switching to another app and back makes it come
# to live, and so does creating a "Contents" folder next to the binaries. Go figure.
mkdir $1/bin/Contents
