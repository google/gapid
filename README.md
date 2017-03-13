# GAPID: **G**raphics **API** **D**ebugger

[![GoDoc](https://godoc.org/github.com/google/gapid?status.svg)](https://godoc.org/github.com/google/gapid)

GAPID is a collection of tools that allows you to inspect, tweak and replay calls from an application to a graphics driver. 

GAPID can trace any Android [debuggable application](https://developer.android.com/guide/topics/manifest/application-element.html#debug), or if you have root access to the device any application can be traced.

## Status
GAPID is still in development but already can be used to debug many Android OpenGL ES and Vulkan applications.

The UI runs on Windows, Linux and MacOS and can currently be used to trace on Android.
We also plan to be able to trace OpenGL ES and Vulkan applications on hosts that support those APIs.

Downloadable prebuilts will be available once the project reaches the beta milestone.

Detailed current status for Vulkan can be found [here](gapis/gfxapi/vulkan/README.md).

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
