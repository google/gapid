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

include(${GRADLE_ENV})

separate_arguments(args UNIX_COMMAND ${GRADLE_ARGS})

set(gradle "./gradlew")
if(CMAKE_HOST_WIN32)
    set(gradle "gradlew.bat")
endif(CMAKE_HOST_WIN32)

execute_process(
    COMMAND "${gradle}"
            ${args}
            -Djava.awt.headless=true # Prevent gradle from stealing window focus.
            build
    RESULT_VARIABLE result
)
if(result)
    message(FATAL_ERROR ${result})
endif()
