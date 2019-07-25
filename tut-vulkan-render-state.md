---
layout: default
title: Check Vulkan render state
permalink: /tutorials/renderstate
---

The state of rendering of Vulkan applications is reflected in two parts in
GAPID:
 - [Command pane](../inspect/commands): Submitted commands grouped by renderpass and subpasses.
 - [State pane](../inspect/state): Bound resources, framebuffers, renderpasses, etc.

## Command grouping
Under a expanded `vkQueueSubmit` command, the submitted command buffer commands
are grouped in the [command pane](../inspect/commands) according to submission
batches and command buffers. An expanded `vkQueueSubmit` with drawing commands
in it will looks like the image below.

<div class="callout-img">
    <div style="margin: 246px 39px">1</div>
    <div style="margin: 474px 39px">2</div>
    <img src="../images/command-pane-vulkan-renderpass-grouping.png" alt="Vulkan commands">
</div>

<div class="callouts" markdown="block">
1. Renderpass groups: Submitted commands between `vkCmdBeginRenderPass` and
   `vkCmdEndRenderPass` (inclusive) are grouped in a renderpass group. The
   group will be named with the renderpass name if the renderpass has been
   [assigned a name][vkDebugMarkerSetObjectNameEXT], the otherwise the value of
   the `VkRenderPass` will used as the name, as shown in the image above.

1. Subpass groups: Submitted commands divided by `vkCmdNextSubpass` will be
   grouped into corresponding subpass groups. Subpass groups will be skipped if
   the renderpass contains only one subpass, otherwise the subpass groups will
   be added under the renderpass group, named with the subpass number. Each
   subpass group ends with either `vkCmdNextSubpass` or `vkCmdEndRenderPass`.
</div>

## Drawing information
To check the render state after a specific submitted command, click the command
in question in command pane, the render state can be examed in the following
items in the [state pane](../inspect/state).

### Last bound queue (currently bound queue)
The `LastBoundQueue` node contains the information of the queue used for the
`vkQueueSubmit`, which submits the command in question. The `VulkanHandle` will
be used to find the drawing information of the current render state in
`LastDrawInfos`.

<div class="callout-img">
    <div style="margin: 89px 282px">1</div>
    <div style="margin: 197px 282px">2</div>
    <img src="../images/state-pane-vulkan-last-bound-queue.png"/>
</div>

<div class="callouts" markdown="block">

1. The `VulkanHandle` shows the value of the **last** used `VkQueue`, which is
   actually the currently bound queue for the submitted command in question.

1. The information of the current render state is stored in `LastDrawInfos`,
   indexed by `VkQueue` value.

</div>

### Last draw infos (current render state info)
`LastDrawInfos` contains the information of the **last** drawing for each
`VkQueue`, includes Framebuffer info, renderpass info, Bound descriptor sets,
Bound vertex/index buffers, Graphics pipeline and drawing parameters.

#### Bound framebuffer
<div class="callout-img">
    <div style="margin: 44px 352px">1</div>
    <div style="margin: 108px 352px">2</div>
    <div style="margin: 256px 352px">3</div>
    <div style="margin: 442px 352px">4</div>
    <img src="../images/state-pane-vulkan-draw-info-framebuffer.png"/>
</div>

<div class="callouts" markdown="block">

1. Framebuffer node shows the info of the currently bound framebuffer. This
   node gets updated after each `vkCmdBeginRenderPass` executes on the same
   queue.

1. Renderpass node shows the info of the renderpass used to create the
   framebuffer. Note that this is not the renderpass currently bound for
   drawing.

1. ImageAttachments node lists all the image attachments (`VkImageViews`) bound
   to the framebuffer. Each item of the list shows the info of the image view.

1. Image node shows the info of the image bound to the image view.

</div>

#### Bound renderpass
<div class="callout-img">
    <div style="margin: 171px 352px">1</div>
    <div style="margin: 235px 352px">2</div>
    <div style="margin: 547px 352px">3</div>
    <div style="margin: 778px 352px">4</div>
<img src="../images/state-pane-vulkan-draw-info-renderpass.png"/>
</div>

<div class="callouts" markdown="block">

1. Renderpass node shows the info of the renderpass currently used for
   rendering.  It gets updated after each `VkCmdBeginRenderPass` executes on
   the same queue.

1. AttachmentDescriptions node lists all the `VkAttachmentDescription` of the
   current renderpass in use.

1. SubpassDescriptions node lists the `VkSubpassDescription` for each subpass.

1. SubpassDependencies node lists the `VkSubpassDependency` for each subpass.

</div>

#### Bound descriptor sets
<div class="callout-img">
    <div style="margin: 65px 352px">1</div>
    <div style="margin: 166px 352px">2</div>
    <div style="margin: 278px 352px">3</div>
    <div style="margin: 590px 352px">4</div>
<img src="../images/state-pane-vulkan-draw-info-descriptorsets.png"/>
</div>

<div class="callouts" markdown="block">

1. DescriptorSets node lists all the currently bound descriptor sets. The list
   of bounded descriptor sets reflect the state after the last
   `vkCmdBindDescriptorSets` being rolled out on the same queue, and the original
   descriptor set info will be overwritten or new info will be added according
   to the parameters of the last executed `vkCmdBindDescriptorSets`.

1. Bindings node lists all the currently bound descriptor bindings in the
   descriptor set.

1. Each descriptor binding also lists its bound descriptors.

1. Layout node shows the info of the `VkDescriptorSetLayout` used to allocate
   the descriptor set.

</div>

### Bound graphics pipeline
<div class="callout-img">
    <div style="margin: 92px 258px">1</div>
<img src="../images/state-pane-vulkan-draw-info-gfx-pipeline.png"/>
</div>

<div class="callouts" markdown="block">

1. GraphicsPipeline node contains the info about the **last** bound graphics
   pipeline. This node gets updated after each `VkCmdBindPipeline` executes on
   the current queue.

</div>

#### Bound buffers
<div class="callout-img">
    <div style="margin: 110px 297px">1</div>
    <div style="margin: 380px 297px">2</div>
<img src="../images/state-pane-vulkan-draw-info-bound-buffers.png"/>
</div>

<div class="callouts" markdown="block">

1. BoundVertexBuffers node lists all the bound vertex buffers. For each bound
   vertex buffer, it shows the info of the backing buffer. The list gets
   updated accordingly after each `vkCmdBindVertexBuffers` executes on the
   same queue.

1. BoundIndexBuffer node shows the last bound index buffer, including the index
   type and the info of the backing buffer.

</div>

#### Draw command parameters
<div class="callout-img">
    <div style="margin: 149px 227px">1</div>
<img src="../images/state-pane-vulkan-draw-info-draw-params.png"/>
</div>

<div class="callouts" markdown="block">

1. CommandParameters node contains the parameters to `vkCmdDraw`,
   `vkCmdDrawIndexed`, `vkCmdDrawIndirect` and `vkCmdDrawIndirectIndexed`. For
   each type of drawining command, there is a corresponding sub-node to
   contains the parameter values. As these four types of drawining commands
   cannot be used at the same time, only one of the four sub-nodes can be
   populated at a time. The content of `CommandParameters` gets updated after
   any one of the four drawining commands being executed on the same queue.

</div>

[vkDebugMarkerSetObjectNameEXT]:https://www.khronos.org/registry/vulkan/specs/1.0-extensions/html/vkspec.html#vkDebugMarkerSetObjectNameEXT
