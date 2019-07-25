---
layout: default
title: How do I create a video of my trace?
permalink: /tutorials/createvideo
---

GAPID has the ability to export a captured trace as a video.

This feature is only available via the command line tool [`gapit`](../cli).

### Prerequisites
* A previously captured trace file ending in *.gfxtrace
* Either avconv or ffmpeg encoding libraries on your path (for regular video encoding)

1. Launch the command line interface.

2. Note the path in which gapid was installed.
<span class="info">e.g. `C:\Program Files (x86)\gapid\` on Windows; `/Applications/GAPID.app/Contents/MacOS/` on Mac OS X; or `/opt/gapid/` on linux.</span>

3. Run the following command
```
$ <gapid_install_path>gapit video -out <output_path>.mp4 <path_to_tracefile>
```
  a. If you wish to instead output all frames as individual images you may instead run
```
$ <gapid_install_path>gapit video -type frames -out <output_path>.png <path_to_tracefile>
```
