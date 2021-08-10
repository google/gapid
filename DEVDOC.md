# Android GPU Inspector Developer Documentation

## Build instructions

See [BUILDING.md](BUILDING.md).

## Setup to run presubmit tests locally

Before creating a pull-request, check that your code can compile and that the
presubmit tests pass.

To be able to run the presubmit tests locally, install the following:

```
# Buildifier
go get github.com/bazelbuild/buildtools/buildifier

# Buildozer
go get github.com/bazelbuild/buildtools/buildozer

# Clang format 6.0
## On Debian-based Linux (see https://releases.llvm.org/download.html for binaries)
apt-get install clang-format-6.0
## Make sure to set the CLANG_FORMAT environment variable, e.g. in bash:
export CLANG_FORMAT=clang-format-6.0
```

With the above setup, you can run presubmit tests locally with:

```
./kokoro/presubmit/presubmit.sh
```

## Setup Go development

This project contains Go code, but it does not have the file hierarchy of
regular Go projects (this is due to the use of Bazel as a build system).
The `cmd/gofuse` utility enables to re-create the file hierarchy expected by Go
tools. Note, however, `cmd/gofuse` is not supported on Windows, but it is not
required to build/develop AGI either. The steps described here are optional and
are only intended to facilitate working on the AGI codebase using an IDE or
other Go tooling.

```sh
# Make sure to build to have all compile-time generated files
cd <path-to-agi-source>
bazel build pkg

# Prepare a agi-gofuse directory **outside of the AGI checkout directory**
mkdir <path-outside-agi-source>/agi-gofuse

# Run gofuse with the previous directory as a target
bazel run //cmd/gofuse -- -dir <path-to-agi-gofuse>

# If the previous command fails to correctly guess the sub-directory under `bazel-out`,
# please pass it with the command below. For example if you build on Windows, the
# standard <bazelout-subdir> is `x64_windows-fastbuild`. If you build on Linux with
# `bazel build -c dbg pkg`, the <bazelout-subdir> is `k8-dbg`.
bazel run //cmd/gofuse -- -dir <path-to-agi-gofuse> -bazelout <bazelout-subdir>

# Build the package again to output the original compile-time generated files again.
bazel build pkg

# Add agi-gofuse directory to your GOPATH environment variable.
# On Linux, with a bash shell, you can add the following to your ~/.bashrc file:
export GOPATH="${GOPATH:+${GOPATH}:}<path-to-agi-gofuse>"
# On other configurations, please search online how to add/edit environment variables.
```

After adding the gofuse directory to your GOPATH, Go tools should work as
expected. You can edit files under the newly populated gofuse directory. You
should still compile under the original AGI checkout directory.

> Despite its name, the gofuse command does NOT use FUSE (filesystem in userspace).
> It just creates directories and links to source files, including generated files.
> It is a good idea to re-run gofuse from time to time, to re-sync links to potential
> new files.

