---
layout: default
title: Commands Pane
permalink: /inspect/commands
---


The **Commands** pane displays the calls made by the application, grouped by frame and draw call or by user markers.

<figure style="display: inline-block;">
	<img src="../images/commands-pane/opengl.png" width="400px" />
	<figcaption>Viewing an OpenGL trace</figcaption>
</figure>

<figure style="display: inline-block;">
	<img src="../images/commands-pane/vulcan.png" width="400px" />
	<figcaption>Viewing a Vulcan trace</figcaption>
</figure>


## Operations

You can perform the following operations in this pane:

<table>
   <tr>
      <th style="width:10%"> Operation
      </th>
      <th>Description
      </th>
   </tr>
   <tr>
      <td>Show result
      </td>
      <td>Click on a command or group to update the other panes to reflect the state <strong>after</strong> the selected command / group.
      </td>
   </tr>
   <tr>
      <td>Expand or collapse the call hierarchy
      </td>
      <td>Click the gray triangle to the left of a grouping or double-click the grouping to expand or collapse it.
      </td>
   </tr>
   <tr>
      <td>Search
      </td>
      <td>
         Type a string in the search bar at the top of the pane, and then press Return (see image above). To find the next occurrence, make sure the bar is selected and press Return again.
         <p>
            Select the <strong>Regex</strong> box to use a regular expression search pattern. For example, <strong>glClear.*</strong> will match both commands <code>glClear()</code> and <code>glClearColor()</code>. You can also search for command parameter values such as <strong>buffer: 2</strong>, which is used in <code>glBindBuffer()</code>.
        </p>
      </td>
   </tr>
   <tr>
      <td>Edit
      </td>
      <td>Right-click a command and select <strong>Edit</strong>. In the <strong>Edit</strong> dialog, change one or more values and click <strong>OK</strong>.
      </td>
   </tr>
   <tr>
      <td>View state or memory information
      </td>
      <td>Click a command argument that refers to a state parameter, such as a texture ID. The <a href="/inspect/state">State</a> pane opens to provide additional information. Click a memory address or pointer to open the <a href="/inspect/memory">Memory</a> pane.
      </td>
   </tr>
   <tr>
      <td>Copy commands
      </td>
      <td>Select items in the pane and press Control+C (or Command+C) to copy commands with their argument values. You can paste this information into a text file.
      </td>
   </tr>
   <tr>
      <td>Magnify thumbnail
      </td>
      <td>
         The thumbnail appears to the left of a call (see image below). Hover the cursor over the thumbnail to enlarge it.  <br>
         <img src="../images/commands-pane/magnify-thumbnail.png" width="300px" alt="alt_text" title="image_tooltip" />
      </td>
   </tr>
</table>

## Debug Markers

Depending on your app, the **Commands** pane can contain a very long list of commands within one frame. For better navigation and readability, you can define debug markers that group calls together under a heading in the tree. This could include a grouping named **Setup** or **Render World**.

If debug markers are enabled, you need to click the **Commands** pane to reveal a link to this information.

OpenGL ES has the following APIs to group commands:

Extension / Version                  | Push                     | Pop
------------------------------------ | ------------------------ | -----------------------
[KHR_debug][KHR_debug]               | `glPushDebugGroupKHR()`  | `glPopDebugGroupKHR()`
[EXT_debug_marker][EXT_debug_marker] | `glPushGroupMarkerEXT()` | `glPopGroupMarkerEXT()`
[OpenGL ES 3.2][glPopDebugGroup]     | `glPushDebugGroup()`     | `glPopDebugGroup()`

Vulkan has the following APIs to group commands:

Extension / Version                        | Push                          | Pop
------------------------------------------ | ----------------------------- | -----------------------
[VK_EXT_debug_marker][VK_EXT_debug_marker] | `vkCmdDebugMarkerBeginEXT()`  | `vkCmdDebugMarkerEndEXT()`


[KHR_debug]:        https://www.khronos.org/registry/gles/extensions/KHR/KHR_debug.txt
[EXT_debug_marker]: https://www.khronos.org/registry/gles/extensions/EXT/EXT_debug_marker.txt
[glPopDebugGroup]:  https://www.khronos.org/opengles/sdk/docs/man32/html/glPopDebugGroup.xhtml
[VK_EXT_debug_marker]: https://github.com/KhronosGroup/Vulkan-Docs/blob/1.0/doc/specs/vulkan/chapters/VK_EXT_debug_marker.txt
