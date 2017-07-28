---
layout: default
title: Inspecting a trace
sidebar: Inspecting a trace
order: 20
permalink: /inspecting/
group: inspect
---

Once you've finished taking a trace, the capture will automatically open. You can also open previously created `.gfxtrace` files using the `File` &rarr; `Open` toolbar item.

Upon opening a capture, you will be presented with the following window:

<img src="../images/main-view.png" width="933px"/>

<div class="callouts" markdown="block">

1. The top of the view contains a rendering [context](https://www.opengl.org/wiki/OpenGL_Context) filter. By default all contexts are shown. By selecting a context the other panes will be filtered to just this selected context.

1. The film-strip view displays all the frames rendered in chronological order. Clicking on a frame will select that frame group.

1. On the left is the `Commands` pane. This is a hierarchical view of all the commands recorded. Placing your cursor over the thumbnails of groups will show a larger preview image.

1. On the right is the `Framebuffer` pane. This displays the contents of the currently bound framebuffer up to and including the selected command.

1. Select other tabs to explore the graphical objects, state, and memory values associated with the frame.
