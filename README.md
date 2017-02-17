# GAPID: **G**raphics **API** **D**ebugger

GAPID is a collection of tools that allows you to inspect, tweak and replay GPU command streams captured from any Android application.

GAPID consists of the following sub-components:

### GAPI**I**: Graphics API Interceptor
A layer that sits between the application / game and the GPU driver, recording all the calls and memory accesses.

### GAPI**S**: Graphics API Server
A process that analyses capture streams reporting incorrect API usage, processes the data for replay on various target devices, and provides an RPC interface to the client.

### GAPI**R**: Graphics API Replay daemon
A stack-based VM used to playback capture files, imitating the original application’s / game's calls to the GPU driver. Supports read-back of any buffer / framebuffer, and provides profiling functionality.

### GAPI**C**: Graphics API Client
The frontend user interface application. Provides visual inspection of the capture data, memory, resources, and frame-buffer content.

### GAPI**L**: Graphics API Language
A new domain specific language to describe a graphics API in its entirety. Combined with our template system to generate huge parts of the interceptor, server and replay systems.

### APIC: The API compiler
Used to validate and format API files, and when combined with templates, used to generate code.

## Dependencies
* Go 1.7 (must be on your PATH)
* Android SDK ([windows](https://dl.google.com/android/repository/tools_r25.2.3-windows.zip), [osx](https://dl.google.com/android/repository/tools_r25.2.3-macosx.zip), [linux](https://dl.google.com/android/repository/tools_r25.2.3-linux.zip))
  * Requires Android 5.0.1 (API 21)
  * Requires Android SDK Build-tools 21.1.2
* Android NDK ([windows](https://dl.google.com/android/repository/android-ndk-r13b-windows-x86_64.zip), [osx](https://dl.google.com/android/repository/android-ndk-r13b-darwin-x86_64.zip), [linux](https://dl.google.com/android/repository/android-ndk-r13b-linux-x86_64.zip))
* CMake ([windows](https://cmake.org/files/v3.7/cmake-3.7.1-win32-x86.msi), [osx](https://cmake.org/files/v3.7/cmake-3.7.1-Darwin-x86_64.dmg), [linux](https://cmake.org/files/v3.7/cmake-3.7.1-Linux-x86_64.sh))
* [Ninja](https://ninja-build.org/)
* Optional: [NodeJS](https://nodejs.org/en/download/) for building the `.api` Visual Studio Code extension.

## Getting the code
```
go get github.com/google/gapid
cd $GOPATH/src/github.com/google/gapid
git submodule update --init
```

## Building
Run `./do` and follow the instructions

