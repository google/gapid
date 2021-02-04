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
set JAVA_HOME=c:\Program Files\Java\jdk1.8.0_152

REM Install the Android SDK components and NDK.
set ANDROID_HOME=c:\Android\android-sdk

REM Install a license file for the Android SDK to avoid license query.
REM This file might need to be updated in the future.
copy /Y "%SRC%\kokoro\windows\android-sdk-license" "%ANDROID_HOME%\licenses\"

REM Install Android SDK platform, build tools and NDK
setlocal
call %ANDROID_HOME%\tools\bin\sdkmanager.bat platforms;android-26 build-tools;29.0.2
endlocal
echo on

wget -q https://dl.google.com/android/repository/android-ndk-r21d-windows-x86_64.zip
unzip -q android-ndk-r21d-windows-x86_64.zip
set ANDROID_NDK_HOME=%CD%\android-ndk-r21d

REM Download and install MSYS2, because the pre-installed version is too old.
REM Do NOT do a system update (pacman -Syu) because it is a moving target.
wget -q https://github.com/msys2/msys2-installer/releases/download/2020-11-09/msys2-base-x86_64-20201109.sfx.exe
.\msys2-base-x86_64-20201109.sfx.exe -y -o%BUILD_ROOT%\

REM Start empty shell to initialize MSYS2.
%BUILD_ROOT%\msys64\usr\bin\bash --login -c " "

REM Uncomment the following line to list all packages and versions installed in MSYS2.
REM %BUILD_ROOT%\msys64\usr\bin\bash --login -c "pacman -Q"

REM Download and install packages required by the build process.
wget -q http://repo.msys2.org/msys/x86_64/git-2.29.2-1-x86_64.pkg.tar.zst
wget -q http://repo.msys2.org/msys/x86_64/patch-2.7.6-1-x86_64.pkg.tar.xz
wget -q http://repo.msys2.org/msys/x86_64/unzip-6.0-2-x86_64.pkg.tar.xz
wget -q http://repo.msys2.org/msys/x86_64/zip-3.0-3-x86_64.pkg.tar.xz
%BUILD_ROOT%\msys64\usr\bin\bash --login -c "pacman -U --noconfirm /t/src/git-2.29.2-1-x86_64.pkg.tar.zst /t/src/patch-2.7.6-1-x86_64.pkg.tar.xz /t/src/unzip-6.0-2-x86_64.pkg.tar.xz /t/src/zip-3.0-3-x86_64.pkg.tar.xz"

REM Download and install specific compiler version.
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-10.2.0-5-any.pkg.tar.zst
wget -q http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-libs-10.2.0-5-any.pkg.tar.zst
%BUILD_ROOT%\msys64\usr\bin\bash --login -c "pacman -U --noconfirm /t/src/mingw-w64-x86_64-gcc-10.2.0-5-any.pkg.tar.zst /t/src/mingw-w64-x86_64-gcc-libs-10.2.0-5-any.pkg.tar.zst"

REM Configure build process to use the now installed MSYS2.
set PATH=%BUILD_ROOT%\msys64\mingw64\bin;%BUILD_ROOT%\msys64\usr\bin;%PATH%
set BAZEL_SH=%BUILD_ROOT%\msys64\usr\bin\bash

REM Get the JDK from our mirror.
set JDK_BUILD=zulu8.46.0.19-ca
set JDK_VERSION=8.0.252
set JDK_NAME=%JDK_BUILD%-jdk%JDK_VERSION%-win_x64
set JRE_NAME=%JDK_BUILD%-jre%JDK_VERSION%-win_x64
wget -q https://storage.googleapis.com/jdk-mirror/%JDK_BUILD%/%JDK_NAME%.zip
echo "993ef31276d18446ef8b0c249b40aa2dfcea221a5725d9466cbea1ba22686f6b  %JDK_NAME%.zip" | sha256sum --check
unzip -q %JDK_NAME%.zip
set JAVA_HOME=%CD%\%JDK_NAME%

wget -q https://storage.googleapis.com/jdk-mirror/%JDK_BUILD%/%JRE_NAME%.zip
echo "cf5cc2b5bf1206ace9b035dee129a144eda3059f43f204a4ba5e6911d95f0d0c  %JRE_NAME%.zip" | sha256sum --check
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

REM Configure package script to use the pre-installed WiX Toolset.
set WIX=%ProgramFiles(x86)%\WiX Toolset v3.11\bin

REM Build the release packages.
mkdir %BUILD_ROOT%\out
call %SRC%\kokoro\windows\package.bat %BUILD_ROOT%\out %SRC%
