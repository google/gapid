---
layout: default
title: Textures Pane
permalink: /inspect/textures
---

<img src="../images/textures-pane.png" width="558px"/>

The textures pane displays all the texture resources created up to and including the selected command.

If the texture has a mip-map chain, then you can change the displayed mip-map level with the slider at the bottom. By default level 0 (the highest resolution) level will be displayed.

Placing your mouse cursor over the texture will display a zoomed in preview of the surrounding pixels in the bottom-left hand corner of the view.

To the left of the texture view is a toolbar with the following buttons:

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
        Click the button to adjust the graphic to fit completely within the <b>Textures</b> pane.
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
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/color_channels_00%402x.png" alt="Color Channels"/>
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
      <td>Flips the texture vertically.</td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/jump%402x.png" alt="Accesses / Modifications"/>
      </td>
      <td>Select this button to view a list of all calls that updated the texture to this point. Select a call to view the texture after the call completes. The selected frame thumbnail and the Commands pane update accordingly.</td>
    </tr>
  </tbody>
</table>
