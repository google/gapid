---
layout: default
title: Commands Pane
permalink: /inspect/commands
---

<img src="../images/commands-pane.png" width="498px" height="462px"/>

The commands pane displays the calls made by the application, grouped by frame and draw call or by user markers.

Clicking on a command or group will update other panes to reflect the state **after** the selected command / group.

You can perform the following operations in the Commands pane:

* To expand or collapse an item in the call hierarchy, click <img alt="Collapsed icon" src="../images/tree-expand.png" width="16px"/> or double-click the item.
* To search for a command, type a string in the search field <img alt="Search icon" src="../images/search.png" width="114px"/> at the top of the pane, and then press **Return**. If you want to use a regular expression search pattern, select <img alt="Search icon" src="../images/regex.png" width="63px" alt="Regex"/>. To find the next occurrence, select the search field and press **Return** again.
* To change argument values, right-click a function and choose **Edit**. In the **Edit** dialog, change one or more values in the fields, and then click **OK**.
* If you click an argument that refers to a state parameter, such as a texture ID, the [State](state-pane) pane opens and shows you more information. If you click a memory address or pointer, the [Memory](memory-pane) pane opens and shows you that location in memory.
* To magnify a call thumbnail, hover your cursor over it. The thumbnail appears to the left of a call.
* To copy values from the display, select the items to copy and then press Control+C or Command+C. You can then paste the information into a text file.

## Debug Markers

Depending on your app, the **Command** pane can contain a very long list of commands within one frame.
For better navigation and readability, you can define debug markers that group calls together under a heading in the tree.

OpenGL ES has the following APIs to group commands:

Extension / Version                  | Push                     | Pop
------------------------------------ | ------------------------ | -----------------------
[KHR_debug][KHR_debug]               | `glPushDebugGroupKHR()`  | `glPopDebugGroupKHR()`
[EXT_debug_marker][EXT_debug_marker] | `glPushGroupMarkerEXT()` | `glPopGroupMarkerEXT()`
[OpenGL ES 3.2][glPopDebugGroup]     | `glPushDebugGroup()`     | `glPopDebugGroup()`

Vulkan has the following APIs to group commands:

Extension / Version                        | Push                          | Pop
------------------------------------------ | ----------------------------- | -----------------------
[VK_EXT_debug_marker][VK_EXT_debug_marker] | `vkCmdDebugMarkerBeginEXT()`  | `vkCmdDebugMarkerEndEXT()``


[KHR_debug]:        https://www.khronos.org/registry/gles/extensions/KHR/KHR_debug.txt
[EXT_debug_marker]: https://www.khronos.org/registry/gles/extensions/EXT/EXT_debug_marker.txt
[glPopDebugGroup]:  https://www.khronos.org/opengles/sdk/docs/man32/html/glPopDebugGroup.xhtml
[VK_EXT_debug_marker]: https://github.com/KhronosGroup/Vulkan-Docs/blob/1.0/doc/specs/vulkan/chapters/VK_EXT_debug_marker.txt
