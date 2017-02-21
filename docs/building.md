# Building GAPID

## Initial setup

---

### Windows

#### Install [Go 1.7](https://storage.googleapis.com/golang/go1.7.5.windows-amd64.msi)

#### Install Toolchain
* Install [msys2](http://repo.msys2.org/distrib/x86_64/msys2-x86_64-20161025.exe).
* Open the msys2 terminal.
* Type: `pacman -Syu --noconfirm` and press enter.
* Close and reopen the msys2 terminal.
* Type: `pacman -S mingw-w64-x86_64-gcc --noconfirm` and press enter.
* Close the msys2 terminal

#### Install Android SDK
* Unzip the [Android SDK](https://dl.google.com/android/repository/tools_r25.2.3-windows.zip) to a directory of your choosing.
* Using `sdk\tools\android.bat` download:
  ** Android 5.0.1 (API 21)
  ** Android SDK Build-tools 21.1.2

#### Install the [Android NDK](https://dl.google.com/android/repository/android-ndk-r13b-windows-x86_64.zip)

#### Install [CMake](https://cmake.org/files/v3.7/cmake-3.7.1-win32-x86.msi).

#### Install [Ninja](https://ninja-build.org/).

#### Install [Python 3.6](https://www.python.org/ftp/python/3.6.0/python-3.6.0-amd64.exe)

---

### MacOS

#### Install [Go 1.7](https://storage.googleapis.com/golang/go1.7.5.darwin-amd64.pkg)

#### Install Android SDK
* Unzip the [Android SDK](https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip) to a directory of your choosing.
* Using `sdk\tools\android` download:
  ** Android 5.0.1 (API 21)
  ** Android SDK Build-tools 21.1.2

#### Install Android NDK
* Unzip the [Android NDK](https://dl.google.com/android/repository/android-ndk-r13b-darwin-x86_64.zip) to a directory of your choosing.

#### Install [CMake](https://cmake.org/files/v3.7/cmake-3.7.1-Darwin-x86_64.dmg).

#### Install [Ninja](https://ninja-build.org/).

#### Install [Python 3.6](https://www.python.org/ftp/python/3.6.0/python-3.6.0-macosx10.6.pkg)

---

### Linux

#### Install [Go 1.7](https://storage.googleapis.com/golang/go1.7.5.linux-amd64.tar.gz)

#### Install Android SDK
* Unzip the [Android SDK](https://dl.google.com/android/repository/tools_r25.2.3-linux.zip) to a directory of your choosing.
* Using `sdk\tools\android` download:
  ** Android 5.0.1 (API 21)
  ** Android SDK Build-tools 21.1.2

#### Install Android NDK
* Unzip the [Android NDK](https://dl.google.com/android/repository/android-ndk-r13b-linux-x86_64.zip) to a directory of your choosing.

#### Install [CMake](https://cmake.org/files/v3.7/cmake-3.7.1-Linux-x86_64.sh).

#### Install [Ninja](https://ninja-build.org/).

#### Install Python 3.6

