# Track Coherent Memory

## Problem and Solution
Coherent memories are implicitly synchronized which means there is no *flush*
or *invalid* command to explicitly *write* or *read* to/from those memories.
Henceforth we cannot rely on intercepting the *flush* and *invalid* commands to
track the modifications for coherent memory.

For Posix-based systems, one of the solutions is to utilize segmentation fault
signal handler to tell to which memory page has been written and then read the
data of that page to track the data in coherent memory.

The following explanation is for Posix-based systems only. We intend to
implement a similar system for Windows with `SetUnhandledExceptionFilter` and
`VirtualProtect`.

## Goal
The goal is to implement a memory tracker that:
* Tracks the memory pages that have been written to.
* Transparent to the application under debugging.
 * Assumes zero modification on the application under debugging.
 * For any memory that is not tracked, the previously installed handler should
 be called.
 * Supports multi-threaded context.

## Current Support
The current support status of the memory tracker based on operating systems and
graphics APIs of the tracing target.

|OS        |OpenGL              |Vulkan              |
|----------|--------------------|--------------------|
|Windows   |:white_medium_square:|:white_medium_square:|
|Linux     |:white_medium_square:|:white_check_mark:  |
|Android   |:white_medium_square:|:white_check_mark:  |


## Overall Workflow
* Start Tracing.
* Coherent memory `M` is allocated and mapped.
 * Register the memory tracker's segfault handler for the first time a coherent
 memory is mapped.
* Add the mapped memory range of `M` to the memory tracker, and set the access
permission of all the pages in mapped `M` as read-only.
* Application write data to the coherent memory `M`.
* Segmentation fault signal is triggered on a page `P` of mapped memory `M`.
* Memory tracker's segfault handler records that `P` has been written to, and
set the access permission of `P` to read-write.
* Application calls a graphics command that may refer the coherent memory.
* The debugger gets `P` and read its content.
* Once the coherent memory is unmapped and deallocated, remove the memory `M`
from the memory tracker.

## Components
The memory tracker is consist of three components:
* Signal Handler
  * Check if the segfault address is from coherent memories.
  * If so, records the page's root address to the Dirty Page Table, then set
  the page's permission back to readable and writable.
  * Otherwise, falls back to the system segfault handler or the handler
  registered by the application.
* Dirty Page Table
 * Contains the root addresses of the pages that have been written to.
 * Preallocate its space. Signal Handler is not allowed to
 call `malloc()`.
* Coherent Memory Table
 * Bookkeeping for the coherent memories that should be tracked.
 * Sets the mapped memory pages of the added coherent memories to read-only.
 * Readable to Signal Handler.
 * Writeable to the debugger, or say, the application threads.
