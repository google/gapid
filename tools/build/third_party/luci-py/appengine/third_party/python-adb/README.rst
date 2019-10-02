python-adb
==========

This repository contains a pure-python implementation of the ADB and Fastboot
protocols, using libusb1 for USB communications.

This is a complete replacement and rearchitecture of the Android project's ADB
and fastboot code available at
https://github.com/android/platform_system_core/tree/master/adb

This code is mainly targeted to users that need to communicate with Android
devices in an automated fashion, such as in automated testing. It does not have
a daemon between the client and the device, and therefore does not support
multiple simultaneous commands to the same device. It does support any number of
devices and never communicates with a device that it wasn't intended to, unlike
the Android project's ADB.

Pros:
  * Simpler code due to use of libusb1 and Python.
  * API can be used by other Python code easily.
  * Errors are propagated with tracebacks, helping debug connectivity issues.

Cons:
  * Technically slower due to Python, mitigated by no daemon.
  * Only one command per device at a time.
  * More dependencies than Android's ADB.

Dependencies:
  * libusb1 (1.0.16+)
  * python-gflags (2.0+)
  * python-libusb1 (1.2.0+)
  * python-progressbar (for fastboot_debug, 2.3+)
  * python-m2crypto (0.21.1+)

