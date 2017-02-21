# Building GAPID

## Common dependencies
* Go 1.7 (must be on your PATH)
* Android SDK ([windows](https://dl.google.com/android/repository/tools_r25.2.3-windows.zip), [osx](https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip), [linux](https://dl.google.com/android/repository/tools_r25.2.3-linux.zip))
  * Requires Android 5.0.1 (API 21)
  * Requires Android SDK Build-tools 21.1.2
* Android NDK ([windows](https://dl.google.com/android/repository/android-ndk-r13b-windows-x86_64.zip), [osx](https://dl.google.com/android/repository/android-ndk-r13b-darwin-x86_64.zip), [linux](https://dl.google.com/android/repository/android-ndk-r13b-linux-x86_64.zip))
* CMake ([windows](https://cmake.org/files/v3.7/cmake-3.7.1-win32-x86.msi), [osx](https://cmake.org/files/v3.7/cmake-3.7.1-Darwin-x86_64.dmg), [linux](https://cmake.org/files/v3.7/cmake-3.7.1-Linux-x86_64.sh))
* [Ninja](https://ninja-build.org/)
* Optional: [NodeJS](https://nodejs.org/en/download/) for building the `.api` Visual Studio Code extension.

## Windows

### Setting up toolchain
* Install [msys2](http://repo.msys2.org/distrib/x86_64/msys2-x86_64-20161025.exe).
* Open the msys2 terminal.
* Type: `pacman -Syu --noconfirm` and press enter.
* Close and reopen the msys2 terminal.
* Type: `pacman -S mingw-w64-x86_64-gcc --noconfirm` and press enter.
* Close the msys2 terminal
