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
set SRC=%cd%\github\agi

REM Use a fixed JDK.
set JAVA_HOME=c:\Program Files\Java\jdk1.8.0_144

REM Install the Android SDK components and NDK.
set ANDROID_HOME=%LOCALAPPDATA%\Android\Sdk

REM Install a license file for the Android SDK to avoid license query.
REM This file might need to be updated in the future.
copy /Y "%SRC%\kokoro\windows\android-sdk-license" "%ANDROID_HOME%\licenses\"

REM Install Android SDK platform, build tools and NDK
setlocal
call %ANDROID_HOME%\tools\bin\sdkmanager.bat platforms;android-26 build-tools;29.0.2
endlocal

wget -q https://dl.google.com/android/repository/android-ndk-r21d-windows-x86_64.zip
unzip -q android-ndk-r21d-windows-x86_64.zip
set ANDROID_NDK_HOME=%CD%\android-ndk-r21d

REM Install WiX Toolset.
wget -q https://github.com/wixtoolset/wix3/releases/download/wix311rtm/wix311-binaries.zip
unzip -q -d wix wix311-binaries.zip
set WIX=%cd%\wix

REM Manually install only the required MSYS packages. Do NOT do a
REM system update (pacman -Syu) because it is a moving target. In
REM particular, the pacman installed on default Kokoro VMs is old, and
REM supports only ".pkg.tar.xz" package archives, it does not support
REM the more recent ".pkg.tar.zst" packages archives.
c:\tools\msys64\usr\bin\bash --login -c "pacman -R --noconfirm catgets libcatgets"
REM Use an old version of patch known to work with the msys runtime
REM version that comes on Kokoro.
wget -q http://repo.msys2.org/msys/x86_64/patch-2.7.5-1-x86_64.pkg.tar.xz
c:\tools\msys64\usr\bin\bash --login -c "pacman -v -U --noconfirm  /t/src/patch-2.7.5-1-x86_64.pkg.tar.xz"
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-binutils-2.33.1-1-any.pkg.tar.xz
c:\tools\msys64\usr\bin\bash --login -c "pacman -U --noconfirm /t/src/mingw-w64-x86_64-binutils-2.33.1-1-any.pkg.tar.xz"
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-9.2.0-2-any.pkg.tar.xz
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-libs-9.2.0-2-any.pkg.tar.xz
c:\tools\msys64\usr\bin\bash --login -c "pacman -U --noconfirm /t/src/mingw-w64-x86_64-gcc-9.2.0-2-any.pkg.tar.xz /t/src/mingw-w64-x86_64-gcc-libs-9.2.0-2-any.pkg.tar.xz"
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-crt-git-8.0.0.5647.1fe2e62e-1-any.pkg.tar.xz
c:\tools\msys64\usr\bin\bash --login -c "pacman -U --noconfirm /t/src/mingw-w64-x86_64-crt-git-8.0.0.5647.1fe2e62e-1-any.pkg.tar.xz"
set PATH=c:\tools\msys64\mingw64\bin;c:\tools\msys64\usr\bin;%PATH%
set BAZEL_SH=c:\tools\msys64\usr\bin\bash

REM Get the JDK and JRE from our mirror. This needs to be after the Android update above (needs 1.8).
set JDK_BUILD=zulu11.39.15-ca
set JDK_VERSION=11.0.7
set JDK_NAME=%JDK_BUILD%-jdk%JDK_VERSION%-win_x64
set JRE_NAME=%JDK_BUILD%-jre%JDK_VERSION%-win_x64
wget -q https://storage.googleapis.com/jdk-mirror/%JDK_BUILD%/%JDK_NAME%.zip
echo "1d947cb3a846d87aca12d4ae94e1b51e1079fdcd9e4faea9684909b8072f8ed4  %JDK_NAME%.zip" | sha256sum --check
unzip -q %JDK_NAME%.zip
set JAVA_HOME=%CD%\%JDK_NAME%

wget -q https://storage.googleapis.com/jdk-mirror/%JDK_BUILD%/%JRE_NAME%.zip
echo "339f622f688b129de16876ce8ee252eac0ab7663d81596016abf2efacab01d86  %JRE_NAME%.zip" | sha256sum --check
unzip -q %JRE_NAME%.zip
set JRE_HOME=%CD%\%JRE_NAME%

REM Install Bazel.
set BAZEL_VERSION=2.0.0
wget -q https://github.com/bazelbuild/bazel/releases/download/%BAZEL_VERSION%/bazel-%BAZEL_VERSION%-windows-x86_64.zip
unzip -q bazel-%BAZEL_VERSION%-windows-x86_64.zip
set PATH=C:\python27;%PATH%

cd %SRC%

REM Invoke the build.
echo %DATE% %TIME%
if "%KOKORO_GITHUB_COMMIT%." == "." (
  set BUILD_SHA=%DEV_PREFIX%%KOKORO_GITHUB_PULL_REQUEST_COMMIT%
) else (
  set BUILD_SHA=%DEV_PREFIX%%KOKORO_GITHUB_COMMIT%
)

%BUILD_ROOT%\bazel build -c opt --config symbols ^
    --define AGI_BUILD_NUMBER="%KOKORO_BUILD_NUMBER%" ^
    --define AGI_BUILD_SHA="%BUILD_SHA%" ^
    //gapis/api/vulkan:go_default_library
if %ERRORLEVEL% GEQ 1 exit /b %ERRORLEVEL%

REM Build everything else.
%BUILD_ROOT%\bazel build -c opt --config symbols ^
    --define AGI_BUILD_NUMBER="%KOKORO_BUILD_NUMBER%" ^
    --define AGI_BUILD_SHA="%BUILD_SHA%" ^
    //:pkg //:symbols //cmd/smoketests //cmd/vulkan_sample:vulkan_sample //tools/logo:agi_ico
if %ERRORLEVEL% GEQ 1 exit /b %ERRORLEVEL%
echo %DATE% %TIME%

REM Smoketests
%SRC%\bazel-bin\cmd\smoketests\windows_amd64_stripped\smoketests -gapit bazel-bin\pkg\gapit -traces test\traces
echo %DATE% %TIME%

REM Build the release packages.
mkdir %BUILD_ROOT%\out
call %SRC%\kokoro\windows\package.bat %BUILD_ROOT%\out %SRC%
