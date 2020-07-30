# Building Android GPU Inspector

AGI uses the [Bazel build system](https://bazel.build/). The recommended version of Bazel is **2.0.0**.

Bazel is able to fetch most of the dependencies required to build AGI, but currently the Android SDK and NDK both need to be downloaded and installed by hand.

Please see the following OS specific guides for setting up the build environment.

After setting up the build environment, AGI can be built in a terminal with:

```
cd <path-to-agi-source>
bazel build pkg
```

> :warning: We currently use NDK r21d to have access to the unified Khronos
> validation layer. Bazel 2.0.0 has no official support for r21d, so **the
> following build warning is expected and can be ignored:**
>
> ```
> WARNING: The major revision of the Android NDK referenced by
> android_ndk_repository rule 'androidndk' is 21. The major
> revisions supported by Bazel are [10, 11, 12, 13, 14, 15, 16,
> 17, 18, 19, 20]. Bazel will attempt to treat the NDK as if it
> was r20. This may cause compilation and linkage problems.
> Please download a supported NDK version.
> ```
>
> This issue is tracked by https://github.com/google/agi/issues/305

The build output will be at `<path-to-agi-source>/bazel-bin/pkg`.

---

## Windows

### Install Chocolatey

[Follow these instructions](https://chocolatey.org/install) to install Chocolatey.

### Install Bazel

Start a console, with administrator privilege, and type:

`choco install bazel --version 2.0.0`

In the same console, install Python 2.7 and MSYS2 as well:

`choco install python2`
`choco install msys2`

### Install additional tools

Using the msys64 shell (installed by choco at `C:\tools\msys64\mingw64`):
1. Update MSYS with: `pacman -Syu`.
2. If the update ends with “close the window and run it again”, close and reopen the window and repeat 1.
3. Fetch required tools with: `pacman -S curl git zip unzip patch`
4. Download gcc with: `curl -O http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-9.2.0-2-any.pkg.tar.xz`
5. Download gcc-libs with: `curl -O http://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-gcc-libs-9.2.0-2-any.pkg.tar.xz`
6. Install gcc with: `pacman -U mingw-w64-x86_64-gcc*-9.2.0-2-any.pkg.tar.xz`
7. Close the MSYS terminal

### Install Java Development Kit 1.8

A JDK is required to run a custom build of AGI. If you do not already have a suitable JDK installed,
you can [download the OpenJDK](http://jdk-mirror.storage.googleapis.com/index.html) we use on our
build bots.

Make sure the `JAVA_HOME` environment variable points to the JDK.

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-windows-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools\bin\sdkmanager.bat "platforms;android-26" "build-tools;29.0.2"
```

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools\bin\sdkmanager.bat platform-tools
```

Unzip the
[Android NDK **r21d**](https://dl.google.com/android/repository/android-ndk-r21d-windows-x86_64.zip)
into a directory of your choosing, and set the `ANDROID_NDK_HOME` environment
variable to point to this directory.

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

### Install Java Development Kit 1.8

A JDK is required to run a custom build of AGI. If you do not already have a suitable JDK installed,
you can [download the OpenJDK](http://jdk-mirror.storage.googleapis.com/index.html) we use on our
build bots.

Make sure the `JAVA_HOME` environment variable points to the JDK.

> :warning: If you find the application menu bar non-responsive when launching your build of AGI,
> the following command should fix it:
> `sudo mkdir $JAVA_HOME/bin/Contents`

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-darwin-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-26" "build-tools;29.0.2"
```

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools/bin/sdkmanager platform-tools
```

Install
[Android NDK **r21d**](https://dl.google.com/android/repository/android-ndk-r21d-darwin-x86_64.dmg) (installing the App Bundle is needed in order to keep the notarization) 
into the /Applications/ folder, and set the `ANDROID_NDK_HOME` environment pointing to NDK subdirectory:

```
export ANDROID_NDK_HOME=/Applications/AndroidNDK6528147.app/Contents/NDK
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

### Install Java Development Kit 1.8

A JDK is required to run a custom build of AGI. If you do not already have a suitable JDK installed,
you can [download the OpenJDK](http://jdk-mirror.storage.googleapis.com/index.html) we use on our
build bots.

Make sure the `JAVA_HOME` environment variable points to the JDK.

### Install Android SDK and NDK

Unzip the [Android SDK](https://dl.google.com/android/repository/sdk-tools-linux-3859397.zip) to a directory of your choosing.

To fetch the required packages, using a console type:

```
cd <sdk-path>
tools/bin/sdkmanager "platforms;android-26" "build-tools;29.0.2"
```

If you do not have adb installed you can do so with:
```
cd <sdk-path>
tools/bin/sdkmanager platform-tools
```

Unzip the
[Android NDK **r21d**](https://dl.google.com/android/repository/android-ndk-r21d-linux-x86_64.zip)
into a directory of your choosing, and set the `ANDROID_NDK_HOME` environment
variable to point to this directory:

```
export ANDROID_NDK_HOME=<ndk-path>
```

### Install other libraries

```
sudo apt-get update
sudo apt-get install mesa-common-dev libncurses5-dev libgl1-mesa-dev zlib1g-dev
```

### Configure the environment

Either do this globally or in your shell every time.

Make sure the environment is setup before you run bazel (`bazel shutdown` will shut it down).

The following environment variables will need to be set prior to building:

| Variable            | Target              |
| ------------------- | ------------------- |
| `ANDROID_HOME`      | Path to Android SDK |
| `ANDROID_NDK_HOME`  | Path to Android NDK |
