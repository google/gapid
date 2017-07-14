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

# TODO: Debug builds when CMAKE_BUILD_TYPE=Debug
# TODO: Test support, with coverage helpers
# TODO: Options for things like -race

set(CMAKE_Go_COMPILE_OBJECT "")
set(CMAKE_Go_LINK_EXECUTABLE "\"${CMAKE_COMMAND}\" -DGO_ENV=${GO_ENV} -DGO_BUILD=<TARGET> <LINK_FLAGS> -P ${GO_COMPILE}")
set(CMAKE_Go_CREATE_STATIC_LIBRARY "\"${CMAKE_COMMAND}\" -E touch <TARGET>")
set(CMAKE_Go_ARCHIVE_CREATE ${CMAKE_Go_CREATE_STATIC_LIBRARY})
