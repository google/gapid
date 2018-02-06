# Building GAPID

GAPID uses the [Bazel build system](https://bazel.build/).

Bazel is able to fetch most of the dependencies required to build GAPID, but currently the Android SDK and NDK both need to be downloaded and installed by hand.

Please see the following OS specific guides for setting up the build environment.

## Windows

### Install Chocolatey

[Follow these instructions](https://chocolatey.org/install) to install Chocolatey.

### Install Java Runtime 8

[Install the Java Runtime Environment from here](http://www.oracle.com/technetwork/java/javase/downloads/jre8-downloads-2133155.html).

### Install additional tools

Using the msys64 shell at `c:\tools\msys64\mingw64`:
1. Update MSYS with: `pacman -Syu`.
2. If the update ends with “close the window and run it again”, close and reopen the window and repeat 1.
3. Fetch required tools with: `pacman -S mingw-w64-x86_64-gcc curl git zip unzip`
4. Close the MSYS terminal

### Install Bazel

In the console type:

`choco install bazel`

Note: Installing bazel will also install MSYS into `c:\tools\msys64`.

### Install Android SDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-windows-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools\bin\sdkmanager "platforms;android-21"
tools\bin\sdkmanager "platforms;android-27"
tools\bin\sdkmanager "build-tools;26.0.1
```

### Install the Android NDK

Unzip the [Android NDK](https://dl.google.com/android/repository/android-ndk-r15b-windows-x86_64.zip) to a directory of your choosing.

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

1. Make sure there is no instance of go on your `PATH`
Run `where go`. If a version of go is found, please remove it from `PATH`.

1. Add `c:\tools\msys64\mingw64` to the PATH 
`set PATH=c:\tools\msys64\mingw64\bin;%PATH%`

Running `where gcc` should now find mingw’s gcc.

The following environment variables will need to be set prior to building:

| Variable            | Target                            |
| ------------------- | --------------------------------- |
| `JAVA_HOME`         | Path to the Java Runtime          |
| `ANDROID_HOME`      | Path to Android SDK               |
| `ANDROID_NDK_HOME`  | Path to Android NDK               |
| `BAZEL_SH`          | `C:\tools\msys64\usr\bin\bash.exe`|
| `GOPATH`            | \<unset\>                         |
| `GOROOT`            | \<unset\>                         |
| `TMP`               | `c:\tmp`                          |


### Building

In a terminal type:
```
cd <path-to-gapid-source>
bazel build --config mingw pkg
```
The build output will be at `bazel-bin/pkg`.

---

## MacOS

### Install Java Runtime 8

[Install the Java Runtime Environment from here](http://www.oracle.com/technetwork/java/javase/downloads/jre8-downloads-2133155.html).

### Install Android SDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-darwin-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-21"
tools/bin/sdkmanager "platforms;android-27"
tools/bin/sdkmanager "build-tools;26.0.1
```

### Install Android NDK

Unzip the [Android NDK](https://dl.google.com/android/repository/android-ndk-r15b-darwin-x86_64.zip) to a directory of your choosing.

### Install the XCode command line tools

After installing, ensure the XCode license is signed with:

```
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
sudo xcodebuild -license
```

### Increase the maximum number of OS file handles

Bazel can concurrently use more file handles than the OS supports by default. This can be easily fixed by typing:

```
sudo sysctl -w kern.maxfiles=122880
sudo sysctl -w kern.maxfilesperproc=102400
echo ulimit -S -n 102400 >> ~/.bashrc
```

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

The following environment variables will need to be set prior to building:

| Variable            | Target                            |
| ------------------- | --------------------------------- |
| `JAVA_HOME`         | Path to the Java Runtime          |
| `ANDROID_HOME`      | Path to Android SDK               |
| `ANDROID_NDK_HOME`  | Path to Android NDK               |
| `GOPATH`            | <unset>                           |
| `GOROOT`            | <unset>                           |

### Building

In a terminal type:
```
cd <path-to-gapid-source>
bazel build build pkg
```
The build output will be at `bazel-bin/pkg`.

---

## Linux

### Install Java Runtime 8

[Install the Java Runtime Environment from here](http://www.oracle.com/technetwork/java/javase/downloads/jre8-downloads-2133155.html).

### Install Android SDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-linux-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-21"
tools/bin/sdkmanager "platforms;android-27"
tools/bin/sdkmanager "build-tools;26.0.1
```

### Install Android NDK
* Unzip the [Android NDK](https://dl.google.com/android/repository/android-ndk-r15b-linux-x86_64.zip) to a directory of your choosing.

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

The following environment variables will need to be set prior to building:

| Variable            | Target                            |
| ------------------- | --------------------------------- |
| `JAVA_HOME`         | Path to the Java Runtime          |
| `ANDROID_HOME`      | Path to Android SDK               |
| `ANDROID_NDK_HOME`  | Path to Android NDK               |
| `GOPATH`            | <unset>                           |
| `GOROOT`            | <unset>                           |

### Building

In a terminal type:
```
cd <path-to-gapid-source>
bazel build build pkg
```
The build output will be at `bazel-bin/pkg`.

