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

set(PROPERTIES_FILENAME "source.properties")
set(PROPERTIES_TEMPLATE_FILE "${CMAKE_CURRENT_LIST_DIR}/${PROPERTIES_FILENAME}.in")
set(PROPERTIES_FILE "${CMAKE_BINARY_DIR}/${PROPERTIES_FILENAME}")

file(READ "${CMAKE_SOURCE_DIR}/cmd/gapis/version.go" VERSION_GO)
if(NOT VERSION_GO MATCHES "app.VersionSpec{Major: ([0-9]+), Minor: ([0-9]+)}")
    message(FATAL_ERROR "version.go is not valid")
endif()

set(CPACK_GENERATOR "ZIP")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "GPU Debugging tools")
set(CPACK_PACKAGE_DESCRIPTION "Tools that support GPU debugging and profiling within an IDE.")
set(CPACK_PACKAGE_VENDOR "Android")
set(CPACK_PACKAGE_DESCRIPTION_FILE) # TODO
set(CPACK_RESOURCE_FILE_LICENSE) # TODO
set(CPACK_PACKAGE_VERSION_MAJOR "${CMAKE_MATCH_1}")
set(CPACK_PACKAGE_VERSION_MINOR "${CMAKE_MATCH_2}")
set(CPACK_PACKAGE_VERSION_PATCH "0")
set(CPACK_PACKAGE_INSTALL_DIRECTORY "CMake ${CMake_VERSION_MAJOR}.${CMake_VERSION_MINOR}")
string(TOLOWER "${CMAKE_PROJECT_NAME}-${TARGET_OS}" CPACK_PACKAGE_FILE_NAME)

configure_file("${PROPERTIES_TEMPLATE_FILE}" "${PROPERTIES_FILE}" @ONLY)
install(FILES ${PROPERTIES_FILE} DESTINATION ".")

include(CPack)
