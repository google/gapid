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

find_library(X11_LIBRARY NAMES X11)
include(FindPackageHandleStandardArgs)
find_package_handle_standard_args(X11 REQUIRED_VARS X11_LIBRARY)

if(X11_FOUND AND NOT TARGET X11::Lib)
	add_library(X11::Lib UNKNOWN IMPORTED)
	set_target_properties(X11::Lib PROPERTIES
		IMPORTED_LOCATION "${X11_LIBRARY}"
	)
endif()
