---
layout: default
title: How do I use GAPID to optimize my application?
permalink: /tutorials/optimize
---

Whilst GAPID does not yet support profiling, you can still use GAPID to optimize your application. Here are some tips on what to look for.

## Commands

Use the Commands pane to keep track of how many API calls you are making per frame.

### OpenGL ES
Keeping on top of your draw calls, your state changes and ensuring that the actual calls your engine is making are what you expect are critical to optimal performance.

#### Draw Calls

The number of draw calls (such as `glDrawElements` in OpenGL ES) can largely impact your performance. For example, on mobile VR applications the general consensus is that more than 50-100 draw calls per frame will limit your ability to hit a 60Hz refresh rate. A common reason for a large number of draw calls is that your engine is not taking into account batching or instancing capabilities, which can be confirmed if you observe large numbers of draw calls that render the same or similar geometry multiple times. Refer to the documentation of your particular engine to see why this might be the case. For VR applications you can also confirm whether you have multi-view (also called Single Pass Stereo) modes enabled, which will greatly reduce CPU overhead.

With respect to draw calls, most GPUs like to render their opaque geometry front-to-back to take into account internal optimizations such as Hi-Z. By stepping through each draw call in a frame you can confirm that your frame is being rendered in the order you expect.

#### State Changes

A common performance issue in titles is a large number of state changes between draw calls. Most meshes will share some state between other draw calls and you should only call the API if you know the state has changed. For example, if multiple objects use the same shader program, the program should be bound once for all draw calls that use that shader. This may go against the advice above with draw call order, and some experimentation is required to find the best solution.

Similarly, the application should not needlessly call APIs. Some applications will reset all OpenGL ES state at the start of each frame, even though a lot of the state does not change frame-to-frame. Use the GAPID command pane to identify these issues.

### Vulkan

In Vulkan, the number of individual draw calls and state changes does not necessarily correlate as closely to performance as in OpenGL ES. Things that are more likely to help your performance are the use of Render Passes and Subpasses. Reducing the number of resource transitions and pipeline barriers can also help improve performance. The topic of Vulkan optimization is both new, and complex, so a full discussion is out of the scope for this section.

## Resources

#### Textures

Incorrect or inappropriate texture formats can often cause significant performance problems, especially on mobile devices. Common pitfalls include uncompressed texture formats (on Android, seeing textures using `GL_RGBA` and `GL_UNSIGNED_BYTE` for Format generally indicates an uncompressed texture format, whereas ASTC formats are generally preferred) and a lack of mipmaps. Some engines will not compress or generate textures if the dimensions themselves are not a power of two or multiple of four, so confirm your resource settings in your engine against what GAPID reports to make sure your texture formats are set correctly. On the flipside, resources that have mipmaps that are too large may waste memory and quality when applications never read from the largest mipmap.

#### Geometry

Graphics processing hardware can easily be bottlenecked by the sheer number of vertices processed per frame. Using GAPID there are three options to see how dense your meshes are: in the Framebuffer pane, in the Commands pane or in the Geometry pane.

In the Framebuffer pane you can click the Wireframe icon to show the wireframe view. Generally this will give you a good idea of density. Whilst each platform has its own limits, in general, if you cannot see through your mesh, then you have too many vertices.

<img src="../images/wireframe.png" alt="Wireframe Framebuffer" width="670" height="480">

In the Commands pane you can see the number of vertices/indices in your draw call by inspecting the relevant parameter passed to your draw call.

And in the Geometry pane, when a draw call is selected you can see the Vertex/Index/Triangle count of that draw call below the 3D view.

<img src="../images/geometry.png" alt="Geometry View" width="669" height="480">

#### Shaders

Whilst a full guide on optimizing shaders is out of scope for this document, it's always worth taking a look at your shaders to make sure that they are as simple as you expect. For draw calls that affect large portions of your screen, this is especially important.

## State

Depending on what you are drawing, you should double check the API state in the State pane to see it is what you expect. For OpenGL ES it is often useful to use [debug markers](https://www.khronos.org/registry/OpenGL/extensions/EXT/EXT_debug_marker.txt) in your code to separate your rendering passes. One example might be confirming that depth writes are disabled for rendering back-to-front sorted geometry for your transparent pass, or that color writes are disabled for depth-only passes.
