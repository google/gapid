# Internal Build System

This directory contains scripts used by our internal continuous build system.
Please follow these instructions [to build](../BUILDING.md) the project.

# Package Layouts

The platform specific build files contained in the sub directories create
distributable binary packages that follow the format and layout described
below.

## Windows

A `.zip` file is built containing the application, dependant DLLs and the
JRE. The file structure within the `.zip` archive is:

```
gapid\
├─ jre\...
├─ lib\
│  ├─ GraphicsSpyLayer.json
│  ├─ gapic.jar
│  ├─ libgapii.dll
│  ├─ libVkLayer_VirtualSwapchain.dll
│  └─ VirtualSwapchainLayer.json
├─ strings\en-us.stb
├─ build.properties
├─ gapid-arm64-v8a.apk
├─ gapid-armeabi-v7a.apk
├─ gapid-x86.apk
├─ gapid.bat (shell script to launch the client)
├─ gapir.exe
├─ gapis.exe
├─ gapit.exe
├─ libgcc_s_seh-1.dll
├─ libstdc++-6.dll
└─ libwinpthread-1.dll
```

Plans are in place to also build an executable installer. The installer will
allows the user to install the package anywhere, suggesting
`%PROGRAMFILES%\gapid`. The installer will install GAPID into the selected
folder with the same structure as mentioned aboe.

## MacOS

The built package consists of a `.dmg` disk image containing the `GAPID.app`
application and link to `/Applications` to make "installing" easy. The
`GAPID.app` can be run from anywhere and follows the following structure:

```
GAPID.app/
└─ Contents/
   ├─ MacOS/
   │  └─ <see below>
   ├─ Resources/GAPID.icns
   └─ Info.plist
```

Along with the `.dmg` disk image, a `.zip` file with the same content's as
inside the `Contents/MacOS/` folder of the `.app` package is built:

```
<Content/MacOS inside .app or gapid inside .zip>/
├─ jre/...
├─ lib/
│  ├─ gapic.jar
│  └─ libgapii.dylib
├─ strings/en-us.stb
├─ build.properties
├─ gapid-arm64-v8a.apk
├─ gapid-armeabi-v7a.apk
├─ gapid-x86.apk
├─ gapid (shell script to launch the client)
├─ gapir
├─ gapis
└─ gapit
```

When running the `.app` application, the `gapid` script is invoked. When
using the `.zip` file, the only way to run the application is via the
terminal by running the `gapid` script or the executables.

## Linux

A Debian `.deb` package and a `.zip` file with the same contents are built.
The package installs into `/opt/gapid`, while archives can be expanded anywhere.
The Debian package depends on `openjdk-8-jre` and neither it nor the `.zip`
archives contain the JRE. The launcher script looks for `java` first on the
`$PATH`, then `$JAVA_HOME` and finally the hard-coded
`/usr/lib/jvm/java-8-openjdk-amd64/jre/bin/java`, which the `openjdk-8-jre`
package provides. The file layout is:

```
/{opt|wherever}/gapid/
├─ lib/
│  ├─ gapic.jar
│  ├─ GraphicsSpyLayer.json
│  ├─ libgapii.so
│  ├─ libVkLayer_VirtualSwapchain.so
│  └─ VirtualSwapchainLayer.json
├─ strings/en-us.stb
├─ build.properties
├─ gapid-arm64-v8a.apk
├─ gapid-armeabi-v7a.apk
├─ gapid-x86.apk
├─ gapid (shell script to launch the client)
├─ gapir
├─ gapis
└─ gapit
```
