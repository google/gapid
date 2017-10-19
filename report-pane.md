---
layout: default
title: Report Pane
sidebar: Report Pane
permalink: /inspect/report
parent: inspect
---

<img src="../images/report-pane.png" width="558px"/>

One of GAPID's core goals is to support replays on targets different to those used to create the capture. For example, we may want to capture an OpenGL ES 2.0 application, and replay the capture using a desktop OpenGL 4.0 context. The Report pane will show any compatibility errors that occur in this conversion. If your resulting replay has errors, take a look at the Report to see if there is anything has been highlighted there.

## Note for Vulkan

The report pane currently shows very few messages for Vulkan. This will be expanded upon for a future release. Furthermore, given Vulkan's explicit and low-level nature, replaying a Vulkan trace on a target that is different than the tracing target is unlikely to work for any but the most straight-forward traces.
