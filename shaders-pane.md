---
layout: default
title: Shaders Pane
permalink: /inspect/shaders
---


<div class="tab" id="OpenGL ES" markdown="1">
<img class="display" src="../images/opengles.svg" alt="OpenGL ES" height="50"/>

The shaders pane displays all the shader resources created up to and including the selected command. You can see both individual shaders created as well as programs that are bound to multiple shaders.

## Shaders tab

<img src="../images/shaders-pane-shaders-tab.png" width="558px"/>

In this tab you can see the shader resources created using `glCreateShader`, as well as make changes to them. You can freely edit the shader text field and click `Push Changes` to make changes. After pushing changes, take a look at the Report pane to see if there are any errors, as your changes won't go into effect unless the shader compiles cleanly.

## Programs tab

<img src="../images/shaders-pane-programs-tab.png" width="558px"/>

You can see the list of Programs created using `glCreateProgram` as well as shaders attached using calls to `glAttachShader`.

</div>

<div class="tab" id="Vulkan" markdown="1">
<img class="display" src="../images/vulkan.svg" alt="Vulkan" height="50">

The shaders pane displays all the shader resources created up to and including the selected command.

<img src="../images/vulkan_shaders_pane.png"  width="793px"/>

All of the shaders created with a call to `vkCreateShaderModule` show up here. What is displayed is the `SPIR-V` disassembly of the shader itself. You can freely modify the shader assembly, or paste new assembly into the shader. Once you have made the modifications, you can click `Push Changes` to see the effects of the change. The report view will notify
you of any errors that occur.
</div>
