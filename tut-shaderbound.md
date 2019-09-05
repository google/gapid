---
layout: default
title: How do I see the currently bound shader?
permalink: /tutorials/seeboundshaders
---

<h4>Select the graphics API used:</h4>

<div class="tab" id="OpenGL ES" markdown="1">
<img class="display" src="../images/opengles.svg" alt="OpenGL ES" height="50"/>

To see the currently bound shaders for a particular draw call, you can use either the Command pane or the State pane. Using the Command pane is generally faster, unless the application batches multiple draw calls with the same shader program together. In this case, you might have to search for the draw call.

## Command pane

In the Command pane, navigate to the draw call that you want to investigate. Look above the draw call to find the preceding `glUseProgram()` call. The program parameter for this function is the identifier for the shader program being bound.

Navigate to the Shaders pane and then go to the Programs tab. Select the relevant program from the list. For example, if your application calls `glUseProgram(program:13)`, then navigate to `Program<13>` in the Programs list.

<img src="../images/gles/commands_find_program.png" alt="Finding a bound program through the Command Pane" width="403" height="405">

In cases where your application does not bind a shader program close to a draw call, use the State Pane to find the currently bound program.

## State pane

To find the currently bound program in the State pane, navigate to Bound &rarr; Program. The ID field identifies the currently bound shader program.

Navigate to the Shaders pane and the Programs tab to find the currently bound program.

<img src="../images/gles/get_shader_id.png" alt="Finding the bound program through the State Pane" width="426" height="397">

## Viewing vertex and fragment shaders from the bound program

If you want to [iterate on your shaders](../tutorials/iterateonshaders), locate a specific shader outside of the context of the bound program. In the State pane, find the currently bound program and then expand the Shaders node to find the IDs of the individual shaders.

Navigate to the Shaders pane and go to the Shaders tab to find the shader.

</div>

<div class="tab" id="Vulkan" markdown="1">
<img class="display" src="../images/vulkan.svg" alt="Vulkan" height="50">

To see the currently bound shaders for a particular draw call you can use the State pane.

## State Pane

First select a Vulkan draw call. These are the commands that are nested under a VkQueueSubmit call. Non-nested items indicate only the location where commands were recorded.

<img src="../images/vulkan_commands.png" alt="Selecting a Vulkan draw" width="350" height="549">

Once you have a draw call selected in the State view, navigate to `LastDrawInfos-><ID of the Queue in VkQueueSubmit>->(Graphics|Compute)Pipeline->Stages-><Stage>->Module`. This gives you the VulkanHandle of the shader that is bound for a particular stage.

<img src="../images/shaders_vulkan.png" alt= "Finding the bound program through the State Pane" width="340" height="549">

## Viewing vertex and fragment shaders from the bound program

If you want to [iterate on your shaders](../tutorials/iterateonshaders), locate the bound shader and modify it.

Navigate to the Shaders pane and go to the Shaders tab to find the shader.

</div>
