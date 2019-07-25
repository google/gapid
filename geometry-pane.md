---
layout: default
title: Geometry Pane
permalink: /inspect/geometry
---

<img src="../images/geometry-pane.png" width="727px" height="445px"/>

The geometry pane renders the pre-transformed mesh of the selected draw call. You can use your mouse or touchpad to rotate the model, and zoom in and out.

The following table describes the operations you can perform with the toolbar buttons:


<table>
  <tbody>
    <tr>
      <th width="20%">Button</th>
      <th>Description</th>
      <th>Example</th>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/yup%402x.png" alt="Y-up"/>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/zup%402x.png" alt="Z-up"/>
      </td>
      <td>
        Click the button to toggle between y axis up and z axis up. In OpenGL ES, the default is the y axis pointing up, the x axis horizontal, and the z axis as depth.
      </td>
      <td><img src="../images/geometry-zup.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn"  src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/winding_cw%402x.png" alt="Winding CW"/>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/winding_ccw%402x.png" alt="Winding CCW"/>
      </td>
      <td>
        Toggle between counterclockwise and clockwise triangle winding to view front- versus back-facing triangles.
      </td>
      <td><img src="../images/geometry-winding.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/wireframe_none%402x.png" alt="Shaded"/>
      </td>
      <td>
        Show the geometry rendered as shaded polygons.
      </td>
      <td><img src="../images/geometry-shaded.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/wireframe_all%402x.png" alt="Wireframe"/>
      </td>
      <td>
        Show the geometry rendered as a wireframe.
      </td>
      <td><img src="../images/geometry-wireframe.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/point_cloud%402x.png" alt="Point Cloud"/>
      </td>
      <td>
        Show the geometry rendered as vertex data points.
      </td>
      <td><img src="../images/geometry-points.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/smooth%402x.png" alt="Authored Normals"/>
      </td>
      <td>
        Select this button to display smooth normals as specified in your code. The button is unavailable if you haven't authored normals in your mesh.
      </td>
      <td><img src="../images/geometry-shaded.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/faceted%402x.png" alt="Faceted Normals"/>
      </td>
      <td>
        Select this button to see the lit geometry without using smooth normals. It renders the geometry as if each polygon were flat instead of smoothed, using computed face normals.
      </td>
      <td><img src="../images/geometry-faceted.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/culling_disabled%402x.png" alt="Backface Culling Disabled"/>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/culling_enabled%402x.png" alt="Backface Culling Enabled"/>
      </td>
      <td>
        Click this button to toggle backface culling, which when enabled hides polygons facing away from the camera.
      </td>
      <td><img src="../images/geometry-backface-cull.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/lit%402x.png" alt="Lit"/>
      </td>
      <td>
        Select this button to render the mesh with a simple directional light.
      </td>
      <td><img src="../images/geometry-shaded.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/flat%402x.png" alt="Flat"/>
      </td>
      <td>
        Select this button to render the mesh with just ambient light.
      </td>
      <td><img src="../images/geometry-ambient.png" width="130px" height="79px"/></td>
    </tr>
    <tr>
      <td>
        <img class="toolbar-btn" src="https://raw.githubusercontent.com/google/gapid/master/gapic/res/icons/normals%402x.png" alt="Normals"/>
      </td>
      <td>
        Select this button to view normals. Red indicates a positive x axis value, green a positive y axis value, and blue a positive z axis value.
      </td>
      <td><img src="../images/geometry-normals.png" width="130px" height="79px"/></td>
    </tr>
  </tbody>
</table>
