---
title: Minimum System Requirements
sidebar: System Requirements
layout: default
permalink: /requirements/
order: 60
---

## Windows

* Windows 7 or later.
* OpenGL 4.1 for OpenGL ES replay.
* [Vulkan GPU drivers for desktop trace / replay](https://en.wikipedia.org/wiki/Vulkan_(API)#Compatibility).

## macOS

* El Capitan (10.11) or later.
* OpenGL 4.1 for OpenGL ES replay. [See this table for more device compatibility](https://developer.apple.com/opengl/OpenGL-Capabilities-Tables.pdf).
* Note macOS does not support Vulkan drivers, but you can trace and replay Vulkan applications on a connected Android device.

## Linux

* Ubuntu 'Trusty Tahr' (14.04) or later recommended.
* Java 64-bit JDK or JRE 1.8.
* OpenGL 4.1 for OpenGL ES replay.
* [Vulkan GPU drivers for desktop trace / replay](https://en.wikipedia.org/wiki/Vulkan_(API)#Compatibility).

## Android

* Android M (6.0) or later for OpenGL ES 2.0+ tracing (OpenGL ES 1.x is not supported).
* Android N (7.0) or later for Vulkan tracing.

<div class="issue">
  Please be aware there are known issues tracing on: <br>
  <ul>
    <li> Android x86/x64 emulator.</li>
  </ul>
</div>
