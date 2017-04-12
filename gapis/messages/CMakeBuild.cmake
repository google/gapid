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

stringgen(
    INPUT "en-us.stb.md"
    OUTPUT_GO "messages.go"
    OUTPUT_API "messages.api"
    PACKAGE ${GO_BIN}/strings
)

if(TARGET stringgen-messages)
    install(FILES ${GO_BIN}/strings/en-us.stb DESTINATION "${TARGET_INSTALL_PATH}/strings")
endif()