In terms of editor, [VsCode](https://code.visualstudio.com/) has good Go support
thanks to its
[Go extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode.Go).
With the GOPATH setup to gofuse and opening the `<path-to-agi-gofuse>` directory,
as the root of your workspace, you should get some jump-to-definition and autocomplete
features working. Make sure to edit the files through their link found under the gofuse directory.

## How to debug / breakpoint in Go code

The recommended Go debugger is
[delve](https://github.com/go-delve/delve). You can start a **debug** build of
gapis or a client under this debugger. To build in debug mode, use the `-c dbg`
Bazel flag, e.g.:

```
bazel build -c dbg pkg
```

### Debugging GAPIS

To debug gapis, you can do:

```
dlv exec ./bazel-bin/pkg/gapis -- -enable-local-files -persist -rpc localhost:8888
```

You can then use dlv commands to add breakpoints, and actually start GAPIS, e.g.:

```
(dlv) break gapis/server/server.go:228

(dlv) continue  # this actually starts gapis
```

> See delve documentation on how to specify a breakpoint location, there are
> more convenient alternatives than `path/to/file:line`

Once gapis is started, you can run a client to interact with it and hit somes
breakpoints:

```
# in another terminal
./bazel-bin/pkg/gapit <verb> -gapis-port 8888 <verb args>
```

### Debugging a client

If you want to debug a client like gapit, just start it under dlv:

```
dlv exec ./bazel-bin/pkg/gapit <verb> <verb args>
```

### Debugging a test

You can build a test in debug mode and then start it under dlv.

For instance:

```
bazel test --cache_test_results=no -c dbg //core/data/slice:go_default_test
```

In practice, on Linux this builds and runs a test executable that is located at
`./bazel-out/k8-dbg/bin/core/data/slice/linux_amd64_debug/go_default_test`,
you will have to adapt the `k8-dbg` part will be different on other platforms.
To debug a specific test, you can start this executable under dlv:

```
$ dlv exec ./bazel-out/k8-dbg/bin/core/data/slice/linux_amd64_debug/go_default_test -- -test.run TestReplace
Type 'help' for list of commands.
(dlv) break TestReplace
Breakpoint 1 set at 0x5f5d4b for github.com/google/gapid/core/data/slice_test.TestReplace() core/data/slice/slice_test.go:26
```

### Use a Delve init script

To automate a delve startup sequence, you can edit a script of delve commands to
be executed when delve starts. The script looks like:

```
# This is a comment.
break gapis/server/server.go:228

# add a second breakpoint, with a condition for it to trigger
break gapis/foo/bar.go:123
condition 2 some_variable == 42

# launch program
continue
```

And you can pass this script to delve using the `--init` flag:

```
dlv exec --init my-delve-init-script.txt <program to debug...>
```

### Integration with an IDE

If you want to interact with the debugger via your editor or IDE, be aware that
delve will think file paths start from the AGI top directory, and not your
root directory. This is very likely due to Bazel compilation. You may have to
find workarounds if you call delve from an editor/IDE which consider the file
paths to start from another directory, typically your root directory. There may
be a way to adjust using GOPATH to tell to your IDE a possible root for filename
lookups.

See the workaround for VSCode below, any help to fix it for other IDEs is very welcome!

#### Integration with VSCode and Delve

Follow these steps to use the delve debugger for Go with VSCode to debug `gapis`.

1. Make sure to complete the Go setup above for AGI.

2. Settings file: There are two settings file(`settings.json`) that can be written.
	- Global one that applies to all projects that can be opened with `Ctrl + Shift + P` and `Preferences: Open Settings (JSON)`.
	Add this line to ensure that you have a stable tools directory:
	`"go.toolsGopath": "<path-to-go-plugin-tools-folder>",`

	- Local one is under `.vscode` folder in your project folder. Create one if it does not already exist and add this line to your local settings to be able to search source code in AGI:
	`"go.gopath": "<path-to-agi-gofuse>"`,

3. Launch file: Create a `launch.json` file under the workspace directory with `Ctrl + Shift + P` and `Debug: Open launch.json`

4. Paste the following as one of the launch configurations. This will ensure that there is a launch configuration for attaching to Delve.
```
{
    ...
    "configurations": [
        ...,
        {
            "name": "Attach to Delve",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "apiVersion": 2,
            "remotePath": "",
            "cwd": "`<path-to-agi-gofuse>`",
            "dlvLoadConfig": {
                "followPointers": true,
                "maxVariableRecurse": 1,
                "maxStringLen": 120,
                "maxArrayValues": 120,
                "maxStructFields": -1
            },
            "host": <host>,
            "port": <port>,
        },
    ],
    ...
}
```
As an example, `<host>` could be `127.0.0.1` and `<port>` could be `1234`.

5. Start delve in headless mode in the AGI root folder.
```
dlv exec --headless --listen=<host>:<port> --api-version 2 ./bazel-bin/pkg/gapis -- <gapis-arguments>
```

The command below will allow using port `1234` (or any other preferred port) to connect to delve from VSCode.
```
dlv exec --headless --listen=127.0.0.1:1234 --api-version 2 ./bazel-bin/pkg/gapis -- -persist -rpc localhost:8888
```

6. Start debugging with `Debug->Start Debugging` (on Linux with `F5`) and make sure `Attach to Delve` is selected as the launch configuration.

7. Now VSCode can interact with Delve and can be used for debugging `gapis` in VSCode UI instead of the command line. Enjoy your debugging :)

This allows you to put breakpoint at any line in AGI Go source code regardless if they are handwritten, generated or in Go Standard Library. For the generated file, you can put the breakpoints under the `bazel-bin/` or `bazel-out/` folder and the debugger will still find it under the fuse directory during debugging and open it. The only downside is you will have two versions of the same file but this is the only working workaround until Go Plugin supports bazel-generated files.

## How to debug via printing message

You can use the built-in logging functions to place debug prints.

In Go:

```go
import (
	// ...
	"github.com/google/gapid/core/log"
)

// ...
	log.E(ctx, "Here debug print, myVar: %v", myVar)
```

In C++:

```c++
#include "core/cc/log.h"

// ...
    GAPID_ERROR("Here debug print, myStr: %s", myStr)
```

The usual logging levels are available, listed for instance with `gapit -fullhelp`:

```
$ ./bazel-bin/pkg/gapit -fullhelp
...
-log-level value
	The severity to enable logs at [one of: "Verbose", "Debug", "Info", "Warning", "Error", "Fatal"] (default Info)
```

The Error level is recommended when adding debug print, to make sure it is not
filtered away.

## How to debug a replay crash

The replayer uses [breakpad](https://chromium.googlesource.com/breakpad/breakpad) to catch and optionnaly report crashes.

If you want to analyze a replay crash with a debugger, on **64 bits Android** (for other platforms, adapt as necessary):

1. Start the replayer with the `--wait-for-debugger` flag, e.g. `./gapit screenshot --gapir-args '--wait-for-debugger' mytrace.gfxtrace`.

2. Wait for the replayer to launch on the device.

3. Attach your debugger to the replayer app (`com.google.android.gapid.arm64v8a/com.google.android.gapid.ReplayerActivity`).

4. Once attached, you probably want to add a breakpoint at `CrashHandler::handleMinidump` in order to break upon a crash, before it is reported to the server.

5. When you attach, the replayer is spin-waiting in the loop defined in `core/cc/android/debugger.cpp`. To break this loop, use the debugger to set `gIsDebuggerAttached = true` before continuing execution.

Note that while you attach the debugger and setup the breakpoint, the server might timeout waiting for a gRPC connection. You may increase this timeout by editing the `gapir/client:gRPCConnectTimeout` constant.

## GAPIS build-time options to help with debugging

See `gapis/config/config.go` for a list of various build-time config options
that can help with debugging.

## How to profile AGI internals

### GAPIS

The server has instrumentation to output profiling information:

```
$ ./agi/gapis --fullhelp
[...]
  -profile-pprof
	enable pprof profiling
  -profile-trace string
	write a trace to file
[...]
```

The `-profile-trace` option generates a trace that can be open in Chrome via
`chrome://tracing`, or using [Perfetto web UI](https://ui.perfetto.dev/).

The `gapit` and `agi` (UI starter) clients can pass these arguments to gapis via
`-gapis-args`, e.g.:

```
# GAPIT
./agi/gapit <verb> -gapis-args '-profile-trace my-profile-trace.out' <verb arguments>

# GAPIC
./agi/agi -gapis-args '-profile-trace my-profile-trace.out' foobar.gfxtrace
```

### On-device: GAPII, GAPIR

To profile the interceptor GAPII and the replayer GAPIR on Android devices, you
can resort to classic profiling solutions. Check the Android [system tracing
overview](https://developer.android.com/topic/performance/tracing) doc. Beside
systrace, perfetto and the Android studio profile, also note that
[Inferno](https://android.googlesource.com/platform/system/extras/+/master/simpleperf/doc/inferno.md)
makes it easy to get flamegraphs.

To profile the replayer using AGI itself, you can export a given replay into a
standalone APK (using `gapit export_replay -apk`), and profile that replay-APK
using AGI.

## Unit tests

Unit testing is achieved with separate frameworks depending on the programming
language, but they are all accessible with Bazel.

### List tests

```sh
# List all tests
bazel query 'tests(//...)'

# List all Go tests
bazel query 'kind(go_.*, tests(//...))'
# List all C++ tests
bazel query 'kind(cc_.*, tests(//...))'
```

### Run tests

```sh
# Run all the tests
bazel test tests

# Run a given test
bazel test //core/log:go_default_test
```

### Go

Following the [regular Go test](https://golang.org/doc/code.html#Testing) setup,
tests are written as `func TestXXX(t *testing.T)` functions in `*_test.go`
files.

Adding a Go test file into Bazel is done by invoking Gazelle. The
`kokoro/presubmit/presubmit.sh` script does that for you: if you create or
remove `*_test.go` files, running the presubmit script automatically edit the
BUILD.bazel files to reflect your changes.

A few useful homemade packages:

- `github.com/google/gapid/core/assert` defines an assertion framework.

- `github.com/google/gapid/core/log` lets you create contexts for tests with
  `ctx := log.Testing(t)`

### C++

> TODO

### Java

> TODO

## Code generated at compile time

A large amount of code is generated at compile time. Code is typically generated
via APIC (API Compiler, `cmd/apic/`). APIC takes as input `.api` and `.tmpl`
files, and it generates code in another language.

```

                    +----------+
  <.api files>  --->|          |
                    |   APIC   |---> <generated files>
  <.tmpl files> --->|          |
                    +----------+

```

`.api` files are GAPIL ("graphics API language") sources. GAPIL is a
domain-specific language to specify a graphics API, it is documented in
`gapil/README.md`. AGI currently supports only Vulkan, but its ancestor GAPID
was designed to support arbitrary graphics APIs. Vulkan is described by API files
located under `gapis/api/vulkan/`, the top-level file is
`gapis/api/vulkan/vulkan.api`.

`.tmpl` files are template files. The templating language is documented in
`gapis/api/templates/README.md`. There are various template files in the project
to generate code for the interceptor, server, replayer, Vulkan layers, etc.

APIC is a GAPIL compiler, its entry-point is defined in `cmd/apic/` and the
actual compiler logic is defined in `gapil/`. In a nutshell, APIC parses the
`.api` files to gather information about the graphics API, and then instantiates
the templates in the `.tmpl` files to generate code.

### Where does the generated code end up?

The generated code is not checked under version control. It is generated at
compilation time by Bazel rules calling APIC, resulting in files that can be
seen under e.g. `bazel-bin/`.

For instance, generated C++ code for the interceptor can be found in:

```
user@machine:~/work/agi$ find -L bazel-bin -name '*_spy_*.cpp'
bazel-bin/gapii/cc/vulkan_spy_0.cpp
bazel-bin/gapii/cc/vulkan_spy_subroutines_1.cpp
bazel-bin/gapii/cc/vulkan_spy_subroutines_0.cpp
bazel-bin/gapii/cc/vulkan_spy_2.cpp
bazel-bin/gapii/cc/vulkan_spy_3.cpp
bazel-bin/gapii/cc/vulkan_spy_helpers.cpp
bazel-bin/gapii/cc/vulkan_spy_1.cpp
```

To get a unified file tree view, interleaving source files with generated files,
you can use the `cmd/gofuse` setup. `cmd/gofuse` links the generated source
files in the same directories as their siblings in the same packages/namespaces.

-------------------------------------------------------------------------------

## Life of a gfxtrace

This is an overview of how AGI captures graphics API calls, stores them into a
gfxtrace file, and process that file to replay the calls.

This is oriented to AGI developers who need to work with AGI internals. It is
meant to help build a high-level mental model that facilitates navigating the
actual source code. It references concepts and code modules that are unlikely to
change anytime soon, and it tries not to reference details that are likely to
change. You are encouraged to check the source code while reading this, to
clarify how the various parts actually fit together. If you spot any
discrepancy, please update this doc, but please keep it high-level.

AGI is architectured to be API-agnostic in order to support multiple graphics
APIs. GAPID, from which AGI was forked, supports GLES and Vulkan. AGI's main
focus is Vulkan on Android, so all examples in this doc are related to Vulkan on
Android.

### AGI's overall architecture

AGI is separated in main components that interact mostly via protobuf:

- GAPIS (Graphics API Server, Go code under `gapis/`, see `gapis/README.md`) is
  the main component, running on the developer desktop/laptop. Any complex logic
  is meant to be implemented in GAPIS, while the other components are meant to
  be kept as simple as possible. The protobuf interface of GAPIS is defined in
  `gapis/service/service.proto`.

- GAPII (Graphics API Interceptor, mostly C++ code under `gapii/`, see
  `gapii/README.md`) is the component responsible for intercepting graphics API
  calls during capture. It interacts with GAPIS via a dedicated protocol.

- GAPIR (Graphics API Replayer, mostly C++ code under `gapir/`, see
  `gapir/README.md`) is the component that can replay graphics API calls. Its
  protobuf interface is defined in `gapir/replay_service/service.proto`.

- GAPIC (Graphics API Client, Java code under `gapic/`, see `gapic/README.md`)
  is the GUI client. It uses GAPIS protobuf interface.

- GAPIT (Graphics API Terminal, Go code under `cmd/gapit/`) is the
  developer-oriented CLI client. It uses GAPIS protobuf interface.

When you start AGI via the desktop icon or the `./agi` command, this triggers
the entry point defined under `cmd/agi/`, which by default starts a new GAPIS
and then a new GAPIC that connects to this GAPIS. GAPIS may then itself start
GAPII or GAPIR instances.

To capture and replay on Android, AGI has APKs (one per ABI, e.g.
`gapid-arm64-v8a.apk`) that embeds GAPII and GAPIR.

### Create a gfxtrace: capture, serialize and store Vulkan calls

AGI uses a Vulkan layer to intercept and capture the Vulkan calls emitted by an
application, and stores them into a gfxtrace file. Be aware that in the code
base, the act of intercepting graphics API calls may be referred by the words
"capture", "trace", "intercept" and "spy".

For a Vulkan capture on Android, the main steps are:

1. GAPIS issues a few adb commands to edit global settings that tell the Android
   Vulkan loader to insert AGI's Vulkan layer when starting the app to capture.
   This layer is called `GraphicsSpy` and is implemented by
   `libVkLayer_GraphicsSpy.so` which is mostly a wrapper around `libgapii.so`.

2. The capture layer and GAPIS establish a TCP connection over ADB. The
   `gapii::Spy::Spy()` creator contains the logic to establish this connection.

3. The capture layer monitors every Vulkan API call, and shadows the Vulkan
   state accordingly. The logic of each Vulkan command is implemented in `.api`
   files under `gapis/api/vulkan/`, and code is auto-generated from these files
   (see [Code generated at compile time](#code-generated-at-compile-time)). In
   particular, each command has a `mutate` function that mutates (updates) the
   Vulkan state with respect to the command logic. For instance,
   `vkCreateBuffer()` mutation results in adding a new buffer in the Vulkan
   state.

4. When the user clicks "Start" to start the one-frame capture, GAPIS tells the
   capture layer that it wants the next frame to be captured. The capture layer
   waits for the current frame to end before streaming the whole Vulkan state
   back to GAPIS (see e.g. `VulkanSpy::serializeGPUBuffers`). After that, each
   new intercepted Vulkan command and related memory observations are streamed
   to GAPIS. When the frame terminates, the capture layer sends a special "end"
   message to GAPIS.

5. GAPIS receives the serialized data from the capture layer and stores it into
   a `gfxtrace` file, until it receives the special "end" message.

### What's in a gfxtrace file?

A `gfxtrace` file contains data encoded in
[proto-pack](https://github.com/google/agi/tree/master/core/data/pack/README.md),
a homemade format to encapsulate protobuf messages.

To see the plain content of a gfxtrace, you can use the `./gapit unpack -verbose
myfile.gfxtrace` command. On a simple Vulkan app, this produces the following
(here edited to fit, and with added `#` comments):

```
# Header
Object(msg: Headerᵈ{ABI: ABIᵈ{OS: 4, architecture: 1, ...})
# API state at the start of the capture
BeginGroup(msg: GlobalStateᵈ{...}, id: 24)
Object(msg: Resourceᵈ{index: 1})
  ChildObject(msg: Observationᵈ{pool: 3, res_index: 1}, parentID: 24)
[...]
EndGroup(id: 24)
# List of commands
[...]
# Example of a command: vkQueueSubmit
## The command and its arguments
BeginGroup(msg: vkQueueSubmitᵈ{fence: 3916257488, pSubmits: 3141568848, ...}, id: 599)
## A few memory observations (resource + observation)
Object(msg: Resourceᵈ{data: [4 0 0 0 0 0 0 0 1 0 0 0 208 ...] (truncated 40 bytes), index: 12})
  ChildObject(msg: Observationᵈ{base: 3141568848, res_index: 12, size: 40}, parentID: 599)
Object(msg: Resourceᵈ{data: [240 87 109 233 0 0 0 0 176 110 109 233 0 0 0 0], index: 13})
  ChildObject(msg: Observationᵈ{base: 3838428368, res_index: 13, size: 16}, parentID: 599)
Object(msg: Resourceᵈ{data: [112 38 139 233], index: 14})
  ChildObject(msg: Observationᵈ{base: 3918206552, res_index: 14, size: 4}, parentID: 599)
## The actual call to the driver, and the return value (not being printed here as it is 0 == VK_SUCCESS)
  ChildObject(msg: vkQueueSubmitCallᵈ{}, parentID: 599)
EndGroup(id: 599)
[...]
```

At the proto-pack level, we have `Object`, `ChildObject`, `BeginGroup` and
`EndGroup`. The `msg` fields are the protobuf messages. For instance, the
`Header` protobuf message is defined in `gapis/capture/capture.proto`. The
Vulkan-specific protobuf messages are defined in the `api.proto` file which is
generated from the `.api` files.

We can see the capture header followed by `GlobalState`, the dump of Vulkan
state. Then, the rest of the capture is made of a series of Vulkan calls. Each
call is represented as a protopack group. This example illustrates how a
`vkQueueSubmit` call is encoded.

In general, the call to a graphics API command leads to objects and messages
that represent:

1. The API command and its arguments.

2. Zero or more memory observations that represent memory that the driver may
   read during this command. For instance, many Vulkan commands have pointers to
   struct as arguments: the content of these structures is read by the driver.

3. The actual call to the driver, and its return value (if any).

4. Zero or more memory observations that represent memory that the driver may
   have written to during its processing of the command. For instance, a
   `vkCreateBuffer` call writes the handle of the newly create buffer to the
   memory pointed at by its `VkBuffer* pBuffer` argument.

Note that on multithreaded apps, the object groups may be interleaved.

### GAPIS handling of a gfxtrace

GAPIS parses gfxtrace files and represents them in a `GraphicsCapture` Go
object. This type gives access to the header, the initial state and the list of
commands, among other things. Once a gfxtrace file is loaded inside GAPIS, the
clients can use GAPIS protobuf interface to interact with the capture.

For instance, a client may request the Vulkan state after a certain command. To
obtain this state, GAPIS will use the Go version of the `mutate` functions
(generated from the `.api` files) to mutate the initial state up to the required
command, and return the resulting state. Note that this does not require a
proper replay, as the state mutation happens entirely in GAPIS.

The `.api` files can be seen here as an implementation of a driver for a given
graphics API, as they describe how each command affects the graphics API state.
Thus, it is possible to simulate the evolution of the graphics API state inside
GAPIS. This is what enables GAPIS to provide a snapshot of the state at some
point in the capture without having to do an actual replay.

However, this driver simulation is not complete: the actual effects of draw
calls to their render targets are not implemented. This means that while GAPIS
does not need a replay to say e.g. how many images are in the state at a given
point, it does need a replay to show you the content of such images after some
draw calls. This replay is necessary as the result depends on the actual device
and driver used for replay.

### Replay of a gfxtrace

GAPIR is effectively a stack-based virtual machine specialized for the replay of
graphics API commands, see details in its own [README](gapir/README.md). To
replay a gfxtrace, GAPIS takes a `GraphicsCapture` object and generates a
payload made of GAPIR VM opcodes.

The actual transformation of API commands into replay opcodes is made by the
`mutate` function of each command. These functions take a "replay builder" (see
`gapis/replay/builder` module) as an optional argument. When this builder is not
nil, the `mutate` function uses it to generate the replay opcodes corresponding
to the API command.

When the initial state of a capture is not empty, GAPIS also generates "initial"
commands that are required to reconstruct the initial state from a fresh empty
state. For instance, if the initial state of a Vulkan capture has a Vulkan
instance, GAPIS generates a `vkCreateInstance` command to recreate a similar
instance. These initial commands must be replayed first in order to obtain a
state from wich the capture commands can be called.

#### The command transformation framework

One of the key feature of AGI is being able to replay variations of the original
capture. For instance, in order to see what the framebuffer looks like after a
certain draw call, AGI must make a replay up to the desired draw call, but no
further. In Vulkan, this typically requires to re-write the content of a command
buffer to only include the relevant draw calls.

In order to implement such modifications on the capture, GAPIS has a command
transformation framework. Conceptually, the commands of a capture are streamed
into a chain of "transforms". A transform receives commands, may modify them,
and then pass the resulting commands to the next transform. At the end of the
chain, the obtained commands are mutated with a replay builder to obtain replay
opcodes.

At a high-level, this can be compared to a series of unix pipes:

```
cat commands | transform1 | transform2 | ... | replay_builder > replay_payload
```

To see examples of Vulkan transforms, look at `gapis/api/vulkan/transform_*.go`
files.

### GAPIR executes the replay instructions

GAPIS uses the GAPIR protobuf service (`gapir/replay_service/service.proto`) to
request the replay of a payload. This payload contains replay opcodes, and
references to the required replay resources (i.e. the raw bytes of e.g. buffer
contents).

The resources are not directly embedded in the replay payload: GAPIR lazily
requests them to GAPIS during the replay. On Android, GAPIR runs on the device
and communicates with GAPIS over ADB. Pushing the resources from GAPIS to GAPIR
is very slow, as they must be transferred over ADB. Because a replay may not
require all resources, and also because all resources may not fit in the replay
device memory, AGI refrains from uploading all resources upfront. Instead, GAPIR
maintains a resource cache and has some logic to request batches of resources to
lower the number of requests while keeping the cache full.

During the replay, GAPIR can send back various information to GAPIS. For
instance, it can send back the content of a render target at a given point. It
also regularly sends notifications of how many instructions have been processed
so far, this information is used to reflect the progress of a replay in GAPIC.

#### Split-replay: pre-warm replay with initial commands

In order to speed-up replays, GAPIS requests the initial commands of a capture
to be replayed even before the user asks for a replay.

Depending on the replay request, the capture commands may be transformed to
produce a relevant replay payload. Most of the time, these transformations only
need to be applied on the capture commands, not on the initial commands. Hence,
when waiting for user input, it makes sense to replay the default initial
commands to rebuilt the initial state: once the user requests a specific replay,
only the (transformed) capture commands needs to be replayed.

If the user-requested replay does require to transform the initial commands,
then the pre-warm replay is abandonned and a new replay of the transformed
initial and capture commands is executed.

## Life of a perfetto trace (system profile trace)

AGI heavily relies on the [perfetto](https://perfetto.dev/) system profiling
framework to obtain, store and process profiling data. Hence we often refer to
system profile traces as "perfetto traces". If you have to deal with perfetto
code, make sure to refer to the [perfetto
documentation](https://perfetto.dev/docs/).

Perfetto is built into Android since Android 9 Pie. In general, to take a
perfetto trace, you can use the `perfetto` command-line tool on the device. See
perfetto's web interface at https://ui.perfetto.dev/ and click on "Recording
command" to see an example of how the `perfetto` command-line tool can be used
to obtain a profiling trace.

To take a trace, AGI may use either the `perfetto` command-line tool, or
perfetto's client interface. See e.g.
`gapis/perfetto/android/trace.go:Capture()` for how a capture is started on
Android. To see an example of AGI using perfetto's command line interface, see
`core/os/android/adb/perfetto.go:StartPerfettoTrace()`. Alternatively, AGI may
interact via perfetto's client interface by talking to the `traced` deamon
running on the device. This deamon listens to the `/dev/socket/traced_consumer`,
and AGI connects directly to this socket. The related AGI code is under
`gapis/perfetto/client/`.

One specificity of GPU profiling is that some of the perfetto data producers are
inside GPU drivers, and they need to be started before a trace with these GPU
data sources can be taken. To start these GPU-specific perfetto data producers,
AGI has a small `agi_launch_producer` utility (see source in
`cmd/launch_producer/`). On Android, AGI extracts this utiliy from its own APK
and invokes it, such that the relevant GPU data producers are started (see
`gapidapk/gapidapk.go:EnsurePerfettoProducerLaunched()`).

Once a perfetto trace has been collected, AGI uses perfetto's [trace
processor](https://perfetto.dev/docs/analysis/trace-processor) to retrieve
profiling data using SQL queries.
