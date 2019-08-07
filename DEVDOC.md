# GAPID Developer Documentation

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

Any help to fix this is very welcome!

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
