# Building GAPID

GAPID uses the [Bazel build system](https://bazel.build/). The current minimum required version of bazel is 0.25.1.

Bazel is able to fetch most of the dependencies required to build GAPID, but currently the Android SDK and NDK both need to be downloaded and installed by hand.

Please see the following OS specific guides for setting up the build environment.

After setting up the build environment, GAPID can be built in a terminal with:

```
cd <path-to-gapid-source>
bazel build pkg
```

The build output will be at `<path-to-gapid-source>/bazel-bin/pkg`.

---

## Windows

### Install Chocolatey

[Follow these instructions](https://chocolatey.org/install) to install Chocolatey.

### Install Bazel

In the console (with administrator privilege) type:

`choco install bazel`

Note: Installing bazel will also install MSYS into `C:\tools\msys64` and Python into `C:\tools\python27`.

### Install additional tools

Using the msys64 shell at `C:\tools\msys64\mingw64`:
1. Update MSYS with: `pacman -Syu`.
2. If the update ends with “close the window and run it again”, close and reopen the window and repeat 1.
3. Fetch required tools with: `pacman -S curl git zip unzip patch`
4. Download gcc with: `curl -O http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-7.3.0-2-any.pkg.tar.xz`
5. Download gcc-libs with: `curl -O http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-libs-7.3.0-2-any.pkg.tar.xz`
6. Install gcc with: `pacman -U mingw-w64-x86_64-gcc*-7.3.0-2-any.pkg.tar.xz`
7. Close the MSYS terminal

### Install Java Runtime 8

A JRE is required by the Android SDK tools. If you do not already have a JRE installed, please [install the Java Runtime Environment from here](http://www.oracle.com/technetwork/java/javase/downloads/jre8-downloads-2133155.html).

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-windows-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools\bin\sdkmanager.bat "platforms;android-26" "build-tools;26.0.1" ndk-bundle
```

Note: this will install the latest NDK in `<sdk-path>\ndk-bundle`. The minimum required version of the NDK is r16b.

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools\bin\sdkmanager.bat platform-tools
```

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

1. Add `C:\tools\msys64\mingw64` to the PATH:
   `set PATH=C:\tools\msys64\mingw64\bin;%PATH%`
   Running `where gcc` should now find mingw’s gcc.

1. Add `C:\tools\python27` to the PATH:
   `set PATH=C:\tools\python27;%PATH%`
   Alternatively, pass the path to python via the `--python_path` to bazel. See the [bazel documentation](https://docs.bazel.build/versions/master/windows.html#build-python) for more info.

1. Set TMP to something very short. `C:\tmp` is known to work. For faster builds, add this folder to the excemptions of the Windows Defender anti-malware scanner.

The following environment variables will need to be set prior to building:

| Variable            | Target                             |
| ------------------- | ---------------------------------- |
| `ANDROID_HOME`      | Path to Android SDK                |
| `ANDROID_NDK_HOME`  | Path to Android NDK                |
| `BAZEL_SH`          | `C:\tools\msys64\usr\bin\bash.exe` |
| `TMP`               | `C:\tmp`                           |

---

## MacOS

### Install Bazel

Follow the [MacOS Bazel Install](https://docs.bazel.build/versions/master/install-os-x.html) directions to install bazel.

### Install Java Runtime 8

A JRE is required by the Android SDK tools. If you do not already have a JRE installed, please [install the Java Runtime Environment from here](http://www.oracle.com/technetwork/java/javase/downloads/jre8-downloads-2133155.html).

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-darwin-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-26" "build-tools;26.0.1" ndk-bundle
```

Note: this will install the latest NDK in `<sdk-path>/ndk-bundle`. The minimum required version of the NDK is r16b.

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools/bin/sdkmanager platform-tools
```

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

| Variable            | Target               |
| ------------------- | -------------------- |
| `ANDROID_HOME`      | Path to Android SDK  |
| `ANDROID_NDK_HOME`  | Path to Android NDK  |

---

## Linux

### Install Bazel

Follow the [Ubuntu Bazel Install](https://docs.bazel.build/versions/master/install-ubuntu.html) or the[Fedora/CentOS Bazel Install](https://docs.bazel.build/versions/master/install-redhat.html) directions to install bazel.

Alternatively, bazel can be downloaded from its [GitHub Releases Page](https://github.com/bazelbuild/bazel/releases).

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-linux-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-26" "build-tools;26.0.1" ndk-bundle
```

Note: this will install the latest NDK in `<sdk-path>/ndk-bundle`. The minimum required version of the NDK is r16b.

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools/bin/sdkmanager platform-tools
```

### Install other libraries

```
sudo apt-get update
sudo apt-get install mesa-common-dev libncurses5-dev libgl1-mesa-dev
```

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

The following environment variables will need to be set prior to building:

| Variable            | Target              |
| ------------------- | ------------------- |
| `ANDROID_HOME`      | Path to Android SDK |
| `ANDROID_NDK_HOME`  | Path to Android NDK |
