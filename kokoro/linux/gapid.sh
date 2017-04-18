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

# GAPID launch script. https://github.com/google/gapid

DEFAULT_JAVA=/usr/lib/jvm/java-8-openjdk-amd64/jre/bin/java

function checkJavaVersion {
  IFS=. read major minor extra <<< $($1 -version 2>&1 | awk -F '"' '/version/ {print $2}')
  [[ $major -ge 1 && $minor -ge 8 ]] && return 0 || return 1
}

function absname {
  echo "$(cd $1 && pwd)"
}

if type -p java > /dev/null && checkJavaVersion java; then
  JAVA=java
elif [[ -n "$JAVA_HOME" && -x "$JAVA_HOME/bin/java" ]] && checkJavaVersion "$JAVA_HOME/bin/java"; then
  JAVA=$JAVA_HOME/bin/java
elif [ -x $DEFAULT_JAVA ] && checkJavaVersoin $DEFAULT_JAVA; then
  JAVA=DEFAULT_JAVA
else
  >&2 echo "Could not find Java. Ensure it is on the PATH or set JAVA_HOME."
  exit 1
fi

GAPID=$(absname "$(dirname "${BASH_SOURCE[0]}")")
GAPID=$GAPID SWT_GTK3=0 LIBOVERLAY_SCROLLBAR=0 $JAVA -jar $GAPID/lib/gapic.jar $@
