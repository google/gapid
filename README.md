# GAPID: **G**raphics **API** **D**ebugger

[![GoDoc](https://godoc.org/github.com/google/gapid?status.svg)](https://godoc.org/github.com/google/gapid)
[![Gitter](https://badges.gitter.im/google/gapid.svg)](https://gitter.im/google/gapid?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
<img alt="Linux" src="kokoro/img/linux.png" width="20px" height="20px" hspace="2px"/>![Linux Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_linux.svg)
<img alt="MacOS" src="kokoro/img/macos.png" width="20px" height="20px" hspace="2px"/>![MacOS Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_macos.svg)
<img alt="Windows" src="kokoro/img/windows.png" width="20px" height="20px" hspace="2px"/>![Windows Build Status](https://gapid-build.storage.googleapis.com/badges/build_status_windows.svg)


<p>
  <a href="https://github.com/google/gapid/releases">
    <b>Download the latest version of GAPID here.</b>
  </a>
</p>

## About

GAPID is a collection of tools that allows you to inspect, tweak and replay calls from an application to a graphics driver.

GAPID can trace any Android [debuggable application](https://developer.android.com/guide/topics/manifest/application-element.html#debug), or if you have root access to the device any application can be traced.
GAPID can also trace any Desktop Vulkan application.

<table>
  <tr>
    <td>
      <a href="https://google.github.io/gapid/images/screenshots/framebuffer.png">
        <img src="https://google.github.io/gapid/images/screenshots/framebuffer_thumb.jpg" alt="Screenshot 1">
      </a>
    </td>
    <td>
      <a href="https://google.github.io/gapid/images/screenshots/geometry.png">
        <img src="https://google.github.io/gapid/images/screenshots/geometry_thumb.jpg" alt="Screenshot 2">
      </a>
    </td>
  </tr>
  <tr>
    <td>
      <a href="https://google.github.io/gapid/images/screenshots/textures.png">
        <img src="https://google.github.io/gapid/images/screenshots/textures_thumb.jpg" alt="Screenshot 3">
      </a>
    </td>
    <td>
      <a href="https://google.github.io/gapid/images/screenshots/shaders.png">
        <img src="https://google.github.io/gapid/images/screenshots/shaders_thumb.jpg" alt="Screenshot 4">
      </a>
    </td>
  </tr>
</table>

## Status
GAPID is still in development but already can be used to debug many Android OpenGL ES and Vulkan applications.

The UI runs on Windows, Linux and MacOS and can currently be used to trace GLES on Android as well as Vulkan
on Windows, Linux and Android.
We also plan to be able to trace OpenGL ES applications on hosts that support the API.

Pre-release downloadable binaries can be found [here](https://github.com/google/gapid/releases).

Detailed current status for Vulkan can be found [here](gapis/api/vulkan/README.md).

## Building
See [Building GAPID](BUILDING.md).

## Running the client

<table>
  <tr>
    <th>Windows</th>
    <th>MacOS / Linux</th>
  </tr>
  <tr>
    <td><pre>cd %GOPATH%\src\github.com\google\gapid<br>do run gapic</pre></td>
    <td><pre>cd $GOPATH/src/github.com/google/gapid<br>./do run gapic</pre></td>
  </tr>
</table>

## Overview
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
