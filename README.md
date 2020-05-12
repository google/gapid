# GAPID: Graphics API Debugger

[![GoDoc](https://godoc.org/github.com/google/gapid?status.svg)](https://godoc.org/github.com/google/gapid)
<img alt="Linux" src="kokoro/img/linux.png" width="20px" height="20px" hspace="2px"/>
[![Linux Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_linux.svg)](https://gapid-build.storage.googleapis.com/badges/build_result_linux.html)
<img alt="MacOS" src="kokoro/img/macos.png" width="20px" height="20px" hspace="2px"/>
[![MacOS Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_macos.svg)](https://gapid-build.storage.googleapis.com/badges/build_result_macos.html)
<img alt="Windows" src="kokoro/img/windows.png" width="20px" height="20px" hspace="2px"/>
[![Windows Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_windows.svg)](https://gapid-build.storage.googleapis.com/badges/build_result_windows.html)

## Maintenance Mode

GAPID is not currently under active development. Issues filed against this repo might not be answered.

For Android GPU debugging and profiling, please use [AGI](https://github.com/google/agi) which is based on GAPID.

## Downloads

**[Download the latest version of GAPID here.](https://github.com/google/gapid/releases)**

*Unstable* developer releases are [here](https://github.com/google/gapid-dev-releases/releases).

## Documentation

**[User documentation can be found at gapid.dev](https://gapid.dev)**

The [developer documentation](DEVDOC.md) contains some hints for GAPID
developers. See also the README files under some source directories.

## About

GAPID is a collection of tools that allows you to inspect, tweak and replay calls from an application to a graphics driver.

GAPID can trace any Android [debuggable application](https://developer.android.com/guide/topics/manifest/application-element.html#debug), or if you have root access to the device any application can be traced.
GAPID can also trace any desktop Vulkan application.

<table>
  <tr>
    <td>
      <a href="https://gapid.dev/images/screenshots/framebuffer.png">
        <img src="https://gapid.dev/images/screenshots/framebuffer_thumb.jpg" alt="Screenshot 1">
      </a>
    </td>
    <td>
      <a href="https://gapid.dev/images/screenshots/geometry.png">
        <img src="https://gapid.dev/images/screenshots/geometry_thumb.jpg" alt="Screenshot 2">
      </a>
    </td>
  </tr>
  <tr>
    <td>
      <a href="https://gapid.dev/images/screenshots/textures.png">
        <img src="https://gapid.dev/images/screenshots/textures_thumb.jpg" alt="Screenshot 3">
      </a>
    </td>
    <td>
      <a href="https://gapid.dev/images/screenshots/shaders.png">
        <img src="https://gapid.dev/images/screenshots/shaders_thumb.jpg" alt="Screenshot 4">
      </a>
    </td>
  </tr>
</table>

## Building

**See [Building GAPID](BUILDING.md).**

## Running the client

After building GAPID, you can run the client from `<gapid-root>/bazel-bin/pkg/gapid`.

## Command-Line Interface

GAPID exposes most of its functionality via a CLI *gapit*. You can find auto-generated documentation [here](https://gapid.dev/cli/).

## Project Structure

GAPID consists of the following sub-components:

### [`gapii`](gapii): Graphics API Interceptor
A layer that sits between the application / game and the GPU driver, recording all the calls and memory accesses.

### [`gapis`](gapis): Graphics API Server
A process that analyses capture streams reporting incorrect API usage, processes the data for replay on various target devices, and provides an RPC interface to the client.

### [`gapir`](gapir): Graphics API Replay daemon
A stack-based VM used to playback capture files, imitating the original application’s / game's calls to the GPU driver. Supports read-back of any buffer / framebuffer, and provides profiling functionality.

### [`gapic`](gapic): Graphics API Client
The frontend user interface application. Provides visual inspection of the capture data, memory, resources, and frame-buffer content.

### [`gapil`](gapil): Graphics API Language
A new domain specific language to describe a graphics API in its entirety. Combined with our template system to generate huge parts of the interceptor, server and replay systems.

