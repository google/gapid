#!/bin/bash

# Copyright (C) 2022 Google Inc.
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

# Script to download and build ANGLE and all of its dependencies for AGI
# Use this ANGLE branch for AGI (Note: keep it main until a release branch is created)

ANGLE_BRANCH=main

PYTHON_VERSION="3.9.5"

APK_DIR="$KOKORO_ARTIFACTS_DIR/ANGLE/angle/out/APK"

gn_args=(
"android32_ndk_api_level = 26"
"android64_ndk_api_level = 26"
"angle_enable_annotator_run_time_checks = true"
"angle_enable_trace = true"
"angle_enable_vulkan = true"
"angle_enable_vulkan_validation_layers = false"
"angle_expose_non_conformant_extensions_and_versions = true"
"angle_force_thread_safety = true"
"angle_libs_suffix = \"_angle\""
"dcheck_always_on = false"
"enable_remoting = true"
"ffmpeg_branding = \"Chrome\""
"is_component_build = false"
"is_debug = false"
"is_official_build = true"
"proprietary_codecs = true"
"symbol_level = 2"
"target_cpu = \"arm64\""
"target_os = \"android\""
)


function setup_python_virtual_env {
  pyenv uninstall -f "$1"
  sudo apt-get install liblzma-dev
  pyenv install "$1"
  echo "Python versions available."
  pyenv versions
  pyenv global "$1"

  python -m venv ./kokoro_env
  source ./kokoro_env/bin/activate
  pip install --upgrade pip setuptools wheel

  echo "Python version details in virtual env."
  python --version
  pip --version
}

setup_python_virtual_env $PYTHON_VERSION


### Check if depot_tools exists othrewise Download Chromium's depot_tools
cd $KOKORO_ARTIFACTS_DIR
ls depot_tools
if [[ $? -ne 0 ]]
then
  git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
fi

if [[ $? -ne 0 ]]
then
  echo "FAILURE to download Chromium depot_tools"
  exit 1
else
  echo "Successfully cloned depot_tools"
fi

export PATH=$PWD/depot_tools:$PATH


mkdir -p ANGLE
cd ANGLE

### Clone the ANGLE source
git clone https://chromium.googlesource.com/angle/angle
cd angle

if [[ $? -ne 0 ]]
then
  echo "FAILURE to download ANGLE source"
  exit 1
else
  echo "Successfully cloned Angle"
fi

### Checkout the correct branch, and sync ANGLE's "external" dependencies
git checkout $ANGLE_BRANCH

python scripts/bootstrap.py

gclient sync

if [[ $? -ne 0 ]]
then
    echo "FAILURE in gclient sync"
    exit 1
fi

yes | ./build/install-build-deps.sh

if [[ $? -ne 0 ]]
then
    echo "FAILURE to install build deps"
    exit 1
fi

gn gen "$APK_DIR" --args="${gn_args[*]}"

if [[ $? -ne 0 ]]
then
    echo "FAILURE to generate files for arm64"
    exit 1
fi

echo "Successfully generated files"

### Do the build
yes | autoninja -C "$APK_DIR" angle_apks ;

if [[ $? -ne 0 ]]
then
    echo "FAILURE to build ANGLE APKs"
    exit 1
fi

echo "Successfully built ANGLE APKs"
