#!/bin/bash
# Copyright (C) 2019 Google Inc.
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

# Temporary: try out isolate access

set -x

echo "Starting testing isolate access"

date=`date`

echo $date

# SRC=$PWD/github/gapid

# SERVER='https://chrome-isolated.appspot.com'

# LUCI_CLIENT_ROOT=$SRC/tools/build/third_party/luci-py/client

# isolatesha=f9296e7cc5e29130250b1933e08b32af0a73990d

# tmpdir=$SRC/my-tmp
# rm -rf $tmpdir

# $LUCI_CLIENT_ROOT/auth.py -v -v login --service=$SERVER

# echo "HERE Actual command"

# $LUCI_CLIENT_ROOT/isolateserver.py download -I $SERVER --namespace default-gzip -s f9296e7cc5e29130250b1933e08b32af0a73990d --target $tmpdir

# ls $tmpdir

# /usr/bin/python3 $tmpdir/hello.py

echo "THIS IS THE END"

