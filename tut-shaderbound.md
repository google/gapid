---
layout: default
title: How do I see currently bound shader?
sidebar: See bound shaders?
permalink: /tutorials/seeboundshaders
parent: tutorials
---

To see the currently bound shaders for a particular draw call, you can use either the Command pane or the State pane. Using the Command pane is generally easier, unless the application batches multiple draw calls with the same shader program together, which may require some searching.

## Command Pane

In the Command pane, navigate to the draw call you would like to investigate. For GLES, look upwards to find the preceding glUseProgram() call, and the program parameter for this function is the identifier for the Shader Program being bound. Navigate to the Shaders pane and then the Programs tab and select the relevant program from the list. For example, if your application calls glUseProgram(program:22) - then navigate to Program<22> in the Programs list.

In the cases where your application does not bind a shader program close to a draw call, use the following method.

## State Pane

To find the currently bound program in the State pane, navigate to Bound -> Program and the ID field identifies the currently bound shader program.

As above, you can then navigate to the Shaders pane and the Programs tab to find the currently bound program.

## Viewing vertex and fragment shaders from the bound Program

If you would like to [iterate on your shaders](../tut-shaderiterating.md), then you want to see the specific shader itself, outside of the context of the bound program. In the State pane, find the currently bound program and then expand the Shaders node to find the IDs of the individual shaders.

Navigate to the Shaders pane and the Shaders tab and find the shader from here.
