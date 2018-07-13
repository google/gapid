#!/bin/bash
# Copyright (C) 2018 Google Inc.
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

# Presubmit Checks Script.
BAZEL=${BAZEL:-bazel}
BUILDIFIER=${BUILDIFIER:-buildifier}
BUILDOZER=${BUILDOZER:-buildozer}
CLANG_FORMAT=${CLANG_FORMAT:-clang-format}
GOFMT=${GOFMT:-gofmt}

if test -t 1; then
  ncolors=$(tput colors)
  if test -n "$ncolors" && test $ncolors -ge 8; then
    normal="$(tput sgr0)"
    red="$(tput setaf 1)"
    green="$(tput setaf 2)"
  fi
fi

function check() {
  local name=$1; shift
  echo -n "Running check $name... "

  if ! "$@"; then
    echo "${red}FAILED${normal}"
    echo "  Error executing: $@";
    exit 1
  fi

  if ! git diff --quiet HEAD; then
    echo "${red}FAILED${normal}"
    echo "  Git workspace not clean:"
    git --no-pager diff -p HEAD
    echo "${red}Check $name failed.${normal}"
    exit 1
  fi

  echo "${green}OK${normal}"
}

function run_clang_format() {
  find . -name "*.h" -o -name "*.cpp" -o -name "*.mm" -o -name "*.proto" | xargs $CLANG_FORMAT -i -style=Google
}

function run_gofmt() {
  find . -name "*.go" | xargs $GOFMT -w
}

function run_buildifier() {
  find . -name "*.BUILD" -o -name "BUILD.bazel" | xargs $BUILDIFIER
}

function run_buildozer() {
  $BUILDOZER -quiet 'fix movePackageToTop unusedLoads usePlusEqual' //...:__pkg__
  # Handle exit code 3 (success, no changes).
  local r=$?
  [ $r -eq 3 ] && return 0 || return $r
}

function run_gazelle() {
  echo # TODO: figure out a way to make bazel not print anything.
  $BAZEL run gazelle
}

# Ensure we are clean to start out with.
check "git workspace must be clean" true

# Check clang-format.
check clang-format run_clang_format

# Check gofmt.
check gofmt run_gofmt

# Check buildifier.
check buildifier run_buildifier

# Check bazel style.
check "buildozer fix" run_buildozer

# Check gazelle.
check "gazelle" run_gazelle

echo
echo "${green}All check completed successfully.$normal"
