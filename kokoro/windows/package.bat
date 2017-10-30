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

if "%WIX%." == "." (
  echo Expected the WIX env variable to point to the WIX toolset.
  exit /b
)

set BUILD_OUT=%1

if exist "%BUILD_OUT%\dist" (
  rmdir /Q /S "%BUILD_OUT%\dist"
)

mkdir "%BUILD_OUT%\dist\gapid"
pushd "%BUILD_OUT%\dist"

awk -F= 'BEGIN {major=0; minor=0; micro=0}^
         /Major/ {major=$2}^
         /Minor/ {minor=$2}^
         /Micro/ {micro=$2}^
         END {print major"."minor"."micro}' ..\pkg\build.properties > version.txt
set /p VERSION=<version.txt

REM Combine package contents.
xcopy /e ..\pkg\* gapid\
copy ..\current\java\gapic-windows.jar gapid\lib\gapic.jar
copy "%~dp0\gapid.ico" gapid
call "%~dp0\copy_jre.bat" "%cd%\gapid\jre"

REM Package up the zip file.
zip -r gapid-%VERSION%-windows.zip gapid

REM Create an MSI installer.
copy "%~dp0\gapid.wxs" .
copy "%~dp0\*.bmp" .
"%WIX%\heat.exe" dir gapid -ag -cg gapid -dr GAPID -template fragment -sreg -sfrag -srd -suid -o component.wxs
"%WIX%\candle.exe" -dGAPIDVersion="%VERSION%" gapid.wxs component.wxs
"%WIX%\light.exe" gapid.wixobj component.wixobj -b gapid -ext WixUIExtension -cultures:en-us -o gapid-%VERSION%-windows.msi

popd
