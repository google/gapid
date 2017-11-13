---
layout: default
title: Tracing a Desktop application
sidebar: Desktop
permalink: /trace/desktop
parent: trace
---

GAPID supports tracing Vulkan calls made by applications on Windows and Linux.

## Dependencies and prerequisites

* A Vulkan application compiled with x86-64. 32-bit applications are not currently supported. 

## Taking a capture

Click the `Capture Trace...` text in the welcome screen, or click the `File` &rarr; `Capture Trace` toolbar item to open the trace dialog. The second tab will allow a Desktop trace.


<div class="callout-img">
  <div style="margin: 72px 171px">1</div>
  <div style="margin: 107px 240px">2</div>
  <div style="margin: 136px 283px">3</div>
  <div style="margin: 163px 190px">4</div>
  <div style="margin: 194px 190px">5</div>
  <div style="margin: 221px 295px">6</div>
  <div style="margin: 245px 324px">7</div>
  <div style="margin: 270px 281px">8</div>
  <img src="../images/capture-desktop.png"/>
</div>

<div class="callouts" markdown="block">

1. Currently, the only `API` that is supported for Desktop is Vulkan

1. `Browse`, for the Vulkan exectuable that you want to trace.

1. Add any command-line arguments that are necessary for your program.

1. Select the `Working Directory` for your program.

1. Select an output directory.

1. Select an output file name.

1. If you wish to automatically stop tracing after N frames, then use a non-zero number for `Stop After`.

1. If you wish to start tracing as soon as the application is launched, enable the `Trace From Beginning` option. If option is set, then in the tracing dialog, you must press `Start` to start the capture.

</div>

Click `OK` to begin the trace.

