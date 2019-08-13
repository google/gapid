---
layout: default
title: Textures Pane
permalink: /inspect/textures
---

The **Textures** pane displays all the texture resources created up to and including the selected command.

<img src="../images/textures-pane/textures-pane.png" width="700px"/>

Select a texture resource from the list near the top of the tab to display a rendering of the texture below.  Select the **Show deleted textures** checkbox to show textures in the UI even if they have been deleted. Select the **Include all contexts** checkbox to includes all textures from all contexts regardless of what is selected in the **Context** drop-down list.

If the texture has a mip-map chain, you can change the displayed mip-map level with the slider at the bottom (not pictured). By default, the highest resolution level, level 0, will be displayed.

Move the cursor over the texture image to display a zoomed-in preview of the surrounding pixels in the bottom-left hand corner of the view as in the image above. The pane will also show the texture width and height as well as the x and y coordinates, normalized texture coordinates (U and V values), and RBGA hex value for that point on the image.

## Operations

You can perform operations on the image using the following buttons:

<table>
  <tbody>
    <tr>
      <th style="width:5%">Button</th>
      <th>Description</th>
      <th>Example Result</th>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_fit%402x.png" alt="Zoom to Fit"/>
      </td>
      <td>
        Adjusts the image to fit completely within the pane. You can also right-click the image to adjust the zoom to fit the image.
      </td>
      <td>
        <img src="../images/textures-pane/zoom-to-fit.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_actual%402x.png" alt="Zoom to Actual Size"/>
      </td>
      <td>Displays the image at no scale, where one device pixel is equivalent to one screen pixel.</td>
      <td>
        <img src="../images/textures-pane/original-size.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_in%402x.png" alt="Zoom In"/>
      </td>
      <td>Zooms in on the image. You can also use your mouse wheel, or two-finger swipes on a touchpad, to zoom in and out. You can drag the image with your cursor.</td>
      <td>
        <img src="../images/textures-pane/zoom-in.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zoom_out%402x.png" alt="Zoom Out"/>
      </td>
      <td>Zooms out on the image. You can also use your mouse wheel, or two-finger swipes on a touchpad, to zoom in and out.</td>
      <td>
        <img src="../images/textures-pane/zoom-out.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/histogram%402x.png" alt="Color Histogram"/>
      </td>
      <td>Displays the color histogram for the image. You can select the control handles on either side to limit the color values displayed.
      </td>
      <td>
        <img src="../images/textures-pane/histogram.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/color_channels_00%402x.png" alt="Color Channels"/>
      </td>
      <td>Select the color channels to render. The options are <b>Red</b>, <b>Green</b>, <b>Blue</b>, and <b>Alpha</b> (transparency).</td>
      <td>
        <img src="../images/textures-pane/color-channel.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/transparency%402x.png" alt="Background"/>
      </td>
      <td>Select a checkerboard pattern or a solid color for the image background.</td>
      <td>
        <img src="../images/textures-pane/image-background.png" width="500px"/>
      </td> 
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/flip_vertically%402x.png" alt="Flip Vertically"/>
      </td>
      <td>Flips the image vertically.</td>
      <td>
        <img src="../images/textures-pane/flip-vertical.png" width="500px"/>
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
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/jump%402x.png" alt="Accesses / Modifications"/>
      </td>
      <td>Displays the list of all calls that updated the texture to this point. Select a call to view the image after the call completes; the selected frame thumbnail and the pane will update accordingly.</td>
      <td>
        <img src="../images/textures-pane/jump-to-reference.png" width="500px"/>
      </td> 
    </tr>
  </tbody>
</table>
