# GAPID Developer Documentation

## Setup Golang development

This project contains Golang code, but it does not have the file hierarchy of
regular Golang projects (this is due to the use of Bazel as a build system).
The `cmd/gofuse` utility enables to re-create the file hierarchy expected by Go
tools:

```sh
# Make sure to build to have all compile-time generated files
cd <path-to-gapid-source>
bazel build pkg

# Prepare a gapid-gofuse directory **outside of the gapid checkout directory**
mkdir <path-outside-gapid-source>/gapid-gofuse

# Run gofuse with the previous directory as a target
bazel run //cmd/gofuse -- -dir <path-to-gapid-gofuse>

# Add gapid-gofuse directory to your GOPATH environment variable.
# On Linux, with a bash shell, you can add the following to your ~/.bashrc file:
export GOPATH="${GOPATH:+${GOPATH}:}<path-to-gapid-gofuse>"
# On other configurations, please search online how to add/edit environment variables.
```

If you encounter a symlink error on Windows like 'a required privilege is not held by the client',
you have to use a command prompt with administrator privileges or enable
[Developer Mode](https://docs.microsoft.com/en-us/windows/uwp/get-started/enable-your-device-for-development)
as described [here](https://blogs.windows.com/windowsdeveloper/2016/12/02/symlinks-windows-10/).

After adding the gofuse directory to your GOPATH, Go tools should work as
expected. You can edit files under the newly populated gofuse directory. You
should still compile under the original checkout directory of GAPID.

> Despite its name, the gofuse command does NOT use FUSE (filesystem in userspace).
> It just creates directories and links to source files, including generated files.
> It is a good idea to re-run gofuse from time to time, to re-sync links to potential
> new files.

In terms of editor, [VsCode](https://code.visualstudio.com/) has good Go support
thanks to its
[Go extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode.Go).
With the GOPATH setup to gofuse and opening the `<path-to-gapid-gofuse>` directory,
as the root of your workspace, you should get some jump-to-definition and autocomplete
features working. Make sure to edit the files through their link found under the gofuse directory.

## How to debug / breakpoint in Golang code

The recommended Golang debugger is
[delve](https://github.com/go-delve/delve). You can start a debug build of gapis
or a client under this debugger.

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
delve will think file paths start from the gapid top directory, and not your
root directory. This is very likely due to Bazel compilation. You may have to
find workarounds if you call delve from an editor/IDE which consider the file
paths to start from another directory, typically your root directory. There may
be a way to adjust using GOPATH to tell to your IDE a possible root for filename
lookups.

See the workaround for VSCode below, any help to fix it for other IDEs is very welcome!

#### Integration with VSCode and Delve

To use the delve debugger for Go with VSCode to debug `gapis`. These steps can be followed:

1. Make sure to complete Golang Setup for GAPID.

2. Create a `launch.json` under the workspace directory with `Ctrl + Shift + P` and `Debug: Open launch.json`

3. Paste this as one of the launch configurations. This will ensure that there is a launch configuration for attaching to Delve.
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
            "remotePath": "gapis/",
            "cwd": "${workspaceFolder}/src/github.com/google/gapid/gapis",
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

4. Start delve in headless mode at gapid check-in folder.
```
dlv exec --headless --listen=<host>:<port> --api-version 2 ./bazel-bin/pkg/gapis -- <gapis-arguments>
```

The command below will allow using port `1234` (or any other preferred port) to connect to delve from VSCode.
```
dlv exec --headless --listen=127.0.0.1:1234 --api-version 2 ./bazel-bin/pkg/gapis -- -persist -rpc localhost:8888
```

5. Start debugging with `Debug->Start Debugging` (on Linux with `F5`) and make sure `Attach to Delve` is selected as the launch configuration.

6. Now VSCode can interact with Delve and can be used for debugging `gapis` in VSCode UI instead of command line. Enjoy your debugging :)

## How to debug via printing message

You can use the built-in logging functions to place debug prints.

In Golang:

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
