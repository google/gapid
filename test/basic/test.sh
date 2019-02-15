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

# Uncomment this can be useful when debugging this script
# set -u
# set -e
# set -x

tmpdir=`mktemp -d -p .`
tracedir=`pwd`/traces

do_gapit() {
  verb=$1
  gapit $* > $verb.out 2> $verb.err
  retval=$?
  if test $retval -eq 0
  then
    echo "[OK ] gapit $*"
  else
    echo "[ERR] gapit $*"
  fi
}

for t in $tracedir/*.gfxtrace
do
  trace=`basename $t .gfxtrace`
  destdir=$tmpdir/$trace
  mkdir $destdir
  cp $t $destdir/

  (
    cd $destdir/
    gfxtrace="$trace.gfxtrace"
    echo "========== `date` START $gfxtrace =========="
    do_gapit commands $gfxtrace
    do_gapit create_graph_visualization -format dot -out $trace.dot $gfxtrace
    do_gapit dump $gfxtrace
    do_gapit dump_fbo $gfxtrace
    do_gapit dump_pipeline $gfxtrace
    do_gapit dump_replay $gfxtrace
    do_gapit dump_resources $gfxtrace
    do_gapit export_replay $gfxtrace
    do_gapit memory $gfxtrace
    do_gapit stats $gfxtrace
    do_gapit trim -frames-start 2 -frames-count 2 $gfxtrace
    do_gapit unpack $gfxtrace
    echo "========== `date` FINISH $gfxtrace =========="
  )

  rm -rf $destdir
done

rm -rf $tmpdir
