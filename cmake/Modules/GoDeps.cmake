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

include(${GO_ENV})

execute_process(

COMMAND "${CMAKE_Go_COMPILER}" list -tags "analytics crashreporting integration" -pkgdir ${GO_PKG} -f "
set(Imports {{join .Imports \"\\n    \"}})
set(TestImports {{join .TestImports \"\\n    \"}})
set(XTestImports {{join .XTestImports \"\\n    \"}})
" ${GO_PACKAGE}
    OUTPUT_VARIABLE CMAKE_CONFIGURABLE_FILE_CONTENT
)
configure_file("${CMAKE_ROOT}/Modules/CMakeConfigurableFile.in" "${GO_DEPS_FILE}" @ONLY)
