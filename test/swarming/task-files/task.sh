#!/bin/bash
# Copyright (C) 2020 Google Inc.
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

# Swarming task script: this is what runs on the swarming bot

# The only argument is the path to the isolate output directory
OUTDIR=$1
# The script is launched in a directory above the task-files, start by cd'ing in
cd task-files

# Clean-up on exit
cleanup() {
  STATUS=$?
  if [ -z "$EXIT_STATUS" ] ; then
    EXIT_STATUS=${STATUS}
  fi
  # Key "power" (26) toggle between screen off and on, so first make sure to
  # have the screen on with key "wake up" (224), then press "power" (26)
  adb shell input keyevent 224
  sleep 1 # wake-up animation can take some time
  adb shell input keyevent 26
  exit ${EXIT_STATUS}
}

trap cleanup EXIT

set -x

##############################################################################
# Swarming test

GAPIT=./agi/gapit

# Prepare device
# launch adb deamon, this is a good thing to do when starting a swarming task
adb devices
# remove any implicit vulkan layers
adb shell settings delete global enable_gpu_debug_layers
adb shell settings delete global gpu_debug_app
adb shell settings delete global gpu_debug_layers
adb shell settings delete global gpu_debug_layer_app

# NOTE: This script uses the APKs under the apk/ directory, and it assumes that
# all APKs are named as <app-package-name>.apk, e.g. com.example.mygame.apk

for apk in apk/*.apk
do

  # Install APK
  package=`basename ${apk} .apk`
  adb install -g -t ${apk}

  # Capture
  adb logcat -c
  $GAPIT -log-level Verbose trace -disable-pcs -disable-unknown-extensions -record-errors -no-buffer -api vulkan --start-at-frame 20 -capture-frames 5 -observe-frames 1 -out ${OUTDIR}/$package.gfxtrace $package > ${OUTDIR}/trace.out 2> ${OUTDIR}/trace.err
  adb logcat -d > ${OUTDIR}/$package.trace.logcat

  # Stop the app
  adb shell force-stop $package

  # Replay
  adb logcat -c
  $GAPIT video -gapir-nofallback -type sxs -frames-minimum 5 -out ${OUTDIR}/$package.mp4 ${OUTDIR}/$package.gfxtrace > ${OUTDIR}/video.out 2> ${OUTDIR}/video.err
  adb logcat -d > ${OUTDIR}/$package.video.logcat

  if grep 'FramebufferObservation did not match replayed framebuffer' ${OUTDIR}/video.err > /dev/null ; then
    echo "ERROR: replay leads to different image"
    EXIT_STATUS=1
    cleanup
    # cleanup should never return, but be safe and exit anyway
    exit 1
  fi

done

EXIT_STATUS=0
cleanup
exit 0
