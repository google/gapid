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

if [ -z "${SWARMING_TASK_PREFIX}" ] ; then
  echo "Error: empty environment variable: SWARMING_TASK_PREFIX"
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
SWARMING_TASK_NAME="${SWARMING_TASK_PREFIX}_${SWARMING_TEST_NAME}"
SWARMING_ISOLATE_SERVER=https://chrome-isolated.appspot.com
SWARMING_SERVER=https://chrome-swarming.appspot.com
SWARMING_POOL=SkiaInternal
# Bash-array of device names: (dev1 dev2 dev3)
SWARMING_DEVICES=(flame)
# Priority: lower value is higher priority, defaults to 200: PR short test tasks
# should be of higher priority than the default
SWARMING_PRIORITY=100
# Timeout: maximum number of seconds for the task to terminate
SWARMING_TIMEOUT=300
# Expiration: number of seconds to wait for a bot to be available
SWARMING_EXPIRATION=600

# The test may override some of the environment variables. For security reasons,
# we don't want to 'source' the test 'env.sh' as this could lead to code
# injection. Instead, we grep all relevant environment variables that may be
# overriden.
for envvar in SWARMING_PRIORITY SWARMING_TIMEOUT SWARMING_EXPIRATION ; do
  value=`grep ${envvar} ${SWARMING_TEST_DIR}/env.sh | sed -e 's/^.*=//'`
  if [ ! -z "${value}" ] ; then
    declare ${envvar}=${value}
  fi
done

# Special case for SWARMING_DEVICES, which is a bash array, we must handle the
# parenthesis accordingly
# This sed line transforms "SWARMING_DEVICES={dev1 dev2)" into "dev1 dev2"
devices=`grep  SWARMING_DEVICES ${SWARMING_TEST_DIR}/env.sh |sed -e 's/^.*=[(]//' -e 's/)$//'`
if [ ! -z "${devices}" ] ; then
  declare SWARMING_DEVICES=(${devices})
fi

# generate config for isolate
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
for DEV in ${SWARMING_DEVICES[@]} ; do
  mkdir -p ${SWARMING_TRIGGERED_DIR}/${DEV}
  SWARMING_TASK_JSON=${SWARMING_TRIGGERED_DIR}/${DEV}/${SWARMING_TEST_NAME}.json
  ${LUCI_CLIENT_ROOT}/swarming.py trigger ${SWARMING_AUTH_FLAG} --swarming ${SWARMING_SERVER} --isolate-server ${SWARMING_ISOLATE_SERVER} --isolated ${SWARMING_ISOLATED_SHA} --task-name ${SWARMING_TASK_NAME} --dump-json ${SWARMING_TASK_JSON} --dimension pool ${SWARMING_POOL} --dimension device_type ${DEV} --priority=${SWARMING_PRIORITY} --expiration=${SWARMING_EXPIRATION} --hard-timeout=${SWARMING_TIMEOUT}
done
