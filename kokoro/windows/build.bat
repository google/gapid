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

Windows Build Script.

:start
SET BUILD_ROOT=%cd%
SET SRC=%cd%\github\src\github.com\google\gapid

REM Install the Android SDK components and NDK.
set ANDROID_SDK_HOME=%LOCALAPPDATA%\Android\Sdk
echo y | %ANDROID_SDK_HOME%\tools\android.bat update sdk -u -a --filter build-tools-21.1.2,android-21
wget -q https://dl.google.com/android/repository/android-ndk-r15b-windows-x86_64.zip
unzip -q android-ndk-r15b-windows-x86_64.zip

REM Get GO 1.8.3
set GO_ARCHIVE=go1.8.3.windows-amd64.zip
wget -q https://storage.googleapis.com/golang/%GO_ARCHIVE%
unzip -q %GO_ARCHIVE%
set GOROOT=%BUILD_ROOT%\go
set PATH=%GOROOT%\bin;%PATH%

REM Install WiX Toolset.
wget -q https://github.com/wixtoolset/wix3/releases/download/wix311rtm/wix311-binaries.zip
unzip -q -d wix wix311-binaries.zip
set WIX=%cd%\wix

REM Fix up the MSYS environment: remove gcc and add mingw's gcc
c:\tools\msys64\usr\bin\bash --login -c "pacman -R --noconfirm gcc"
c:\tools\msys64\usr\bin\bash --login -c "pacman -S --noconfirm mingw-w64-x86_64-gcc"

REM Setup the build config file.
(
  echo {
  echo  "Flavor": "release",
  echo  "OutRoot": "%cd%\out",
  echo  "JavaHome": "%JAVA_HOME%",
  echo  "AndroidSDKRoot": "%ANDROID_SDK_HOME%",
  echo  "AndroidNDKRoot": "%cd%\android-ndk-r15b",
  echo  "CMakePath": "c:\Program Files\Cmake\bin\cmake.exe",
  echo  "NinjaPath": "c:\ProgramData\chocolatey\bin\ninja.exe",
  echo  "PythonPath": "c:\Python35\python.exe",
  echo  "MSYS2Path": "c:\tools\msys64"
  echo }
) > gapid-config
type gapid-config
sed -e s/\\/\\\\/g gapid-config > %SRC%\.gapid-config

REM Fetch the submodules.
cd %SRC%
git submodule update --init

REM Invoke the build. At this point, only ensure that the tests build, but don't
REM execute the tests.
echo %DATE% %TIME%
if "%KOKORO_GITHUB_COMMIT%." == "." (
  set BUILD_SHA=%KOKORO_GITHUB_PULL_REQUEST_COMMIT%
) else (
  set BUILD_SHA=%KOKORO_GITHUB_COMMIT%
)
call do.bat build --test build --buildnum %KOKORO_BUILD_NUMBER% --buildsha "%BUILD_SHA%"
if %ERRORLEVEL% GEQ 1 exit /b %ERRORLEVEL%
echo %DATE% %TIME%
cd %BUILD_ROOT%

REM Build the release packages.
call %SRC%\kokoro\windows\package.bat %cd%\out

REM Clean up - this prevents kokoro from rsyncing many unneeded files
cd %BUILD_ROOT%
rmdir /s /q github\src\github.com\google\gapid\third_party
rmdir /s /q out\release
for /d %%f in (*) do if not "%%f"=="github" if not "%%f"=="out" rmdir /s /q %%f
del /q *.zip
