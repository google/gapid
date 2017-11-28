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

if(MSVC) # WIN32 Symbols require building with MSVC

elseif(APPLE)
  #execute_process(COMMAND "dsymutil" "gapir" "-o" "gapir.dSYM")
  # "dump_syms" -g "gapir.dSYM" "gapir" > gapir.sym && rm -r "gapir.dSYM" && strip "gapir")
else() # LINUX or ANDROID
  # "dump_syms" "gapir" > gapir.sym && strip "gapir")
endif()

#execute_process(COMMAND "${DUMP_SYMS_CMD}" RESULT_VARIABLE result)
# if(result)
#   message(FATAL_ERROR ${result})
# endif()
