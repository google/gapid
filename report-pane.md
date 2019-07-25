---
layout: default
title: Report Pane
permalink: /inspect/report
---

<img src="../images/report-pane.png" width="386" height="297"/>

The report pane shows any issues found with the capture and its replay.
This includes issues caused by incorrect parameter usage, invalid command sequences or errors reported by the driver used for replay.

If you are using GAPID to diagnose incorrect rendering, check the report pane for any issues.

## Note for Vulkan

The report pane currently shows very few messages for Vulkan. This will be expanded upon for a future release. Furthermore, given Vulkan's explicit and low-level nature, replaying a Vulkan trace on a target that is different than the tracing target is unlikely to work for any but the most straight-forward traces.
