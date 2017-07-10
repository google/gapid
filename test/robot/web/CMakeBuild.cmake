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

set(web_content "${CMAKE_CURRENT_SOURCE_DIR}/CMakeContent.cmake")

if(NOT EXISTS ${web_content})
    file(RELATIVE_PATH prefix ${CMAKE_SOURCE_DIR} ${CMAKE_CURRENT_SOURCE_DIR})
    file(GLOB_RECURSE all_content RELATIVE ${CMAKE_CURRENT_SOURCE_DIR} 
        "${prefix}/www/*.html"
        "${prefix}/www/*.css"
        "${prefix}/www/*.js"
        "${prefix}/www/*.png"
        "${prefix}/template/*.tmpl"
    )
    list(SORT all_content)
    file(WRITE ${web_content} "
#The set of auto generated embed rules
embed(
  WEB
")
    foreach(entry ${all_content})
        file(APPEND ${web_content} "  \"${entry}\"
")
    endforeach()
    file(APPEND ${web_content} ")
")
endif()
include(${web_content})

build_subdirectory(client)
