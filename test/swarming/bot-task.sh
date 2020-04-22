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

# This script is the actual AGI test to run as a Swarming test. Files written
# under SWARMING_OUT_DIR will be available as Swarming output artifacts.

set -x
set -e

if [ -z "${SWARMING_TEST_DIR}" -o ! -d "${SWARMING_TEST_DIR}" ] ; then
  echo "Error: missing or invalid value for environment variable SWARMING_TEST_DIR"
  exit 1
fi

if [ -z "${SWARMING_OUT_DIR}" -o ! -d "${SWARMING_OUT_DIR}" ] ; then
  echo "Error: missing or invalid value for environment variable SWARMING_OUT_DIR"
  exit 1
fi

if [ -z "${SWARMING_AGI}" -o ! -d "${SWARMING_AGI}" ] ; then
  echo "Error: missing or invalid value for environment variable SWARMING_AGI"
  exit 1
fi

cd ${SWARMING_TEST_DIR}

source env.sh

# APK install
if [ -z "${SWARMING_FORCE_INSTALL}" ] ; then
  # Force-install not imposed, let's see if it is already installed
  num_matching_packages=`adb shell pm list packages | grep ${SWARMING_PACKAGE} | wc -l`
  if [ "${num_matching_packages}" != "1" ] ; then
    SWARMING_FORCE_INSTALL=1
  else
    installed_package=`adb shell pm list packages | grep ${SWARMING_PACKAGE} | sed -e 's/^package://'`
    if [ "${installed_package}" != "${SWARMING_PACKAGE}" ] ; then
      SWARMING_FORCE_INSTALL=1
    fi
  fi
fi

if [ ! -z "${SWARMING_FORCE_INSTALL}" ] ; then
  adb install -g -t "$SWARMING_APK"
fi

# Run AGI test
$SWARMING_AGI/gapit benchmark -startframe "${SWARMING_STARTFRAME}" -numframes "${SWARMING_NUMFRAME}" "${SWARMING_PACKAGE}/${SWARMING_ACTIVITY}"

# Save gfxtrace
mv benchmark.gfxtrace ${SWARMING_OUT_DIR}/
