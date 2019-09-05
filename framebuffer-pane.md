---
layout: default
title: Framebuffer Pane
permalink: /inspect/framebuffer
---

The **Framebuffer** pane shows the contents of the currently-bound framebuffer. Depending on the item you select in the **Commands** pane, the 
**Framebuffer** pane can show onscreen or offscreen framebuffers.

<img src="../images/framebuffer-pane/framebuffer-pane.png" width="600px" />

When you select a command in the **Commands** pane, the **Framebuffer** pane displays the contents of the framebuffer after that call finishes. If you select a command group, it displays the framebuffer that best represents the group. Typically, this is the framebuffer after the last call in the group finishes.

Start by selecting the first call within a frame, and then click each successive call to watch the framebuffer components draw one-by-one until the end of the frame. These framebuffer displays, for both onscreen and offscreen graphics, help you to locate the source of any rendering errors.

Move the cursor over the image to display a zoomed-in preview of the surrounding pixels in the bottom-left hand corner of the view as in the image above. The pane will also show the image width and height as well as the x and y coordinates, normalized image coordinates (U and V values), and RBGA hex value for that point on the image.

If the framebuffer contains undefined data, you’ll see diagonal bright green lines, as shown below.

<img src="../images/framebuffer-pane/undefined-data.png" width="600px" />

## Operations

You can perform operations on the framebuffer image using the following buttons:

<table>
  <tbody>
    <tr>
      <th style="width:5%">Button</th>
      <th>Description</th>
      <th>Example Result</th>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/color_buffer0%402x.png" alt="Choose framebuffer attachment to display"/>
      </td>
      <td>
        Selects the framebuffer attachment to display. You can select one of four color attachments or a depth attachment.
      </td>
      <td>
        <img src="../images/framebuffer-pane/attachments.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/wireframe_none%402x.png" alt="Render shaded geometry"/>
      </td>
      <td>Renders the shaded geometry of the image.</td>
      <td>
        <img src="../images/framebuffer-pane/shaded-geometry.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/wireframe_overlay%402x.png" alt="Render shaded geometry and overlay wireframe of last draw call"/>
      </td>
      <td>Renders the shaded geometry of the image and overlays the wireframe of the image.</td>
      <td>
        <img src="../images/framebuffer-pane/shaded-geometry-wireframe.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/wireframe_all%402x.png" alt="Render wireframe geometry"/>
      </td>
      <td>Shows the wireframe of the image.</td>
      <td>
        <img src="../images/framebuffer-pane/wireframe.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/overdraw%402x.png" alt="Render overdraw"/>
      </td>
      <td>Renders the overdraw of the image. This is not supported in OpenGL traces.
      </td>
      <td>
        <img src="../images/framebuffer-pane/overdraw.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_fit%402x.png" alt="Zoom to Fit"/>
      </td>
      <td>
        Adjusts the image to fit completely within the pane. You can also right-click the image to adjust the zoom to fit the image.
      </td>
      <td>
        <img src="../images/framebuffer-pane/zoom-to-fit.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_actual%402x.png" alt="Zoom to Actual Size"/>
      </td>
      <td>Displays the image at no scale, where one device pixel is equivalent to one screen pixel.</td>
      <td>
        <img src="../images/framebuffer-pane/original-size.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_in%402x.png" alt="Zoom In"/>
      </td>
      <td>Zooms in on the image. You can also use your mouse wheel, or two-finger swipes on a touchpad, to zoom in and out. You can drag the image with your cursor.</td>
      <td>
        <img src="../images/framebuffer-pane/zoom-in.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_out%402x.png" alt="Zoom Out"/>
      </td>
      <td>Zooms out on the image. You can also use your mouse wheel, or two-finger swipes on a touchpad, to zoom in and out.</td>
      <td>
        <img src="../images/framebuffer-pane/zoom-out.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/histogram%402x.png" alt="Color Histogram"/>
      </td>
      <td>Displays the color histogram for the image. You can select the control handles on either side to limit the color values displayed.
      </td>
      <td>
        <img src="../images/framebuffer-pane/histogram.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/color_channels_15%402x.png" alt="Color Channels"/>
      </td>
      <td>Select the color channels to render. The options are <b>Red</b>, <b>Green</b>, <b>Blue</b>, and <b>Alpha</b> (transparency).</td>
      <td>
        <img src="../images/framebuffer-pane/color-channel.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/transparency%402x.png" alt="Background"/>
      </td>
      <td>Select a checkerboard pattern or a solid color for the image background.</td>
      <td>
        <img src="../images/framebuffer-pane/image-background.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/flip_vertically%402x.png" alt="Flip Vertically"/>
      </td>
      <td>Flips the image vertically.</td>
      <td>
        <img src="../images/framebuffer-pane/flip-vertical.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/save%402x.png" alt="Save To File"/>
      </td>
      <td>Saves the image to a file.</td>
      <td>
      </td> 
    </tr>
  </tbody>
</table>