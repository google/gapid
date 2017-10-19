---
layout: default
title: Tracing an Android application
sidebar: Android
permalink: /trace/android
parent: trace
---

GAPID supports tracing all OpenGL ES and Vulkan calls made by an Android application. This works whether the application is pure Java, native or hybrid.

## Dependencies and prerequisites

* A device running Android Lollipop 5.0 (or more recent).
* Either a [debuggable](https://developer.android.com/guide/topics/manifest/application-element.html#debug) application, or a device running a 'rooted' user-debug build.
* Android SDK installed on the host machine.
* Android hardware device connected through USB.
* The device must have [USB debugging enabled](https://developer.android.com/studio/debug/dev-options.html) and the host machine must be authorized for debugging.

## Taking a capture

Click the `Capture Trace...` text in the welcome screen, or click the `File` &rarr; `Capture Trace` toolbar item to open the trace dialog.

<img src="../images/capture-gles.png" width="730px"/>

<div class="callouts" markdown="block">

1. From the `API` drop-down, select the graphics API you want to trace.

1. From the `Device` drop-down, select the Android device.

1. Using the `...` button, select the Android Activity you want to trace.

1. Select an output directory.

1. Select an output file name.

1. If you wish to automatically stop tracing after N frames, then use a non-zero number for `Stop After`.

1. If you wish to start tracing as soon as the application is launched, enable the `Trace From Beginning` option. If option is set, then in the tracing dialog, you must press `Start` to start the capture.
<span class="info">Tracing OpenGL ES calls currently requires recording form the very start of the application. If the part of the application you want to debug is takes significant time to reach from application startup, consider creating a separate Activity that launches straight into the part of the application you care about.</span>

1. If you would like to erase the package cache before taking the trace, enable the `Clear package cache` option.

1. If tracing an OpenGL ES application you likely want to keep the `Disable pre-compiled shaders` option enabled. This option fakes no driver support for pre-compiled shaders, usually forcing the application to use `glShaderSource()`. GAPID is currently unable to replay captures that uses pre-compiled shaders.

</div>

Click `OK` to begin the trace.

## Known issues

<div class="issue" markdown="span">
  Please close any running instances of Android Studio before attempting to take a trace. [#911](https://github.com/google/gapid/issues/911)
</div>
