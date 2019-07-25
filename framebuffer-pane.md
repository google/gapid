---
layout: default
title: Framebuffer Pane
permalink: /inspect/framebuffer
---

The **Framebuffer** pane shows the contents of the currently bound framebuffer. Depending on the item you select in the **Commands** pane, it can show onscreen or offscreen framebuffers.

When you select a command in the **Commands** pane, the **Framebuffer** pane displays the contents of the framebuffer after that call finishes. If you select a group, it displays the framebuffer after the last call in that group finishes.

In essence, you can start by selecting the first call within a frame, and then click each successive call to watch the framebuffer components draw one-by-one until the end of the frame. These framebuffer displays, for both onscreen and offscreen graphics, help you to locate the source of any rendering errors.

<img src="../images/framebuffer-pane.png" width="696px" height="425px"/>

If the framebuffer contains undefined data, you'll see diagonal bright green lines, as shown in figure 5.

<img src="../images/framebuffer-undefined.png" width="696px" height="425px"/>

The following table describes the operations you can perform with the toolbar buttons.

<table>
  <tbody>
    <tr>
      <th width="20%">Button</th>
      <th>Description</th>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_fit%402x.png" alt="Zoom to Fit"/>
      </td>
      <td>
        Click the button to adjust the graphic to fit completely within the <b>Framebuffer</b> pane.
        <br/>Right-clicking the image is another way to <b>Zoom to Fit</b>.
      </td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_actual%402x.png" alt="Zoom to Actual Size"/>
      </td>
      <td>Click the button to show the image at no scale, where one device pixel is equivalent to one screen pixel.</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_in%402x.png" alt="Zoom In"/>
      </td>
      <td>Click the button to zoom in. You can also use your mouse wheel, or two-finger swipes on a touchpad, to scroll in and out. You can drag the image with your cursor.</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_out%402x.png" alt="Zoom Out"/>
      </td>
      <td>Click the button to zoom out. You can also use your mouse wheel, or two-finger swipes on a touchpad, to scroll in and out.</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/color_channels_00@2x.png" alt="Color Channels"/>
      </td>
      <td>Click the button and then select the color channels to render or deselect color channels so they aren't rendered. The options are <b>Red</b>, <b>Green</b>, <b>Blue</b>, and <b>Alpha</b> (transparency).</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/transparency%402x.png" alt="Background"/>
      </td>
      <td>Select the button to display a checkerboard background or solid color.</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/flip_vertically%402x.png" alt="Flip Vertically"/>
      </td>
      <td>Flips the image vertically.</td>
    </tr>
  </tbody>
</table>
