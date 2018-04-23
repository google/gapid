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
set BUILD_ROOT=%cd%
set SRC=%cd%\github\gapid

REM Use a fixed JDK.
set JAVA_HOME=c:\Program Files\Java\jdk1.8.0_144

REM Install the Android SDK components and NDK.
set ANDROID_HOME=%LOCALAPPDATA%\Android\Sdk
echo y | %ANDROID_HOME%\tools\bin\sdkmanager build-tools;26.0.1 platforms;android-21
wget -q https://dl.google.com/android/repository/android-ndk-r16b-windows-x86_64.zip
unzip -q android-ndk-r16b-windows-x86_64.zip
set ANDROID_NDK_HOME=%CD%\android-ndk-r16b

REM Install WiX Toolset.
wget -q https://github.com/wixtoolset/wix3/releases/download/wix311rtm/wix311-binaries.zip
unzip -q -d wix wix311-binaries.zip
set WIX=%cd%\wix

REM Fix up the MSYS environment.
c:\tools\msys64\usr\bin\bash --login -c "pacman -R --noconfirm catgets libcatgets"
c:\tools\msys64\usr\bin\bash --login -c "pacman -Syu --noconfirm"
c:\tools\msys64\usr\bin\bash --login -c "pacman -S --noconfirm mingw-w64-x86_64-gcc patch"
set PATH=c:\tools\msys64\mingw64\bin;%PATH%

REM TODO set up msvc build env
REM set Platform="X64"
REM set PreferredToolArchitecture="x64"
REM call "C:\Program Files (x86)\Microsoft Visual Studio 14.0\VC\vcvarsall.bat" amd64

REM Install Bazel.
wget -q https://github.com/bazelbuild/bazel/releases/download/0.12.0/bazel-0.12.0-without-jdk-windows-x86_64.zip
unzip -q bazel-0.12.0-without-jdk-windows-x86_64.zip
set PATH=C:\python27;%PATH%

cd %SRC%

REM Invoke the build.
echo %DATE% %TIME%
if "%KOKORO_GITHUB_COMMIT%." == "." (
  set BUILD_SHA=%KOKORO_GITHUB_PULL_REQUEST_COMMIT%
) else (
  set BUILD_SHA=%KOKORO_GITHUB_COMMIT%
)
%BUILD_ROOT%\bazel build -c opt --config symbols ^
    --define GAPID_BUILD_NUMBER="%KOKORO_BUILD_NUMBER%" ^
    --define GAPID_BUILD_SHA="%BUILD_SHA%" ^
    //:pkg //cmd/gapir/cc:gapir.sym
if %ERRORLEVEL% GEQ 1 exit /b %ERRORLEVEL%
echo %DATE% %TIME%
cd %BUILD_ROOT%

REM Build the release packages.
mkdir %BUILD_ROOT%\out
call %SRC%\kokoro\windows\package.bat %BUILD_ROOT%\out %SRC%
