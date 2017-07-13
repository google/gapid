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

set(sources
    "third_party/cityhash/src/city.cc"
)

set(cityhash_gen "${CMAKE_BINARY_DIR}/third_party/cityhash")

configure_file("third_party/cityhash_config.h.in" "${cityhash_gen}/config.h" @ONLY)
configure_file("third_party/cityhash_byteswap.h.in" "${cityhash_gen}/byteswap.h" @ONLY)

if(NOT DISABLED_CXX)
    add_library(cityhash ${sources})

    target_include_directories(cityhash PUBLIC "${cityhash_gen}")
    target_include_directories(cityhash PUBLIC "third_party/cityhash/src")

    if(ANDROID)
        target_link_libraries(cityhash)
    endif()
endif()
