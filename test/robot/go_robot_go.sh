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
# limitations under the License

# Script that controls robot master instance, runs a background process to
# swipe on attached devices to keep them unlocked, and manages uploading
# and clearing artifacts and subjects in robot's storage directory.
# can use '--clear' to remove all artifacts and subjects from robot on boot.
shopt -s extglob
OUTDIR="bazel-out/robot"
BUILD_BOT_DIR="${OUTDIR}/build_bot"
SWIPE_COMMAND="shell input swipe 300 300 0 0 500"

function handle_opts() {
  while true; do
    case $1 in
      -c|--clean) CLEAN=1; shift ;;
      --) shift; break ;;
      *) echo "internal error!" ; exit 1 ;;
    esac
  done
}

function howmany() { echo $#; }

function find_pid() { pgrep -f "$1" -d " "; }

function is_device_awake() {
  local STATE=`adb -s $1 shell dumpsys nfc | awk -F "=" '/mScreenState/ {print $2}'`
  [[ "$STATE" = "ON_UNLOCKED" ]]
  return
}

function kill_swipe() {
  local SWIPE_PID=$(find_pid "$SWIPE_COMMAND")
  for spid in ${SWIPE_PID}; do
    kill -9 $spid
  done
}
function start_swipe() {
  kill_swipe

  echo ">>> Starting swipe process..."
  declare -i NUM_DEVICES=0
  for device in $(adb devices | awk '!/List/ {print $1}'); do
    if is_device_awake $device; then
      echo ">>> starting swipe process on $device"
      nohup watch -n60 "adb -s $device $SWIPE_COMMAND" > /dev/null 2>&1 &
      let NUM_DEVICES++
    fi
  done
  # need to sleep, else we get duplicate processes in our pid list from watch
  sleep 2s
  local SWIPE_PID=$(find_pid "$SWIPE_COMMAND")
  echo ">>> Swiping processes started! PIDs: ${SWIPE_PID}"
  if [[ $(howmany $SWIPE_PID) -ne ${NUM_DEVICES} ]]; then
    echo ">>> Warning, swiping processed started does not match devices connected..."
    echo "    processes found: ($(howmany $SWIPE_PID))"
    echo "    number of devices: (${NUM_DEVICES})"
  fi
}


function build_grid() {
  gopherjs build --minify --output ./test/robot/web/www/grid/grid.js ./test/robot/web/client
  bazel build //test/robot/web:embed
}

function start_master() {
  MASTER_COMMAND="bazel-bin/pkg/robot start master"
  OLD_MASTER_PID=$(find_pid "$MASTER_COMMAND")
  if [[ $OLD_MASTER_PID ]]; then
    echo ">>> Found old master process: ${OLD_MASTER_PID}; Killing..."
    kill -9 $OLD_MASTER_PID
  fi

  echo ">>> Building Robot..."
  build_grid
  bazel build //cmd/robot:robot
  cp -f bazel-bin/cmd/robot/linux_amd64_stripped/robot bazel-bin/pkg/robot
  echo ">>> Waiting for master to start..."
  nohup bazel-bin/pkg/robot start master -baseaddr "${OUTDIR}" > robot.log 2> roboterr.log &
  while ! grep -q "Starting grpc server" < robot.log; do
    sleep 1s
  done
  MASTER_PID=$(find_pid "$MASTER_COMMAND")
  echo ">>> Master process started! PID: ${MASTER_PID}"
}

function clean_output() {
  echo ">>> Clearing output directory..."
  rm -rf "${OUTDIR}";
}

function reset_build_bot() {
  if [[ -e "${BUILD_BOT_DIR}" ]]; then
    echo ">>> Clearing old build bot artifacts..."
    rm -rf "${BUILD_BOT_DIR}"/*
  else
    echo ">>> Making build bot directory..."
    mkdir -p "${BUILD_BOT_DIR}"
  fi
  pushd "${BUILD_BOT_DIR}" > /dev/null
  mkdir gapid/
  mkdir gapid/android-armv8a/
  mkdir gapid/android-armv7a/
  mkdir gapid/android-x86/
  popd > /dev/null
}

function pack_local() {
  reset_build_bot

  echo ">>> Building local artifacts..."
  bazel build pkg
  echo ">>> Packaging local artifacts..."
  cp bazel-bin/pkg/lib/*VirtualSwapchain* ${BUILD_BOT_DIR}/gapid/
  cp bazel-bin/pkg/gapir ${BUILD_BOT_DIR}/gapid/
  cp bazel-bin/pkg/gapis ${BUILD_BOT_DIR}/gapid/
  cp bazel-bin/pkg/gapit ${BUILD_BOT_DIR}/gapid/
  cp bazel-bin/pkg/gapid-arm64-v8a.apk ${BUILD_BOT_DIR}/gapid/android-armv8a/
  cp bazel-bin/pkg/gapid-armeabi-v7a.apk ${BUILD_BOT_DIR}/gapid/android-armv7a/
  cp bazel-bin/pkg/gapid-x86.apk ${BUILD_BOT_DIR}/gapid/android-x86/
  if [[ -z "$(git status -z | tr -d \0)" ]]; then
    BUILD_BOT_CL="$(git log --pretty="format:%H" -1 .)"
    BUILD_BOT_UPLOADER="$(git log --pretty="format:%an" -1 .)"
    BUILD_BOT_TRACK="$(git rev-parse --abbrev-ref HEAD)"
    BUILD_BOT_DESCRIPTION="$(git log --pretty="%s" -1 .)"
  else
    BUILD_BOT_CL="$(git log --pretty="format:%H" -1 .)⚫"
    BUILD_BOT_UPLOADER="$USER"
    BUILD_BOT_TRACK="$(git rev-parse --abbrev-ref HEAD)⚫"
  fi

  upload_build_bot
}

function upload_build_bot() {
  pushd "${BUILD_BOT_DIR}" > /dev/null
  zip gapid-build-bot.zip gapid/gapi[rst] gapid/*VirtualSwapchain* gapid/android-*/*.apk
  popd > /dev/null

  bazel-bin/pkg/robot upload build $BUILD_BOT_TAG -cl="$BUILD_BOT_CL" -description="$BUILD_BOT_DESCRIPTION" -track="$BUILD_BOT_TRACK" -uploader="$BUILD_BOT_UPLOADER" "${BUILD_BOT_DIR}/gapid-build-bot.zip" > /dev/null 2> uploaderr.log

  BUILD_BOT_CL=
  BUILD_BOT_UPLOADER=
  BUILD_BOT_TRACK=
  BUILD_BOT_DESCRIPTION=
  BUILD_BOT_TAG=

  if grep -q "exit status 1" uploaderr.log; then
    echo ">>> Upload failed:"
    cat uploaderr.log
    return 1
  fi
  echo ">>> Uploaded build bot artifacts"
  return 0
}

function test_subj() {
  local SUBJ=$1
  local SUBJFILE=${SUBJ##*/}
  local OBBFILE=( ${SUBJ%/*}/main.*.${SUBJFILE%%.apk}.obb )

  echo "Choose device to test on..."
  adb devices
  local DEV
  read DEV

  adb -s $DEV install $SUBJ
  if [[ -e $OBBFILE ]]; then
    adb -s $DEV push $OBBFILE /sdcard/Android/obb/${SUBJFILE%%.apk}/${OBBFILE##*/}
  fi

  echo "Subject installed, press any key to continue..."
  read -n 1 -s -r


  echo "Running gapid..."
  bazel-bin/pkg/gapid
  echo "gapid done."

  adb -s $DEV uninstall ${SUBJFILE%%.apk}
}

function upload_subj() {
  local SUBJ=$1
  local SUBJFILE=${SUBJ##*/}
  local OBBFILE=( ${SUBJ%/*}/main.*.${SUBJFILE%%.apk}.obb )
  local API
  local OBB
  API=gles
  if [[ $* = *"vk"* || $* = *"vulkan"* ]]; then
    API=vulkan
    echo "vulkan apk detected..."
  fi
  if [[ -e "$OBBFILE" ]]; then
    OBB=-obb="$OBBFILE"
    echo "obb file detected \"$OBBFILE\"..."
  fi

  echo ">>> Uploading subject $SUBJ"
  local FRAMES
  read -p ">>> Enter frames to observe [default: 30]: " FRAMES
  FRAMES=${FRAMES:=30}
  bazel-bin/pkg/robot upload subject -observeframes=$FRAMES -api="$API" $OBB $SUBJ > /dev/null 2>> uploaderr.log
  echo ">>> Uploaded"
}

function upload_dir() {
  for subj in "$1"/*.apk; do
    upload_subj "$subj"
  done
}

function runtime_usage () {
  echo ">>> Now running robot, usage..."
  echo ">quit - kill all processes and exit"
  echo ">swipe - restart swipe processes"
  echo ">restart - restart master process"
  echo ">clean - remove stash and shelf, restart master process"
  echo ">test <apk> - install apk on device, open client for testing, uninstall apk"
  echo ">upload <apk> - upload apk"
  echo ">upload <dir> - upload all of the apks in <dir>"
  echo ">upload artifacts - package and reupload local artifacts"
}

# Main process
pushd $(dirname $0/../..) > /dev/null
handle_opts $(getopt -l "clean" -o "c" -n "go_robot_go.sh" -- "$@")

start_swipe
if [[ $CLEAN ]]; then
  clean_output
fi
start_master

QUIT=0
while (( $QUIT == 0 )); do
  runtime_usage
  read -e -a COMMANDS
  COMMAND=${COMMANDS[0]}
  if [[ $COMMAND = "quit" ]]; then
    echo ">>> Quitting..."
    bazel-bin/pkg/robot stop
    echo ">>> Stopped master"
    kill_swipe
    echo ">>> Killed swipe"
    QUIT=1
  elif [[ $COMMAND = "swipe" ]]; then
    start_swipe
  elif [[ $COMMAND = "restart" ]]; then
    bazil-bin/pkg/robot stop
    start_master
  elif [[ $COMMAND = "clean" ]]; then
    bazil-bin/pkg/robot stop
    clean_output
    start_master
  elif [[ $COMMAND = "test" ]]; then
    UPLOAD_PATH=$(realpath ${COMMANDS[1]/#\~/$HOME})
      case $UPLOAD_PATH in
        *.apk)
          if [[ -e $UPLOAD_PATH ]]; then
            test_subj $UPLOAD_PATH
          fi ;;
        *)
          echo "not implemented yet" ;;
      esac
  elif [[ $COMMAND = "upload" ]]; then
    case ${COMMANDS[1]} in
      artifacts)
        pack_local ;;
      *)
        UPLOAD_PATH=$(realpath ${COMMANDS[1]/#\~/$HOME})
        case $UPLOAD_PATH in
          *.apk)
            if [[ -e $UPLOAD_PATH ]]; then
              upload_subj $UPLOAD_PATH
            fi ;;
          *)
            if [[ -d $UPLOAD_PATH ]]; then
              upload_dir $UPLOAD_PATH
            fi ;;
        esac ;;
    esac
  fi
done
popd > /dev/null

