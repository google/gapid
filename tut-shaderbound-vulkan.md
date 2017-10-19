---
layout: default
title: How do I see the currently bound shader in Vulkan?
sidebar: See bound shaders in Vulkan?
order: 21
permalink: /tutorials/seeboundshaders_vulkan
parent: tutorials
---

To see the currently bound shaders for a particular draw call you can use the State pane. 

## State Pane

First select a Vulkan draw call. These are the commands that are nested under a VkQueueSubmit call. The non-nested ones are only the location that the commands were recorded.

![alt text](../images/vulkan_commands.png "Selecting a vulkan draw")

Once you have a draw-call selected, in the state view, you can navigate to `LastDrawInfos-><ID of the Queue in VkQueueSubmit>->(Graphics|Compute)Pipeline->Stages-><Stage>->Module`. This will give you the VulkanHandle of the shader that is bound for a particular stage.

![alt text](../images/shaders_vulkan.png "Finding the bound program through the State Pane")

## Viewing vertex and fragment shaders from the bound Program

If you would like to [iterate on your shaders](../tutorials/iterateonshaders), then you can locate the bound shader, and modify it.

Navigate to the Shaders pane and the Shaders tab and find the shader from here.
