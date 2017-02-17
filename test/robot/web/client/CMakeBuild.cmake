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

set(GOPHERJS_FILES "")

go_glob(${CMAKE_CURRENT_SOURCE_DIR})
set(GOPHERJS_FILES ${GOPHERJS_FILES} ${gofiles})
go_glob(${CMAKE_CURRENT_SOURCE_DIR}/dom)
set(GOPHERJS_FILES ${GOPHERJS_FILES} ${gofiles})
go_glob(${CMAKE_CURRENT_SOURCE_DIR}/widgets/button)
set(GOPHERJS_FILES ${GOPHERJS_FILES} ${gofiles})
go_glob(${CMAKE_CURRENT_SOURCE_DIR}/widgets/grid)
set(GOPHERJS_FILES ${GOPHERJS_FILES} ${gofiles})

gopherjs(MINIFY OUTPUT ../www/grid/grid.js ${GOPHERJS_FILES})
