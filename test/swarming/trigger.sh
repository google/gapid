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

# This script triggers a Swarming task for a given test.

set -x
set -e

if [ -z "${LUCI_CLIENT_ROOT}" ] ; then
  echo "Error: empty environment variable: LUCI_CLIENT_ROOT"
  exit 1
fi

if [ -z "${SWARMING_BUILD_INFO}" ] ; then
  echo "Error: empty environment variable: SWARMING_BUILD_INFO"
  exit 1
fi

if [ -z "${SWARMING_TRIGGERED_DIR}" ] ; then
  echo "Error: empty environment variable: SWARMING_TRIGGERED_DIR"
  exit 1
fi

SWARMING_TEST_DIR=$1
if [ -z "${SWARMING_TEST_DIR}" -o ! -d "${SWARMING_TEST_DIR}" ] ; then
  echo "Error: missing or invalid test directory argument."
  echo "Usage: `basename $0` tests/foobar"
  exit 1
fi
# Sanitize name
SWARMING_TEST_DIR="`dirname ${SWARMING_TEST_DIR}`/`basename ${SWARMING_TEST_DIR}`"
SWARMING_TEST_NAME=`basename ${SWARMING_TEST_DIR}`

# Set up environment
SWARMING_TASK_NAME="${SWARMING_BUILD_INFO}_${SWARMING_TEST_NAME}"
SWARMING_ISOLATE_SERVER=https://chrome-isolated.appspot.com
SWARMING_SERVER=https://chrome-swarming.appspot.com
SWARMING_POOL=SkiaInternal
SWARMING_DEVICES=flame # pixel4
# Priority: lower value is higher priority, defaults to 200: PR short test tasks should be of higher priority than the default
SWARMING_PRIORITY=100
# Timeout: maximum number of seconds for the task to terminate
SWARMING_TIMEOUT=300
# Expiration: number of seconds to wait for a bot to be available
SWARMING_EXPIRATION=600

# The test may override some of the environment variables
source ${SWARMING_TEST_DIR}/env.sh

# Generate config for isolate
cat << EOF > ${SWARMING_TEST_NAME}.isolate
{
  'variables': {
    'files': [
      'agi/',
      'bot-harness.sh',
      'bot-task.sh',
      '${SWARMING_TEST_DIR}/',
    ],
    'command': [
      './bot-harness.sh',
      '${SWARMING_TEST_DIR}',
      '${SWARMING_TIMEOUT}',
      '\${ISOLATED_OUTDIR}',
    ],
  },
}
EOF

# Upload to isolate
${LUCI_CLIENT_ROOT}/isolate.py archive ${SWARMING_AUTH_FLAG} --isolate-server ${SWARMING_ISOLATE_SERVER} --isolate ${SWARMING_TEST_NAME}.isolate --isolated ${SWARMING_TEST_NAME}.isolated
SWARMING_ISOLATED_SHA=`sha1sum ${SWARMING_TEST_NAME}.isolated | awk '{ print $1 }'`

# Trigger Swarming task
for DEV in ${SWARMING_DEVICES} ; do
  SWARMING_TASK_JSON=${SWARMING_TRIGGERED_DIR}/${SWARMING_TEST_NAME}.${DEV}.json
  ${LUCI_CLIENT_ROOT}/swarming.py trigger ${SWARMING_AUTH_FLAG} --swarming ${SWARMING_SERVER} --isolate-server ${SWARMING_ISOLATE_SERVER} --isolated ${SWARMING_ISOLATED_SHA} --task-name ${SWARMING_TASK_NAME} --dump-json ${SWARMING_TASK_JSON} --dimension pool ${SWARMING_POOL} --dimension device_type ${SWARMING_DEVICES} --priority=${SWARMING_PRIORITY} --expiration=${SWARMING_EXPIRATION} --hard-timeout=${SWARMING_TIMEOUT}
done
