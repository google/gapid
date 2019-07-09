---
title: About
sidebar: About
layout: default
permalink: /about/
order: 10
---

GAPID is a developer tool for recording and inspecting calls made by an application to the graphics driver.

It is open-source: [https://github.com/google/gapid](https://github.com/google/gapid)

<div style="text-align: center;">
    <img src="../images/hero.gif" alt="GAPID image" width="540" height="337">
    <figcaption>Using GAPID to step through each individual draw call of a frame</figcaption>
</div>

Once a capture of a target application has been made, GAPID lets you disconnect from the target and inspect all the graphics commands made by the application.

GAPID is able to replay the command stream, letting you visualize the frame composition by stepping through each command and inspecting the driver state at any point in the stream. Replay also supports modifications, allowing you to adjust command parameters and shader source to instantly see what effect this would have on the frame.

GAPID can also visualize the textures, shaders and draw call geometry used by the application.

## API support

|                              | Android | Windows | macOS  | Linux | Stadia
| ---------------------------- | ------- | ------- |------- | ----- | ------
| OpenGL ES - Trace            |   ✓     |         |        |       |
| OpenGL ES - Replay           |   ✓     |   ✓     |   ✓    |   ✓   |
| Vulkan - Trace               |   ✓     |   ✓     |        |   ✓   |   ✓
| Vulkan - Replay <sup>*</sup> |   ✓     |   ✓     |        |   ✓   |   ✓

<sup>*</sup> Vulkan replay currently needs to be performed on the same device used to trace.

While GAPID is primarily targeted for games developers, it can also help developers to inspect low-level 2D graphics calls made by the Android graphics framework.

GAPID is under active development and has some [known issues](https://github.com/google/gapid/issues). Your feedback is appreciated!
