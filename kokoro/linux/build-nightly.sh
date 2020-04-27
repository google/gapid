#!/bin/bash

# Copyright 2020 Google LLC
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

# Kokoro linux/nightly build script

export SWARMING_TIMESTAMP=`date '+%Y%m%d-%H%M%S'`
export SWARMING_TASK_PREFIX="Nightly_${SWARMING_TIMESTAMP}"
export SWARMING_X20_TEST_DIR="tests_nightly"

# We obtain the cumulated results from previous nightly runs through our x20
# input, and we want to make sure to propagate those results event if the
# current build fails, so we start by moving the result file to the
# artifacts. This file will be updated later with this run's Swarming results.
if [ -f ${KOKORO_GFILE_DIR}/results.json ] ; then
  mkdir -p ${KOKORO_ARTIFACTS_DIR}/swarming
  cp ${KOKORO_GFILE_DIR}/results.json ${KOKORO_ARTIFACTS_DIR}/swarming/results.json
fi

. $PWD/github/agi/kokoro/linux/build.sh
