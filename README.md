# Android GPU Inspector

<!-- TODO(b/155159330) Once we reach a stable release, re-enabled godoc and switch to pkg.go.dev, see https://go.dev/about#adding-a-package -->
<!-- [![GoDoc](https://godoc.org/github.com/google/gapid?status.svg)](https://godoc.org/github.com/google/gapid) -->
![]() <!-- Empty image needed to have the markdown parser correctly parse the following lines -->
<img alt="Linux" src="kokoro/img/linux.png" width="20px" height="20px" hspace="2px"/>
[![Linux Build Status](https://agi-build.storage.googleapis.com/badges/build_status_linux.svg)](https://agi-build.storage.googleapis.com/badges/build_result_linux.html)
<img alt="MacOS" src="kokoro/img/macos.png" width="20px" height="20px" hspace="2px"/>
[![MacOS Build Status](https://agi-build.storage.googleapis.com/badges/build_status_macos.svg)](https://agi-build.storage.googleapis.com/badges/build_result_macos.html)
<img alt="Windows" src="kokoro/img/windows.png" width="20px" height="20px" hspace="2px"/>
[![Windows Build Status](https://agi-build.storage.googleapis.com/badges/build_status_windows.svg)](https://agi-build.storage.googleapis.com/badges/build_result_windows.html)

## About

Visit [gpuinspector.dev](https://gpuinspector.dev) for information about Android GPU Inspector.

The [developer documentation](DEVDOC.md) contains some hints for AGI
developers. See also the README files under some source directories.

## Downloads

**[Download the latest version of AGI here.](https://github.com/google/agi/releases)**

*Unstable* developer releases are [here](https://github.com/google/agi-dev-releases/releases).

> Dependencies for Linux builds in zip archives: AGI depends on openjdk-11-jre,
> libgtk-3-0, and libwebkit2gtk. These are marked as dependencies in the deb
> package. If you install AGI via a zip archive, make sure to install these
> dependencies as well.

## Building

**See [Building Android GPU Inspector](BUILDING.md).**

## Running the client

After building AGI, you can run the client from `<agi-root>/bazel-bin/pkg/agi`.
