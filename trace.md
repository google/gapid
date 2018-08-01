---
layout: default
title: Capturing a trace
sidebar: Capturing a trace
order: 10
permalink: /trace/
group: trace
---

GAPID supports capturing from both Android devices and Windows/Linux desktop machines. On Android devices, GAPID supports tracing all OpenGL ES and Vulkan calls made by either a pure Java, native or hybrid application. On Windows/Linux desktop machines, GAPID supports tracing Vulkan calls.

## Dependencies and prerequisites

### Android

* A device running Android Lollipop 5.0 (or more recent).
* Either a [debuggable](https://developer.android.com/guide/topics/manifest/application-element.html#debug) application, or a device running a 'rooted' user-debug build.
* Android SDK installed on the host machine.
* Android hardware device connected through USB.
* The device must have [USB debugging enabled](https://developer.android.com/studio/debug/dev-options.html) and the host machine must be authorized for debugging.

### Windows/Linux

* A Vulkan application compiled with x86-64. 32-bit applications are **NOT** currently supported. 

## Taking a capture

Click the `Capture Trace...` text in the welcome screen, or click the `File` &rarr; `Capture Trace` toolbar item to open the trace dialog.

<div class="callout-img">
  <div style="margin: 43px 130px">1</div>
  <div style="margin: 73px 130px">2</div>
  <div style="margin: 103px 130px">3</div>
  <div style="margin: 135px 200px">4</div>
  <div style="margin: 165px 200px">5</div>
  <div style="margin: 195px 200px">6</div>
  <div style="margin: 225px 130px">7</div>
  <div style="margin: 255px 130px">8</div>
  <div style="margin: 281px 130px">9</div>
  <div style="margin: 309px 130px">10</div>
  <div style="margin: 336px 130px">11</div>
  <div style="margin: 363px 130px">12</div>
  <div style="margin: 392px 130px">13</div>
  <div style="margin: 422px 130px">14</div>
  <img src="../images/capture.png"/>
</div>

<div class="callouts" markdown="block">

1. From the `Device` drop-down, select the device to trace.

1. From the `API` drop-down, select the graphics API you want to trace.

1. Using the `...` button, select the Android Activity or browse the application that you want to trace.

1. Add any command-line `Arguments` that are necessary for your program.

1. Select the `Working Directory` for your program, only valid for tracing on Windows/Linux machines.

1. Set the `Environment Variables` for tracing your program, only valid for tracing on Windows/Linux machines.

1. If you wish to automatically stop tracing after N frames, then use a non-zero number for `Stop After`.

1. If you wish to start tracing as soon as the application is launched, enable the `Trace From Beginning` option. If this option is **NOT** set, then in the tracing dialog, you must press `Start` to start the capture.
<span class="info">Tracing OpenGL ES calls at the middle of the execution of an Android Activity (not from the beginning of the application by disabling this option) is currently an experimental feature. </span>

1. `Disable Buffering` disables the bufferring of the capture data on the tracing device which will slow down the tracing process. But in case of a crash, more the most recent data will be provided.

1. If you would like to erase the package cache before taking the trace, enable the `Clear package cache` option.

1. `Hide Unknown Extensions` hides the Vulkan extensions not supported by GAPID to the application when tracing Vulkan calls. For GLES calls, it does not do anything. GAPID always hide unknown extensions when tracing OpenGL ES calls.

1. If tracing an OpenGL ES application you likely want to keep the `Disable pre-compiled shaders` option enabled. This option fakes no driver support for pre-compiled shaders for OpenGL ES, usually forcing the application to use `glShaderSource()`. GAPID is currently unable to replay captures that uses pre-compiled shaders when tracing for OpenGL ES. This option is invalid when tracing Vulkan Calls.

1. Select an output directory

1. Select an output file name.

</div>

Click `OK` to begin the trace.

## Known issues

<div class="issue" markdown="span">	
Please close any running instances of Android Studio before attempting to take a trace on Android devices. [#911](https://github.com/google/gapid/issues/911)	
</div>
