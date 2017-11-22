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

# TODO: actually find msvc rather than assuming a specfic version and location, probably
# by starting from the CXX compiler location
set(paths)
find_program(MSVC_CL cl.exe PATHS paths)
if(MSVC_CL MATCHES NOTFOUND)
    message(FATAL_ERROR "cl not found cannot continue building with MSVC")
endif()
get_filename_component(MSVC_CL_DIR ${MSVC_CL} DIRECTORY)
get_filename_component(MSVC_BIN_DIR ${MSVC_CL_DIR} DIRECTORY)
get_filename_component(MSVC_PATH ${MSVC_BIN_DIR} DIRECTORY)


include(FindPackageHandleStandardArgs)
find_package_handle_standard_args(MSVC REQUIRED_VARS MSVC_PATH)
