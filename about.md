---
title: About
sidebar: About
layout: default
permalink: /about/
---

GAPID is a developer tool for recording and inspecting calls made by an application to the graphics driver.

Once a capture of the target application has been made, GAPID lets you disconnect from the target and inspect all the graphics commands made by the application.

GAPID is able to replay the command stream, letting you visualize the frame composition draw-call by draw-call, and inspect the driver state at any point in the stream. Replay also supports modifications, allowing you to tweak command parameters and shader source to instantly see what effect this would have on the frame.

GAPID can also visualize the textures, shaders and draw call geometry used by the application.

GAPID supports the OpenGL ES and Vulkan graphics APIs:
 * GAPID can trace OpenGL ES and Vulkan applications on Android.
 * GAPID can trace Vulkan applications on Windows and Linux.
 * GAPID can replay Vulkan captures on the same device used to trace.
 * GAPID can replay OpenGL ES captures on Windows, MacOS and Linux.

While GAPID is primarily targetted for games developers, it can also developers to inspect the low-level 2D graphics calls being made by the Android graphics framework.

GAPID is under active development and has [known issues](https://github.com/google/gapid/issues). That said, we believe GAPID is usable and would love for feedback.
