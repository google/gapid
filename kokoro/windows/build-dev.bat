goto :start
Copyright (C) 2019 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Windows Build Script.

:start

REM Get date in YYYYMMDD format
for /f "tokens=1,2,3,4 delims=/ " %%a in ('date /t') do set devdate=%%d%%b%%c

set DEV_PREFIX=dev-%devdate%-

call %cd%\github\agi\kokoro\windows\build.bat
