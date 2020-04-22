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

# This script is the swarming task harness. This is the entry point for the
# Swarming bot.

set -x

export SWARMING_TEST_DIR=$1
if [ -z "${SWARMING_TEST_DIR}" -o ! -d "${SWARMING_TEST_DIR}" ] ; then
  echo "Error: missing or invalid test directory argument."
  exit 1
fi

export SWARMING_TIMEOUT=$2
if [ -z "${SWARMING_TIMEOUT}" ] ; then
  echo "Error: missing timeout argument."
  exit 1
fi

export SWARMING_OUT_DIR=$3
if [ -z "${SWARMING_OUT_DIR}" -o ! -d "${SWARMING_OUT_DIR}" ] ; then
  echo "Error: missing or invalid outdir argument."
  exit 1
fi

## Check expected files
if [ ! -f bot-task.sh ] ; then
  echo "Error: no bot-task.sh file found"
  exit 1
fi

if [ ! -d agi ] ; then
  echo "Error: no agi/ directory found"
  exit 1
fi

## Check we can access the Android device
if adb shell true ; then
  echo "Device fingeprint: " `adb shell getprop ro.build.fingerprint`;
else
  echo "Error: zero or more than one device connected"
  exit 1
fi

## Set up environment
export SWARMING_AGI=`pwd`/agi
source ${SWARMING_TEST_DIR}/env.sh

# Number of seconds to dump logcat and turn the screen off.
SWARMING_TIMEOUT_SAFETY=10
if [ ${SWARMING_TIMEOUT} -lt ${SWARMING_TIMEOUT_SAFETY} ] ; then
  echo "Error: SWARMING_TIMEOUT is less than ${SWARMING_TIMEOUT_SAFETY} seconds."
  exit 1
fi
SWARMING_TIMEOUT_GUARD="$(( ${SWARMING_TIMEOUT} - ${SWARMING_TIMEOUT_SAFETY} ))"

## Lauch task test
adb logcat -c
timeout -k 1 $SWARMING_TIMEOUT_GUARD ./bot-task.sh
EXIT_CODE=$?
adb logcat -d > ${SWARMING_OUT_DIR}/logcat.txt

echo "Exit code: ${EXIT_CODE}"

# Tests may fail halfway through, try to salvage any gfxtrace
mkdir -p ${SWARMING_OUT_DIR}/harness-salvage/
for gfxtrace in ${SWARMING_TEST_DIR}/*.gfxtrace ; do
  mv ${gfxtrace} ${SWARMING_OUT_DIR}/harness-salvage/
done

## Turn off the device screen
# Key "power" (26) toggle between screen off and on, so first make sure to have
# the screen on with key "wake up" (224), then press "power" (26)
adb shell input keyevent 224
sleep 2 # wait a bit to let any kind of device wake up animation terminate
adb shell input keyevent 26

# Analyze the exit code
if test ${EXIT_CODE} -eq 124 -o ${EXIT_CODE} -eq 137 ; then
  echo "TIMEOUT"
  echo "Sleep a bit more to hopefully trigger a Swarming-level timeout, which will be reported as such"
  sleep ${SWARMING_TIMEOUT_SAFETY}
elif test ${EXIT_CODE} -ne 0 ; then
  echo "FAIL"
else
  echo "PASS"
fi

exit ${EXIT_CODE}
