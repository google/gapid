---
layout: default
title: How do I iterate on my shaders in GAPID?
sidebar: Iterate on shaders?
order: 30
permalink: /tutorials/iterateonshaders
parent: tutorials
---

Changing a shader within GAPID will change the result for all draw calls, allowing you to iterate on the look and feel of your application within GAPID. In the future, this will also allow you to see how editing shaders affects performance.

1. First, find the [appropriate shader](../tutorials/seeboundshaders) in the Shader pane.
2. Edit the text of your shader.
3. Click "Push Changes" and wait for the replay to finish processing the results.

<img src="../images/gles/shaders.png" alt="Editing a shader within GAPID" width="426" height="397">

## Notes for Vulkan
The shaders will be presented as the disassembled SPIR-V module that was loaded. This can be edited in place, or a new disassembled SPIR-V module can be inserted in it's place. To generate the disassembly [SPIRV-Tools](SPIRV-Tools) can be used to disassemble any SPIR-V module.

## Tips

If your shader fails to compile, your replay may not operate correctly and you may get an empty framebuffer. To see if you have introduced any compile errors, refer to the [Report pane](..inspect/report) and fix any issues that show up.

[SPIRV-Tools]: https://github.com/khronosgroup/spirv-tools
