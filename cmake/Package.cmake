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

set(PROPERTIES_FILENAME "build.properties")
set(PROPERTIES_TEMPLATE_FILE "${CMAKE_CURRENT_LIST_DIR}/${PROPERTIES_FILENAME}.in")
set(PROPERTIES_FILE "${CMAKE_BINARY_DIR}/${PROPERTIES_FILENAME}")

set(PACKAGE_VERSION_MAJOR "${GAPID_VERSION_MAJOR}")
set(PACKAGE_VERSION_MINOR "${GAPID_VERSION_MINOR}")
set(PACKAGE_VERSION_MICRO "${GAPID_VERSION_POINT}")
set(PACKAGE_BUILD_NUMBER "${GAPID_BUILD_NUMBER}")
set(PACKAGE_BUILD_SHA "${GAPID_BUILD_SHA}")

site_name(PACKAGE_HOST_NAME)
set(PACKAGE_BUILD_HOST "${CMAKE_HOST_SYSTEM} ${PACKAGE_HOST_NAME}")
string(TIMESTAMP PACKAGE_BUILD_DATE "%Y-%m-%dT%H:%M:%SZ" UTC)

configure_file("${PROPERTIES_TEMPLATE_FILE}" "${PROPERTIES_FILE}" @ONLY)
install(FILES ${PROPERTIES_FILE} DESTINATION ${TARGET_INSTALL_PATH})
