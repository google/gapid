#!/bin/bash
shopt -s extglob
clear
pushd $(dirname $0) > /dev/null
OUTDIR="bazel-out/robot/"
BUILD_BOT_DIR="${OUTDIR}build_bot/"

function handle_opts() {
  while true; do
    case $1 in
      -c|--clean) CLEAN=1; shift ;;
      --) shift; break ;;
      *) echo "internal error!" ; exit 1 ;;
    esac
  done
}
handle_opts $(getopt -l "clean" -o "c" -n "go_robot_go.sh" -- "$@")

function howmany() { echo $#; }

function find_pid() { pgrep -f "$1" -d " "; }

function is_device_awake() {
  STATE=`adb -s $1 shell dumpsys nfc \
    | grep mScreenState \
    | awk '{split($1,a,"="); print a[2]}'`

  if [[ "$STATE" = "ON_UNLOCKED" ]]; then
    echo 0
    return
  fi
  echo 1
  return
}

function start_swipe() {
  SWIPE_COMMAND="watch -n60 adb -s"
  OLD_SWIPE_PID=$(find_pid "$SWIPE_COMMAND")
  if [[ $OLD_SWIPE_PID ]]; then
    echo ">>> Found old swipe process: ${OLD_SWIPE_PID}; Killing..."
    kill -9 $OLD_SWIPE_PID
  fi

  echo ">>> Starting swipe process..."
  let NUM_DEVICES=0
  for device in $(adb devices | grep -v List | awk '{print $1}'); do
    if [[ $(is_device_awake $device) -eq 0 ]]; then
      echo ">>> starting swipe process on $device"
      nohup watch -n60 "adb -s $device shell input swipe 300 300 0 0 500" > /dev/null 2>&1 &
      let NUM_DEVICES++
    fi
  done
  # need to sleep, else we get duplicate processes in our pid list from watch
  sleep 2s
  SWIPE_PID=$(find_pid "$SWIPE_COMMAND")
  echo ">>> Swiping processes started! PIDs: ${SWIPE_PID}"
  if [[ $(howmany ${SWIPE_PID}) -ne ${NUM_DEVICES} ]]; then
    echo ">>> Warning, swiping processed started does not match devices connected..."
    echo "    processes found: ($(howmany ${SWIPE_PID}))"
    echo "    number of devices: (${NUM_DEVICES})"
  fi
}

function end_swipe() {
  for spid in ${SWIPE_PID}; do
    kill -9 $spid
  done
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
  echo y | rm bazel-bin/pkg/robot
  cp bazel-bin/cmd/robot/linux_amd64_stripped/robot bazel-bin/pkg/robot
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
  cp bazel-bin/pkg/lib/*VirtualSwapchain* ${BUILD_BOT_DIR}gapid/
  cp bazel-bin/pkg/gapir ${BUILD_BOT_DIR}gapid/
  cp bazel-bin/pkg/gapis ${BUILD_BOT_DIR}gapid/
  cp bazel-bin/pkg/gapit ${BUILD_BOT_DIR}gapid/
  cp bazel-bin/pkg/gapid-aarch64.apk ${BUILD_BOT_DIR}gapid/android-armv8a/
  cp bazel-bin/pkg/gapid-armeabi.apk ${BUILD_BOT_DIR}gapid/android-armv7a/
  cp bazel-bin/pkg/gapid-x86.apk ${BUILD_BOT_DIR}gapid/android-x86/
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

  bazel-bin/pkg/robot upload build $BUILD_BOT_TAG -cl="$BUILD_BOT_CL" -description="$BUILD_BOT_DESCRIPTION" -track="$BUILD_BOT_TRACK" -uploader="$BUILD_BOT_UPLOADER" "${BUILD_BOT_DIR}gapid-build-bot.zip" > /dev/null 2> uploaderr.log

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

function check_subj() {
  # todo: test options -i <interactive> -t <time> -o <manual obb file> etc...
  SUBJ=$1
  SUBJFILE=${SUBJ##*/}
  OBBFILE=( ${SUBJ%/*}/main.*.${SUBJFILE%%.apk}.obb )
  if [[ $* = *"-i"* ]]; then
    read -e -p "Enter subject's API: [default: gles]" API
    API=${API:=gles}
    read -e -p "Enter subject's OBB file: [default: $obb]" OBB
    OBB=${OBB:=-obb="$obb"}
  else
    API=gles
    if [[ $* = *"vk"* || $* = *"vulkan"* ]]; then
      API=vulkan
      echo "vulkan apk detected..."
    fi
    if [[ -e "$OBBFILE" ]]; then
      OBB=-obb="$OBBFILE"
      echo "obb file detected \"$OBBFILE\"..."
    fi
  fi
}

function reset_subj() {
  API=
  OBB=
  OBBFILE=
  SUBJ=
  SUBJFILE=
}


function test_subj() {
  check_subj $*

  echo "Choose device to test on..."
  adb devices
  read DEV

  adb -s $DEV install $SUBJ
  if [[ -n $OBB ]]; then
    adb -s $DEV sh mkdir /sdcard/Android/obb/${SUBJFILE%%.apk}
    adb -s $DEV push $OBBFILE /sdcard/Android/obb/${SUBJFILE%%.apk}/${OBBFILE##*/}
  fi

  echo "Subject installed, press any key to continue..."
  read -n 1 -s -r


  echo "Running gapid..."
  bazel-bin/pkg/gapid
  echo "gapid done."

  adb -s $DEV uninstall ${SUBJFILE%%.apk}
  DEV=

  reset_subj
}

function upload_subj() {
  check_subj $*

  echo ">>> Uploading subject $SUBJFILE"
  read -p ">>> Enter trace time [default: 1m]: " TIME
  TIME=${TIME:=1m}
  bazel-bin/pkg/robot upload subject -tracetime=$TIME -api="$API" $OBB $SUBJ > /dev/null 2>> uploaderr.log
  echo ">>> Uploaded"

  reset_subj
}

function upload_dir() {
  for subj in "$1"/*.apk; do
    upload_subj "$subj"
  done
}

function runtime_usage () {
  sleep 5s
  clear
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
    kill -9 ${SWIPE_PID}
    echo ">>> Killed swipe"
    QUIT=1
  elif [[ $COMMAND = "swipe" ]]; then
    kill -9 ${SWIPE_PID}
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
          fi;;
        *)
          if [[ -d $UPLOAD_PATH ]]; then
            echo "not implemented yet"
          fi;;
      esac
  elif [[ $COMMAND = "upload" ]]; then
    case ${COMMANDS[1]} in
      artifacts)
        pack_local
        ;;
      *)
        UPLOAD_PATH=$(realpath ${COMMANDS[1]/#\~/$HOME})
        case $UPLOAD_PATH in
          *.apk)
            if [[ -e $UPLOAD_PATH ]]; then
              upload_subj $UPLOAD_PATH
            fi;;
          *)
            if [[ -d $UPLOAD_PATH ]]; then
              upload_dir $UPLOAD_PATH
            fi;;
        esac
        ;;
    esac
  fi
done
popd > /dev/null

