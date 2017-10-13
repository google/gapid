---
layout: default
title: How do I iterate on my shaders in GAPID?
sidebar: Iterate on shaders?
permalink: /tutorials/iterateonshaders
parent: tutorials
---

Changing a shader within GAPID will change the result for all draw calls, allowing you to iterate on the look and feel of your application within GAPID. In the future, this will also allow you to see how editing shaders affects performance.

1. First, find the [appropriate shader](../tut-shaderbound.md) in the Shader pane.
2. Edit the text of your shader.
3. Click "Push Changes" and wait for the replay to show up.

![alt text](../images/shader.png "Editing a shader within GAPID")

## Tips

If your shader fails to compile, your replay may not operate correctly and you may get a blank framebuffer. To see if you have introduced any compile errors, refer to the [Report pane](../report.md) pane and fix any issues that show up.
