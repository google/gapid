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
# limitations under the License.

# True source of GAPID versions.
# Increment these numbers immediately after releasing a new version.
set(GAPID_VERSION_MAJOR 0)
set(GAPID_VERSION_MINOR 9)
set(GAPID_VERSION_POINT 1)

if (NOT DEFINED GAPID_BUILD_NUMBER)
    set(GAPID_BUILD_NUMBER 0)
endif()

if (NOT DEFINED GAPID_BUILD_SHA)
    set(GAPID_BUILD_SHA "developer")
endif()


set(GAPID_VERSION_AND_BUILD "${GAPID_VERSION_MAJOR}.${GAPID_VERSION_MINOR}.${GAPID_VERSION_POINT}:${GAPID_BUILD_NUMBER}:${GAPID_BUILD_SHA}")

message(STATUS "Building GAPID ${GAPID_VERSION_AND_BUILD}")
