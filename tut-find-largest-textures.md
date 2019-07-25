---
layout: default
title: How do I find the largest textures?
permalink: /tutorials/findlargesttextures
---

Finding the largest textures loaded by the driver is useful to reduce the memory footprint and load times of your application. For more complex projects, applications may load texture resources that aren't even used, so looking through the list of resources loaded should be done regularly to ensure optimal memory usage.

To find the largest textures, navigate to the [Textures Pane](/inspect/textures).

<img src="../images/textures.png" alt="Texture Pane" width="670" height="480">

Click on the headers to sort by each property. In general, you are looking for the textures with the largest dimensions and format - sorting by `Width` and `Height` is usually a good first step. Any textures with a format of `GL_RGBA` will be larger than textures using `GL_RGB`, and specifically look for RGB or RGBA textures using `GL_FLOAT` as a type.

In the future, finding the largest texture by byte size will be easier when [Issue 1118](https://github.com/google/gapid/issues/1118) has been resolved, as you will be able to simply sort by the byte size column.

