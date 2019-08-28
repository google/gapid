---
layout: default
title: Memory Pane
permalink: /inspect/memory
---


The **Memory** pane displays the values in RAM or GPU memory for the selected command.

<img src="../images/memory-pane/memory-pane.png" width="600px" />

This pane shows which memory locations were read from and/or written to by the selected command. Each command typically has multiple read or write operations; select one from the **Range** list. The view updates to show the starting memory address for the operation. Green denotes a read operation while red denotes a write operation. For example, the command in the image above contained a read operation of 64 bytes starting at memory address 0x000000728185be58. You can change how the data is displayed by selecting a different data type from the **Type** list.

The **Pool** field is set to **0** for displaying values corresponding to application memory. If the **Pool** is set to any other number, the pane shows values for GPU-assigned memory. Application memory uses RAM while GPU-assigned memory may use RAM or GPU memory (this selection is opaque to the user).

Click a pointer value in the **Commands** pane to jump directly to that specific address in the **Memory** pane. 

You aren’t limited to viewing specific address ranges in this pane. Select a command and then the **State** pane. Select **DeviceMemories**. (This section is organized by Vulkan handle for Vulkan traces.) Expand a handle and select **Data**. Click a specific address to display it in the view.

<img src="../images/memory-pane/memory-state.png" width="800px" />