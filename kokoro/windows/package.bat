goto :start
Copyright (C) 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Windows Package Script.
 - Expects MSYS on the %PATH% and %JAVA_HOME% set correctly.

:start

if "%1." == "." (
  echo Expected the build folder as an argument.
  exit /b
)

if "%2." == "." (
  echo Expected the source root as an argument.
  exit /b
)

if "%WIX%." == "." (
  echo Expected the WIX env variable to point to the WIX toolset.
  exit /b
)

set BUILD_OUT=%1
set BIN_DIR=%2\bazel-bin

if exist "%BUILD_OUT%\dist" (
  rmdir /Q /S "%BUILD_OUT%\dist"
)

mkdir "%BUILD_OUT%\dist\agi"
pushd "%BUILD_OUT%\dist"

awk -F= 'BEGIN {major=0; minor=0; micro=0}^
         /Major/ {major=$2}^
         /Minor/ {minor=$2}^
         /Micro/ {micro=$2}^
         END {print major"."minor"."micro}' %BIN_DIR%\pkg\build.properties > version.txt
set /p VERSION=<version.txt

REM Combine package contents.
xcopy /e %BIN_DIR%\pkg\* agi\
copy "%BIN_DIR%\tools\logo\agi_ico.ico" agi\agi.ico
copy "c:\tools\msys64\mingw64\bin\libgcc_s_seh-1.dll" agi
copy "c:\tools\msys64\mingw64\bin\libstdc++-6.dll" agi
copy "c:\tools\msys64\mingw64\bin\libwinpthread-1.dll" agi
call "%~dp0\copy_jre.bat" "%cd%\agi\jre"
REM Must include the JDK source code for legal reasons
wget -q https://mirror.bazel.build/openjdk/azul-zulu11.31.11-ca-jdk11.0.3/zsrc11.31.11-jdk11.0.3.zip
move zsrc11.31.11-jdk11.0.3.zip agi\jre\zsrc11.31.11-jdk11.0.3.zip

REM Package up the zip file.
zip -r agi-%VERSION%-windows.zip agi

REM Create an MSI installer.
copy "%~dp0\gapid.wxs" agi.wxs
copy "%~dp0\*.bmp" .
"%WIX%\heat.exe" dir agi -ag -cg agi -dr AGI -template fragment -sreg -sfrag -srd -suid -o component.wxs
"%WIX%\candle.exe" -dAGIVersion="%VERSION%" agi.wxs component.wxs
"%WIX%\light.exe" agi.wixobj component.wixobj -b agi -ext WixUIExtension -cultures:en-us -o agi-%VERSION%-windows.msi

REM Copy the symbol file to the output.
if exist "%BIN_DIR%\cmd\gapir\cc\gapir.sym" (
  copy "%BIN_DIR%\cmd\gapir\cc\gapir.sym" gapir-%VERSION%-windows.sym
)

popd
